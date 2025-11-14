package botelsqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TraceExporter is a SQLite-based trace exporter that implements the SpanExporter interface
type TraceExporter struct {
	db *sql.DB
}

// NewTraceExporter creates a new SQLite trace exporter
func NewTraceExporter(db *sql.DB) (*TraceExporter, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	exporter := &TraceExporter{
		db: db,
	}

	// Initialize the database schema
	if err := exporter.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return exporter, nil
}

// initSchema creates the necessary tables for storing traces
func (e *TraceExporter) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS traces (
		trace_id TEXT NOT NULL,
		span_id TEXT NOT NULL PRIMARY KEY,
		parent_span_id TEXT,
		name TEXT NOT NULL,
		span_kind INTEGER NOT NULL,
		start_time INTEGER NOT NULL,
		end_time INTEGER NOT NULL,
		status_code TEXT NOT NULL,
		status_description TEXT,
		dropped_attributes INTEGER DEFAULT 0,
		dropped_events INTEGER DEFAULT 0,
		dropped_links INTEGER DEFAULT 0,
		child_span_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_traces_trace_id ON traces(trace_id);
	CREATE INDEX IF NOT EXISTS idx_traces_parent_span_id ON traces(parent_span_id);
	CREATE INDEX IF NOT EXISTS idx_traces_start_time ON traces(start_time);

	CREATE TABLE IF NOT EXISTS trace_attributes (
		span_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value_type TEXT NOT NULL,
		value TEXT NOT NULL,
		FOREIGN KEY (span_id) REFERENCES traces(span_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_trace_attributes_span_id ON trace_attributes(span_id);
	CREATE INDEX IF NOT EXISTS idx_trace_attributes_key ON trace_attributes(key);

	CREATE TABLE IF NOT EXISTS trace_events (
		span_id TEXT NOT NULL,
		name TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		attributes TEXT,
		FOREIGN KEY (span_id) REFERENCES traces(span_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_trace_events_span_id ON trace_events(span_id);

	CREATE TABLE IF NOT EXISTS trace_links (
		span_id TEXT NOT NULL,
		trace_id TEXT NOT NULL,
		linked_span_id TEXT NOT NULL,
		trace_state TEXT,
		attributes TEXT,
		FOREIGN KEY (span_id) REFERENCES traces(span_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_trace_links_span_id ON trace_links(span_id);

	CREATE TABLE IF NOT EXISTS trace_resources (
		span_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value_type TEXT NOT NULL,
		value TEXT NOT NULL,
		FOREIGN KEY (span_id) REFERENCES traces(span_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_trace_resources_span_id ON trace_resources(span_id);

	CREATE TABLE IF NOT EXISTS trace_scopes (
		span_id TEXT NOT NULL PRIMARY KEY,
		name TEXT NOT NULL,
		version TEXT,
		schema_url TEXT,
		FOREIGN KEY (span_id) REFERENCES traces(span_id) ON DELETE CASCADE
	);
	`

	_, err := e.db.Exec(schema)
	return err
}

// ExportSpans exports a batch of spans to SQLite
func (e *TraceExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, span := range spans {
		if err := e.exportSpan(ctx, tx, span); err != nil {
			return fmt.Errorf("failed to export span: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func getStatusCodeFromAttributes(attributes []attribute.KeyValue) codes.Code {
	for _, attr := range attributes {
		if attr.Key == "http.response.status_code" {
			switch attr.Value.Type() {
			case attribute.INT64:
				statusCode := attr.Value.AsInt64()
				if statusCode >= 200 && statusCode < 500 {
					return codes.Ok
				} else {
					return codes.Error
				}
			}
		}
	}
	return codes.Unset
}

// exportSpan exports a single span to the database
func (e *TraceExporter) exportSpan(ctx context.Context, tx *sql.Tx, span sdktrace.ReadOnlySpan) error {
	spanCtx := span.SpanContext()
	parentSpanCtx := span.Parent()

	var parentSpanID *string
	if parentSpanCtx.IsValid() && parentSpanCtx.SpanID().IsValid() {
		id := parentSpanCtx.SpanID().String()
		parentSpanID = &id
	}

	statusCode := getStatusCodeFromAttributes(span.Attributes())
	// Insert main span record
	_, err := tx.ExecContext(ctx, `
		INSERT INTO traces (
			trace_id, span_id, parent_span_id, name, span_kind,
			start_time, end_time, status_code, status_description,
			dropped_attributes, dropped_events, dropped_links, child_span_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		spanCtx.TraceID().String(),
		spanCtx.SpanID().String(),
		parentSpanID,
		span.Name(),
		span.SpanKind(),
		span.StartTime().UnixNano(),
		span.EndTime().UnixNano(),
		statusCode.String(),
		span.Status().Description,
		span.DroppedAttributes(),
		span.DroppedEvents(),
		span.DroppedLinks(),
		span.ChildSpanCount(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert span: %w", err)
	}

	// Insert attributes
	for _, attr := range span.Attributes() {
		if err := e.insertAttribute(ctx, tx, spanCtx.SpanID().String(), attr); err != nil {
			return fmt.Errorf("failed to insert attribute: %w", err)
		}
	}

	// Insert events
	for _, event := range span.Events() {
		attrs, err := json.Marshal(event.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal event attributes: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO trace_events (span_id, name, timestamp, attributes)
			VALUES (?, ?, ?, ?)
		`, spanCtx.SpanID().String(), event.Name, event.Time.UnixNano(), string(attrs))
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
	}

	// Insert links
	for _, link := range span.Links() {
		attrs, err := json.Marshal(link.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal link attributes: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO trace_links (span_id, trace_id, linked_span_id, trace_state, attributes)
			VALUES (?, ?, ?, ?, ?)
		`,
			spanCtx.SpanID().String(),
			link.SpanContext.TraceID().String(),
			link.SpanContext.SpanID().String(),
			link.SpanContext.TraceState().String(),
			string(attrs),
		)
		if err != nil {
			return fmt.Errorf("failed to insert link: %w", err)
		}
	}

	// Insert resource attributes
	for _, attr := range span.Resource().Attributes() {
		if err := e.insertResourceAttribute(ctx, tx, spanCtx.SpanID().String(), attr); err != nil {
			return fmt.Errorf("failed to insert resource attribute: %w", err)
		}
	}

	// Insert instrumentation scope
	scope := span.InstrumentationScope()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trace_scopes (span_id, name, version, schema_url)
		VALUES (?, ?, ?, ?)
	`, spanCtx.SpanID().String(), scope.Name, scope.Version, scope.SchemaURL)
	if err != nil {
		return fmt.Errorf("failed to insert scope: %w", err)
	}

	return nil
}

// insertAttribute inserts a span attribute into the database
func (e *TraceExporter) insertAttribute(ctx context.Context, tx *sql.Tx, spanID string, attr attribute.KeyValue) error {
	valueType := attr.Value.Type()
	value := attr.Value.AsString()
	switch valueType {
	case attribute.INT64:
		value = strconv.Itoa(int(attr.Value.AsInt64()))
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO trace_attributes (span_id, key, value_type, value)
		VALUES (?, ?, ?, ?)
	`, spanID, string(attr.Key), valueType.String(), value)
	return err
}

// insertResourceAttribute inserts a resource attribute into the database
func (e *TraceExporter) insertResourceAttribute(ctx context.Context, tx *sql.Tx, spanID string, attr attribute.KeyValue) error {
	valueType := attr.Value.Type().String()
	value := attr.Value.AsString()

	_, err := tx.ExecContext(ctx, `
		INSERT INTO trace_resources (span_id, key, value_type, value)
		VALUES (?, ?, ?, ?)
	`, spanID, string(attr.Key), valueType, value)
	return err
}

// Shutdown shuts down the exporter
func (e *TraceExporter) Shutdown(ctx context.Context) error {
	// No cleanup needed for SQLite connection as it's managed externally
	return nil
}

// MarshalLog is a helper to log the exporter (implements logr.Marshaler)
func (e *TraceExporter) MarshalLog() any {
	return struct {
		Type string
	}{
		Type: "sqlite",
	}
}

// PrometheusMetrics generates Prometheus-formatted metrics from stored traces
func (e *TraceExporter) PrometheusMetrics(ctx context.Context) (string, error) {
	var sb strings.Builder

	// Metric 1: Total number of spans
	totalSpans, err := e.getTotalSpans(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get total spans: %w", err)
	}
	sb.WriteString("# HELP traces_total Total number of trace spans\n")
	sb.WriteString("# TYPE traces_total counter\n")
	sb.WriteString(fmt.Sprintf("traces_total %d\n\n", totalSpans))

	// Metric 2: Spans by status code
	spansByStatus, err := e.getSpansByStatus(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get spans by status: %w", err)
	}
	sb.WriteString("# HELP traces_by_status_total Total spans by status code\n")
	sb.WriteString("# TYPE traces_by_status_total counter\n")
	for status, count := range spansByStatus {
		sb.WriteString(fmt.Sprintf("traces_by_status_total{status_code=\"%s\"} %d\n", status, count))
	}
	sb.WriteString("\n")

	// Metric 3: Spans by name (operation)
	spansByName, err := e.getSpansByName(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get spans by name: %w", err)
	}
	sb.WriteString("# HELP traces_by_operation_total Total spans by operation name\n")
	sb.WriteString("# TYPE traces_by_operation_total counter\n")
	for name, count := range spansByName {
		// Sanitize label value
		sanitized := strings.ReplaceAll(name, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf("traces_by_operation_total{operation=\"%s\"} %d\n", sanitized, count))
	}
	sb.WriteString("\n")

	// Metric 4: Average span duration by operation
	avgDurations, err := e.getAvgDurationByName(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get average durations: %w", err)
	}
	sb.WriteString("# HELP traces_duration_seconds_avg Average span duration in seconds by operation\n")
	sb.WriteString("# TYPE traces_duration_seconds_avg gauge\n")
	for name, duration := range avgDurations {
		sanitized := strings.ReplaceAll(name, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf("traces_duration_seconds_avg{operation=\"%s\"} %.6f\n", sanitized, duration))
	}
	sb.WriteString("\n")

	// Metric 5: Span duration histogram buckets (p50, p95, p99)
	percentiles, err := e.getDurationPercentiles(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get duration percentiles: %w", err)
	}
	sb.WriteString("# HELP traces_duration_seconds_p50 50th percentile span duration in seconds\n")
	sb.WriteString("# TYPE traces_duration_seconds_p50 gauge\n")
	sb.WriteString(fmt.Sprintf("traces_duration_seconds_p50 %.6f\n", percentiles.P50))
	sb.WriteString("# HELP traces_duration_seconds_p95 95th percentile span duration in seconds\n")
	sb.WriteString("# TYPE traces_duration_seconds_p95 gauge\n")
	sb.WriteString(fmt.Sprintf("traces_duration_seconds_p95 %.6f\n", percentiles.P95))
	sb.WriteString("# HELP traces_duration_seconds_p99 99th percentile span duration in seconds\n")
	sb.WriteString("# TYPE traces_duration_seconds_p99 gauge\n")
	sb.WriteString(fmt.Sprintf("traces_duration_seconds_p99 %.6f\n\n", percentiles.P99))

	// Metric 6: Spans per HTTP method (if available)
	spansByHTTPMethod, err := e.getSpansByHTTPMethod(ctx)
	if err == nil && len(spansByHTTPMethod) > 0 {
		sb.WriteString("# HELP traces_http_requests_total Total HTTP request spans by method\n")
		sb.WriteString("# TYPE traces_http_requests_total counter\n")
		for method, count := range spansByHTTPMethod {
			sb.WriteString(fmt.Sprintf("traces_http_requests_total{method=\"%s\"} %d\n", method, count))
		}
		sb.WriteString("\n")
	}

	// Metric 7: Spans per HTTP status code (if available)
	spansByHTTPStatus, err := e.getSpansByHTTPStatus(ctx)
	if err == nil && len(spansByHTTPStatus) > 0 {
		sb.WriteString("# HELP traces_http_responses_total Total HTTP response spans by status code\n")
		sb.WriteString("# TYPE traces_http_responses_total counter\n")
		for status, count := range spansByHTTPStatus {
			sb.WriteString(fmt.Sprintf("traces_http_responses_total{status_code=\"%s\"} %d\n", status, count))
		}
		sb.WriteString("\n")
	}

	// Metric 8: Recent error count (last 5 minutes)
	recentErrors, err := e.getRecentErrors(ctx, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to get recent errors: %w", err)
	}
	sb.WriteString("# HELP traces_errors_recent_total Recent error spans in the last 5 minutes\n")
	sb.WriteString("# TYPE traces_errors_recent_total gauge\n")
	sb.WriteString(fmt.Sprintf("traces_errors_recent_total %d\n\n", recentErrors))

	if err := e.appendCustomMetrics(ctx, &sb); err != nil {
		sb.WriteString(fmt.Sprintf("# Error loading custom metrics: %s\n", err.Error()))
	}

	return sb.String(), nil
}

type Percentiles struct {
	P50 float64
	P95 float64
	P99 float64
}

func (e *TraceExporter) getTotalSpans(ctx context.Context) (int64, error) {
	var count int64
	err := e.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM traces").Scan(&count)
	return count, err
}

func (e *TraceExporter) getSpansByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT status_code, COUNT(*) as count
		FROM traces
		GROUP BY status_code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, rows.Err()
}

func (e *TraceExporter) getSpansByName(ctx context.Context) (map[string]int64, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT name, COUNT(*) as count
		FROM traces
		GROUP BY name
		ORDER BY count DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var name string
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		result[name] = count
	}
	return result, rows.Err()
}

func (e *TraceExporter) getAvgDurationByName(ctx context.Context) (map[string]float64, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT name, AVG(end_time - start_time) as avg_duration
		FROM traces
		GROUP BY name
		ORDER BY avg_duration DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var name string
		var avgDuration float64 // nanoseconds (as float from AVG)
		if err := rows.Scan(&name, &avgDuration); err != nil {
			return nil, err
		}
		result[name] = avgDuration / 1e9 // convert to seconds
	}
	return result, rows.Err()
}

func (e *TraceExporter) getDurationPercentiles(ctx context.Context) (*Percentiles, error) {
	var p Percentiles

	// SQLite doesn't have built-in percentile functions, so we use a workaround
	// Get duration values and calculate percentiles
	rows, err := e.db.QueryContext(ctx, `
		SELECT (end_time - start_time) as duration
		FROM traces
		ORDER BY duration
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var durations []int64
	for rows.Next() {
		var duration int64
		if err := rows.Scan(&duration); err != nil {
			return nil, err
		}
		durations = append(durations, duration)
	}

	if len(durations) == 0 {
		return &p, nil
	}

	// Calculate percentiles
	p50Idx := len(durations) * 50 / 100
	p95Idx := len(durations) * 95 / 100
	p99Idx := len(durations) * 99 / 100

	if p50Idx < len(durations) {
		p.P50 = float64(durations[p50Idx]) / 1e9
	}
	if p95Idx < len(durations) {
		p.P95 = float64(durations[p95Idx]) / 1e9
	}
	if p99Idx < len(durations) {
		p.P99 = float64(durations[p99Idx]) / 1e9
	}

	return &p, rows.Err()
}

func (e *TraceExporter) getSpansByHTTPMethod(ctx context.Context) (map[string]int64, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT ta.value, COUNT(*) as count
		FROM traces t
		JOIN trace_attributes ta ON t.span_id = ta.span_id
		WHERE ta.key = 'http.request.method'
		GROUP BY ta.value
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var method string
		var count int64
		if err := rows.Scan(&method, &count); err != nil {
			return nil, err
		}
		result[method] = count
	}
	return result, rows.Err()
}

func (e *TraceExporter) getSpansByHTTPStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT ta.value, COUNT(*) as count
		FROM traces t
		JOIN trace_attributes ta ON t.span_id = ta.span_id
		WHERE ta.key = 'http.response.status_code'
		GROUP BY ta.value
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, rows.Err()
}

func (e *TraceExporter) getRecentErrors(ctx context.Context, window time.Duration) (int64, error) {
	cutoff := time.Now().Add(-window).UnixNano()
	var count int64
	err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM traces
		WHERE status_code = 'Error' AND start_time > ?
	`, cutoff).Scan(&count)
	return count, err
}

type metricInfo struct {
	description string
	metricType  string
	values      []metricValue
}

type metricValue struct {
	value      float64
	metricID   int64
	attributes map[string]string
}

// appendCustomMetrics appends custom OTEL metrics from the metrics table to Prometheus output
func (e *TraceExporter) appendCustomMetrics(ctx context.Context, sb *strings.Builder) error {
	// Check if metrics table exists
	var tableExists int
	err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='metrics'
	`).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		return nil // Table doesn't exist, skip
	}

	// Query for latest value of each unique metric (name + attributes combination)
	rows, err := e.db.QueryContext(ctx, `
		WITH latest_metrics AS (
			SELECT
				m.name,
				m.description,
				m.type,
				m.value,
				m.id,
				ROW_NUMBER() OVER (PARTITION BY m.name,
					(SELECT GROUP_CONCAT(ma.key || '=' || ma.value, ',')
					 FROM metric_attributes ma
					 WHERE ma.metric_id = m.id
					 ORDER BY ma.key)
				ORDER BY m.timestamp DESC) as rn
			FROM metrics m
		)
		SELECT name, description, type, value, id
		FROM latest_metrics
		WHERE rn = 1
		ORDER BY name
	`)
	if err != nil {
		return fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	// Group metrics by name for proper Prometheus formatting
	metricsMap := make(map[string]*metricInfo)

	for rows.Next() {
		var name, description, metricType string
		var value float64
		var metricID int64

		if err := rows.Scan(&name, &description, &metricType, &value, &metricID); err != nil {
			return fmt.Errorf("failed to scan metric: %w", err)
		}

		// Get attributes for this metric
		attrs, err := e.getMetricAttributes(ctx, metricID)
		if err != nil {
			return fmt.Errorf("failed to get metric attributes: %w", err)
		}

		if _, exists := metricsMap[name]; !exists {
			metricsMap[name] = &metricInfo{
				description: description,
				metricType:  metricType,
				values:      []metricValue{},
			}
		}

		metricsMap[name].values = append(metricsMap[name].values, metricValue{
			value:      value,
			metricID:   metricID,
			attributes: attrs,
		})
	}

	// Format as Prometheus metrics
	for name, info := range metricsMap {
		// Write HELP and TYPE
		if info.description != "" {
			sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, info.description))
		} else {
			sb.WriteString(fmt.Sprintf("# HELP %s Custom metric\n", name))
		}

		promType := info.metricType
		sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", name, promType))

		// Write values with labels
		for _, v := range info.values {
			if len(v.attributes) > 0 {
				// Build label string
				var labels []string
				for key, val := range v.attributes {
					sanitizedVal := strings.ReplaceAll(val, "\"", "\\\"")
					labels = append(labels, fmt.Sprintf("%s=\"%s\"", key, sanitizedVal))
				}
				sb.WriteString(fmt.Sprintf("%s{%s} %.6f\n", name, strings.Join(labels, ","), v.value))
			} else {
				sb.WriteString(fmt.Sprintf("%s %.6f\n", name, v.value))
			}
		}
		sb.WriteString("\n")
	}

	return rows.Err()
}

// getMetricAttributes retrieves attributes for a specific metric
func (e *TraceExporter) getMetricAttributes(ctx context.Context, metricID int64) (map[string]string, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT key, value
		FROM metric_attributes
		WHERE metric_id = ?
	`, metricID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attrs := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		attrs[key] = value
	}

	return attrs, rows.Err()
}
