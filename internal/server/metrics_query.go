package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	Values [][]interface{}   `json:"values"`
}

func setupMetricsQueryRoutes(router *gin.Engine) {
	router.GET("/metrics/query_range", handleQueryRange)
	router.GET("/metrics/query", handleInstantQuery)
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
	metricName, aggregation, groupBy := parsePromQLQuery(query)

	if metricName == "" {
		return nil, fmt.Errorf("could not parse metric name from query")
	}

	// Convert timestamps to nanoseconds
	startNano := start * 1e9
	endNano := end * 1e9

	// Build SQL query based on aggregation
	var sqlQuery string
	var args []interface{}

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
			GROUP BY label_value, ts / (? * 1000000000)
			ORDER BY ts
		`
		args = []interface{}{groupBy, metricName, startNano, endNano, step}
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
		args = []interface{}{metricName, startNano, endNano, step}
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
		args = []interface{}{metricName, startNano, endNano}
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
				Values: [][]interface{}{},
			}
		}

		// Add timestamp and value
		resultMap[labelKey].Values = append(resultMap[labelKey].Values, []interface{}{ts, fmt.Sprintf("%.6f", value)})
	}

	// Convert map to slice
	results := make([]QueryResult, 0, len(resultMap))
	for _, result := range resultMap {
		results = append(results, *result)
	}

	return results, nil
}

func executeInstantQuery(ctx context.Context, query string, queryTime time.Time) ([]QueryResult, error) {
	metricName, aggregation, groupBy := parsePromQLQuery(query)

	if metricName == "" {
		return nil, fmt.Errorf("could not parse metric name from query")
	}

	queryNano := queryTime.UnixNano()

	var sqlQuery string
	var args []interface{}

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
		args = []interface{}{groupBy, metricName, queryNano}
	} else if aggregation != "" {
		sqlQuery = `
			SELECT ` + getAggregationSQL(aggregation) + `(m.value) as value
			FROM metrics m
			WHERE m.name = ? AND m.timestamp <= ?
		`
		args = []interface{}{metricName, queryNano}
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
		args = []interface{}{metricName, queryNano}
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
			Values: [][]interface{}{{ts, fmt.Sprintf("%.6f", value)}},
		})
	}

	return results, nil
}

func parsePromQLQuery(query string) (metricName, aggregation, groupBy string) {
	query = strings.TrimSpace(query)

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

	// Extract metric name (remove parentheses and brackets)
	query = strings.Trim(query, "()")
	query = strings.TrimSpace(query)

	// Remove label selectors like {job="something"}
	if idx := strings.Index(query, "{"); idx >= 0 {
		metricName = strings.TrimSpace(query[:idx])
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
