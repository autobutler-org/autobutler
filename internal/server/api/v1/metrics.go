package v1

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"autobutler/pkg/botel/exporters/botelsqlite"
	"autobutler/pkg/db"

	"github.com/gin-gonic/gin"
)

// QueryRangeResponse represents the Prometheus-compatible response format
type QueryRangeResponse struct {
	Status string                 `json:"status"`
	Data   QueryRangeResponseData `json:"data"`
}

type QueryRangeResponseData struct {
	ResultType string        `json:"resultType"`
	Result     []QueryResult `json:"result"`
}

type QueryResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]any           `json:"values"`
}

func SetupMetricsRoutes(router *gin.RouterGroup, metricsExporter *botelsqlite.TraceExporter) {
	router.GET("/metrics", newMetricsHandler(metricsExporter))
	router.GET("/metrics/query_range", handleQueryRange)
	router.GET("/metrics/query", handleInstantQuery)
}

func newMetricsHandler(metricsExporter *botelsqlite.TraceExporter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if metricsExporter == nil {
			c.String(http.StatusServiceUnavailable, "# Metrics exporter not initialized\n")
			return
		}

		metrics, err := metricsExporter.PrometheusMetrics(c.Request.Context())
		if err != nil {
			c.String(http.StatusInternalServerError, "# Error generating metrics: %s\n", err.Error())
			return
		}

		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, metrics)
	}
}

func handleQueryRange(c *gin.Context) {
	query := c.Query("query")
	startStr := c.Query("start")
	endStr := c.Query("end")
	stepStr := c.Query("step")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	// Parse timestamps
	start, err := parseTimestamp(startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start timestamp"})
		return
	}

	end, err := parseTimestamp(endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end timestamp"})
		return
	}

	step, err := strconv.ParseInt(stepStr, 10, 64)
	if err != nil || step <= 0 {
		step = 60 // Default to 60 seconds
	}

	// Execute query
	results, err := executeRangeQuery(c.Request.Context(), query, start, end, step)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := QueryRangeResponse{
		Status: "success",
		Data: QueryRangeResponseData{
			ResultType: "matrix",
			Result:     results,
		},
	}

	c.JSON(http.StatusOK, response)
}

func handleInstantQuery(c *gin.Context) {
	query := c.Query("query")
	timeStr := c.Query("time")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	var queryTime time.Time
	if timeStr != "" {
		ts, err := parseTimestamp(timeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time timestamp"})
			return
		}
		queryTime = time.Unix(ts, 0)
	} else {
		queryTime = time.Now()
	}

	// Execute instant query (just get the latest value)
	results, err := executeInstantQuery(c.Request.Context(), query, queryTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := QueryRangeResponse{
		Status: "success",
		Data: QueryRangeResponseData{
			ResultType: "vector",
			Result:     results,
		},
	}

	c.JSON(http.StatusOK, response)
}

func parseTimestamp(ts string) (int64, error) {
	// Try parsing as float (unix timestamp with decimals)
	if strings.Contains(ts, ".") {
		f, err := strconv.ParseFloat(ts, 64)
		if err != nil {
			return 0, err
		}
		return int64(f), nil
	}

	// Try parsing as int
	return strconv.ParseInt(ts, 10, 64)
}

func executeRangeQuery(ctx context.Context, query string, start, end, step int64) ([]QueryResult, error) {
	// Parse the query to extract metric name and aggregation
	metricName, aggregation, groupBy, labelFilters := parsePromQLQuery(query)

	if metricName == "" {
		return nil, fmt.Errorf("could not parse metric name from query")
	}

	// Convert timestamps to nanoseconds
	startNano := start * 1e9
	endNano := end * 1e9

	// Build label filter SQL
	labelFilterSQL := ""
	labelFilterArgs := []any{}
	if len(labelFilters) > 0 {
		for key, filter := range labelFilters {
			if filter.operator == "=~" {
				labelFilterSQL += ` AND EXISTS (
					SELECT 1 FROM metric_attributes ma2
					WHERE ma2.metric_id = m.id
					AND ma2.key = ?
					AND ma2.value LIKE ?
				)`
				labelFilterArgs = append(labelFilterArgs, key, filter.value)
			} else if filter.operator == "!~" {
				labelFilterSQL += ` AND NOT EXISTS (
					SELECT 1 FROM metric_attributes ma2
					WHERE ma2.metric_id = m.id
					AND ma2.key = ?
					AND ma2.value LIKE ?
				)`
				labelFilterArgs = append(labelFilterArgs, key, filter.value)
			}
		}
	}

	// Build SQL query based on aggregation
	var sqlQuery string
	var args []any

	if aggregation != "" && groupBy != "" {
		// Aggregated query with grouping
		sqlQuery = `
			SELECT
				ma.value as label_value,
				m.timestamp / 1000000000 as ts,
				` + getAggregationSQL(aggregation) + `(m.value) as value
			FROM metrics m
			LEFT JOIN metric_attributes ma ON m.id = ma.metric_id AND ma.key = ?
			WHERE m.name = ?
				AND m.timestamp >= ?
				AND m.timestamp <= ?
				` + labelFilterSQL + `
			GROUP BY label_value, ts / (? * 1000000000)
			ORDER BY ts
		`
		args = append([]any{groupBy, metricName, startNano, endNano}, labelFilterArgs...)
		args = append(args, step)
	} else if aggregation != "" {
		// Aggregated query without grouping
		sqlQuery = `
			SELECT
				m.timestamp / 1000000000 as ts,
				` + getAggregationSQL(aggregation) + `(m.value) as value
			FROM metrics m
			WHERE m.name = ?
				AND m.timestamp >= ?
				AND m.timestamp <= ?
			GROUP BY ts / (? * 1000000000)
			ORDER BY ts
		`
		args = []any{metricName, startNano, endNano, step}
	} else {
		// Simple query - get all data points grouped by attributes
		sqlQuery = `
			WITH metric_labels AS (
				SELECT
					m.id,
					m.timestamp / 1000000000 as ts,
					m.value,
					GROUP_CONCAT(ma.key || '=' || ma.value, ',') as labels
				FROM metrics m
				LEFT JOIN metric_attributes ma ON m.id = ma.metric_id
				WHERE m.name = ?
					AND m.timestamp >= ?
					AND m.timestamp <= ?
				GROUP BY m.id, m.timestamp, m.value
			)
			SELECT labels, ts, value
			FROM metric_labels
			ORDER BY labels, ts
		`
		args = []any{metricName, startNano, endNano}
	}

	rows, err := db.HealthInstance.Db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	// Group results by label set
	resultMap := make(map[string]*QueryResult)

	for rows.Next() {
		var labelValue sql.NullString
		var ts int64
		var value float64

		if aggregation != "" && groupBy != "" {
			if err := rows.Scan(&labelValue, &ts, &value); err != nil {
				return nil, err
			}
		} else if aggregation != "" {
			if err := rows.Scan(&ts, &value); err != nil {
				return nil, err
			}
		} else {
			var labels sql.NullString
			if err := rows.Scan(&labels, &ts, &value); err != nil {
				return nil, err
			}
			if labels.Valid {
				labelValue = labels
			}
		}

		// Create metric labels
		labels := make(map[string]string)
		if groupBy != "" && labelValue.Valid {
			labels[groupBy] = labelValue.String
		} else if labelValue.Valid && labelValue.String != "" {
			// Parse label string
			for _, pair := range strings.Split(labelValue.String, ",") {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					labels[parts[0]] = parts[1]
				}
			}
		}
		labels["__name__"] = metricName

		// Create a key for this label set
		labelKey := formatLabels(labels)

		if _, exists := resultMap[labelKey]; !exists {
			resultMap[labelKey] = &QueryResult{
				Metric: labels,
				Values: [][]any{},
			}
		}

		// Add timestamp and value
		resultMap[labelKey].Values = append(resultMap[labelKey].Values, []any{ts, fmt.Sprintf("%.6f", value)})
	}

	// Convert map to slice
	results := make([]QueryResult, 0, len(resultMap))
	for _, result := range resultMap {
		results = append(results, *result)
	}

	return results, nil
}

func executeInstantQuery(ctx context.Context, query string, queryTime time.Time) ([]QueryResult, error) {
	metricName, aggregation, groupBy, _ := parsePromQLQuery(query)

	if metricName == "" {
		return nil, fmt.Errorf("could not parse metric name from query")
	}

	queryNano := queryTime.UnixNano()

	var sqlQuery string
	var args []any

	if aggregation != "" && groupBy != "" {
		sqlQuery = `
			SELECT
				ma.value as label_value,
				` + getAggregationSQL(aggregation) + `(m.value) as value
			FROM metrics m
			LEFT JOIN metric_attributes ma ON m.id = ma.metric_id AND ma.key = ?
			WHERE m.name = ?
				AND m.timestamp <= ?
			GROUP BY label_value
		`
		args = []any{groupBy, metricName, queryNano}
	} else if aggregation != "" {
		sqlQuery = `
			SELECT ` + getAggregationSQL(aggregation) + `(m.value) as value
			FROM metrics m
			WHERE m.name = ? AND m.timestamp <= ?
		`
		args = []any{metricName, queryNano}
	} else {
		sqlQuery = `
			WITH latest_metrics AS (
				SELECT
					m.id,
					m.value,
					ROW_NUMBER() OVER (PARTITION BY (
						SELECT GROUP_CONCAT(ma2.key || '=' || ma2.value, ',')
						FROM metric_attributes ma2
						WHERE ma2.metric_id = m.id
					) ORDER BY m.timestamp DESC) as rn
				FROM metrics m
				WHERE m.name = ? AND m.timestamp <= ?
			)
			SELECT
				lm.value,
				GROUP_CONCAT(ma.key || '=' || ma.value, ',') as labels
			FROM latest_metrics lm
			LEFT JOIN metric_attributes ma ON lm.id = ma.metric_id
			WHERE lm.rn = 1
			GROUP BY lm.id, lm.value
		`
		args = []any{metricName, queryNano}
	}

	rows, err := db.HealthInstance.Db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	results := []QueryResult{}
	ts := queryTime.Unix()

	for rows.Next() {
		var value float64
		var labelValue sql.NullString

		if aggregation != "" && groupBy != "" {
			if err := rows.Scan(&labelValue, &value); err != nil {
				return nil, err
			}
		} else if aggregation != "" {
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
		} else {
			var labels sql.NullString
			if err := rows.Scan(&value, &labels); err != nil {
				return nil, err
			}
			labelValue = labels
		}

		labels := make(map[string]string)
		if groupBy != "" && labelValue.Valid {
			labels[groupBy] = labelValue.String
		} else if labelValue.Valid && labelValue.String != "" {
			for _, pair := range strings.Split(labelValue.String, ",") {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					labels[parts[0]] = parts[1]
				}
			}
		}
		labels["__name__"] = metricName

		results = append(results, QueryResult{
			Metric: labels,
			Values: [][]any{{ts, fmt.Sprintf("%.6f", value)}},
		})
	}

	return results, nil
}

type labelFilter struct {
	operator string
	value    string
}

func parsePromQLQuery(query string) (metricName, aggregation, groupBy string, labelFilters map[string]labelFilter) {
	query = strings.TrimSpace(query)
	labelFilters = make(map[string]labelFilter)

	// Check for aggregation functions: sum, avg, min, max, count
	aggregations := []string{"sum", "avg", "min", "max", "count"}
	for _, agg := range aggregations {
		if strings.HasPrefix(strings.ToLower(query), agg) {
			aggregation = agg
			query = strings.TrimPrefix(query, agg)
			query = strings.TrimSpace(query)

			// Extract "by (label)" clause
			if strings.Contains(query, "by (") {
				start := strings.Index(query, "by (")
				end := strings.Index(query[start:], ")")
				if end > 0 {
					groupBy = strings.TrimSpace(query[start+4 : start+end])
					query = strings.TrimSpace(query[:start] + query[start+end+1:])
				}
			}
			break
		}
	}

	// Extract metric name and label selectors
	query = strings.Trim(query, "()")
	query = strings.TrimSpace(query)

	// Extract label selectors like {http.route=~"/api.*"}
	if idx := strings.Index(query, "{"); idx >= 0 {
		metricName = strings.TrimSpace(query[:idx])
		endIdx := strings.LastIndex(query, "}")
		if endIdx > idx {
			labelSelector := query[idx+1 : endIdx]
			// Parse label filters
			for _, filter := range strings.Split(labelSelector, ",") {
				filter = strings.TrimSpace(filter)
				if strings.Contains(filter, "=~") {
					parts := strings.SplitN(filter, "=~", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
						// Convert regex to SQL LIKE pattern
						value = strings.ReplaceAll(value, ".*", "%")
						labelFilters[key] = labelFilter{operator: "=~", value: value}
					}
				} else if strings.Contains(filter, "!~") {
					parts := strings.SplitN(filter, "!~", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
						// Convert regex to SQL LIKE pattern
						value = strings.ReplaceAll(value, ".*", "%")
						labelFilters[key] = labelFilter{operator: "!~", value: value}
					}
				}
			}
		}
	} else {
		metricName = query
	}

	return
}

func getAggregationSQL(aggregation string) string {
	switch strings.ToLower(aggregation) {
	case "sum":
		return "SUM"
	case "avg":
		return "AVG"
	case "min":
		return "MIN"
	case "max":
		return "MAX"
	case "count":
		return "COUNT"
	default:
		return "AVG"
	}
}

func formatLabels(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		if k != "__name__" {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return strings.Join(parts, ",")
}
