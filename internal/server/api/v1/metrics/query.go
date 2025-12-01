package v1_metrics

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func handleQuery(c *gin.Context) {
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
	results, err := executeQuery(c, query, queryTime)
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

func executeQuery(c *gin.Context, query string, queryTime time.Time) ([]QueryResult, error) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return nil, fmt.Errorf("dependencies not found in context")
	}

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

	rows, err := deps.HealthDatabase().Db.QueryContext(c.Request.Context(), sqlQuery, args...)
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
