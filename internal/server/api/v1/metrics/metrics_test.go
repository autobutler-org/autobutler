package v1_metrics

import (
	"sort"
	"strings"
	"testing"
)

func TestParsePromQLQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantMetric  string
		wantAgg     string
		wantGroupBy string
		wantFilters map[string]labelFilter
	}{
		{
			name:       "simple metric name",
			query:      "system.cpu.utilization",
			wantMetric: "system.cpu.utilization",
		},
		{
			name:       "metric with underscores",
			query:      "http_requests_total",
			wantMetric: "http_requests_total",
		},
		{
			name:       "with aggregation",
			query:      "avg(system.cpu.utilization)",
			wantMetric: "system.cpu.utilization",
			wantAgg:    "avg",
		},
		{
			name:       "sum aggregation",
			query:      "sum(http_requests_total)",
			wantMetric: "http_requests_total",
			wantAgg:    "sum",
		},
		{
			name:       "count aggregation",
			query:      "count(http_requests_total)",
			wantMetric: "http_requests_total",
			wantAgg:    "count",
		},
		{
			name:       "min aggregation",
			query:      "min(response_time)",
			wantMetric: "response_time",
			wantAgg:    "min",
		},
		{
			name:       "max aggregation",
			query:      "max(response_time)",
			wantMetric: "response_time",
			wantAgg:    "max",
		},
		{
			name:        "aggregation with group by",
			query:       "avg by (core) (system.cpu.utilization)",
			wantMetric:  "system.cpu.utilization",
			wantAgg:     "avg",
			wantGroupBy: "core",
		},
		{
			name:        "sum with group by",
			query:       "sum by (host) (http_requests_total)",
			wantMetric:  "http_requests_total",
			wantAgg:     "sum",
			wantGroupBy: "host",
		},
		{
			name:       "label filter with regex match",
			query:      `http_requests{http.route=~"/api.*"}`,
			wantMetric: "http_requests",
			wantFilters: map[string]labelFilter{
				"http.route": {operator: "=~", value: "/api%"},
			},
		},
		{
			name:       "label filter with regex negation",
			query:      `http_requests{http.route!~"/health.*"}`,
			wantMetric: "http_requests",
			wantFilters: map[string]labelFilter{
				"http.route": {operator: "!~", value: "/health%"},
			},
		},
		{
			name:       "multiple label filters",
			query:      `http_requests{http.route=~"/api.*",http.method=~"GET"}`,
			wantMetric: "http_requests",
			wantFilters: map[string]labelFilter{
				"http.route":  {operator: "=~", value: "/api%"},
				"http.method": {operator: "=~", value: "GET"},
			},
		},
		{
			name:       "empty string",
			query:      "",
			wantMetric: "",
		},
		{
			name:       "whitespace only",
			query:      "   ",
			wantMetric: "",
		},
		{
			name:       "metric with leading/trailing whitespace",
			query:      "  system.cpu.utilization  ",
			wantMetric: "system.cpu.utilization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricName, aggregation, groupBy, labelFilters := parsePromQLQuery(tt.query)

			if metricName != tt.wantMetric {
				t.Errorf("metricName = %q, want %q", metricName, tt.wantMetric)
			}
			if aggregation != tt.wantAgg {
				t.Errorf("aggregation = %q, want %q", aggregation, tt.wantAgg)
			}
			if groupBy != tt.wantGroupBy {
				t.Errorf("groupBy = %q, want %q", groupBy, tt.wantGroupBy)
			}

			if tt.wantFilters != nil {
				if len(labelFilters) != len(tt.wantFilters) {
					t.Errorf("labelFilters count = %d, want %d", len(labelFilters), len(tt.wantFilters))
				}
				for key, wantFilter := range tt.wantFilters {
					gotFilter, ok := labelFilters[key]
					if !ok {
						t.Errorf("missing label filter for key %q", key)
						continue
					}
					if gotFilter.operator != wantFilter.operator {
						t.Errorf("filter[%s].operator = %q, want %q", key, gotFilter.operator, wantFilter.operator)
					}
					if gotFilter.value != wantFilter.value {
						t.Errorf("filter[%s].value = %q, want %q", key, gotFilter.value, wantFilter.value)
					}
				}
			} else {
				if len(labelFilters) != 0 {
					t.Errorf("expected no label filters, got %d", len(labelFilters))
				}
			}
		})
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string // for single-label cases; multi-label uses wantParts
	}{
		{
			name:   "empty map",
			labels: map[string]string{},
			want:   "",
		},
		{
			name:   "single label",
			labels: map[string]string{"host": "server1"},
			want:   "host=server1",
		},
		{
			name:   "__name__ excluded",
			labels: map[string]string{"__name__": "cpu_usage", "host": "server1"},
			want:   "host=server1",
		},
		{
			name:   "only __name__",
			labels: map[string]string{"__name__": "cpu_usage"},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLabels(tt.labels)
			if got != tt.want {
				t.Errorf("formatLabels() = %q, want %q", got, tt.want)
			}
		})
	}

	// Separate test for multiple labels since map ordering is non-deterministic
	t.Run("multiple labels", func(t *testing.T) {
		labels := map[string]string{"host": "server1", "region": "us-east"}
		got := formatLabels(labels)
		parts := strings.Split(got, ",")
		sort.Strings(parts)
		want := []string{"host=server1", "region=us-east"}
		if len(parts) != len(want) {
			t.Fatalf("formatLabels() produced %d parts, want %d", len(parts), len(want))
		}
		for i, p := range parts {
			if p != want[i] {
				t.Errorf("part[%d] = %q, want %q", i, p, want[i])
			}
		}
	})
}

func TestGetAggregationSQL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "sum", input: "sum", want: "SUM"},
		{name: "avg", input: "avg", want: "AVG"},
		{name: "min", input: "min", want: "MIN"},
		{name: "max", input: "max", want: "MAX"},
		{name: "count", input: "count", want: "COUNT"},
		{name: "SUM uppercase", input: "SUM", want: "SUM"},
		{name: "Avg mixed case", input: "Avg", want: "AVG"},
		{name: "MIN uppercase", input: "MIN", want: "MIN"},
		{name: "unknown defaults to AVG", input: "median", want: "AVG"},
		{name: "empty defaults to AVG", input: "", want: "AVG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAggregationSQL(tt.input)
			if got != tt.want {
				t.Errorf("getAggregationSQL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:  "integer timestamp",
			input: "1711437960",
			want:  1711437960,
		},
		{
			name:  "float timestamp truncates decimal",
			input: "1711437960.123",
			want:  1711437960,
		},
		{
			name:  "float timestamp with .999",
			input: "1711437960.999",
			want:  1711437960,
		},
		{
			name:  "zero",
			input: "0",
			want:  0,
		},
		{
			name:    "invalid string",
			input:   "not-a-timestamp",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid float",
			input:   "12.34.56",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimestamp(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTimestamp(%q) expected error, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTimestamp(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseTimestamp(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
