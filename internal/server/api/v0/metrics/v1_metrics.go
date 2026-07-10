package v0_metrics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
)

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

type labelFilter struct {
	operator string
	value    string
}

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listMetricsRoute,
		queryRangeRoute,
		queryRoute,
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

	// Extract label selectors like {http.route=~"/serverutil.*"}
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
