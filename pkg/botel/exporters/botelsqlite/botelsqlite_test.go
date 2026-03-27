package botelsqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	_ "modernc.org/sqlite"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func newMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// createTestSpan builds a real ended span via the SDK so we get a valid ReadOnlySpan.
func createTestSpan(t *testing.T, name string, attrs ...attribute.KeyValue) sdktrace.ReadOnlySpan {
	t.Helper()
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { tp.Shutdown(context.Background()) })
	tracer := tp.Tracer("test-tracer")
	_, span := tracer.Start(context.Background(), name, trace.WithAttributes(attrs...))
	span.End()
	// Use the tracetest in-memory exporter approach or just cast.
	// The span is now ended; we can reach it through a SpanRecorder.
	// Re-do with an in-memory exporter to capture it:
	return nil // placeholder – replaced below
}

// createTestSpans returns real ReadOnlySpan values captured via tracetest.
func createTestSpans(t *testing.T, names []string, attrSets [][]attribute.KeyValue) []sdktrace.ReadOnlySpan {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	t.Cleanup(func() { tp.Shutdown(context.Background()) })
	tracer := tp.Tracer("test-tracer")

	for i, name := range names {
		var opts []trace.SpanStartOption
		if i < len(attrSets) && attrSets[i] != nil {
			opts = append(opts, trace.WithAttributes(attrSets[i]...))
		}
		_, span := tracer.Start(context.Background(), name, opts...)
		span.End()
	}

	ended := rec.Ended()
	spans := make([]sdktrace.ReadOnlySpan, len(ended))
	for i := range ended {
		spans[i] = ended[i]
	}
	return spans
}

func createSingleSpan(t *testing.T, name string, attrs ...attribute.KeyValue) sdktrace.ReadOnlySpan {
	t.Helper()
	spans := createTestSpans(t, []string{name}, [][]attribute.KeyValue{attrs})
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}
	return spans[0]
}

// ─── TraceExporter tests ────────────────────────────────────────────────────

func TestNewTraceExporter_NilDB(t *testing.T) {
	_, err := NewTraceExporter(nil)
	if err == nil {
		t.Fatal("expected error for nil db, got nil")
	}
}

func TestNewTraceExporter_ValidDB(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
}

func TestTraceExporter_SchemaCreation(t *testing.T) {
	db := newMemoryDB(t)
	_, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tables := []string{
		"traces",
		"trace_attributes",
		"trace_events",
		"trace_links",
		"trace_resources",
		"trace_scopes",
	}
	for _, table := range tables {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query for table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestTraceExporter_ExportSpans_Empty(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	// nil slice
	if err := exp.ExportSpans(context.Background(), nil); err != nil {
		t.Fatalf("export nil spans: %v", err)
	}
	// empty slice
	if err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{}); err != nil {
		t.Fatalf("export empty spans: %v", err)
	}

	var count int64
	db.QueryRow("SELECT COUNT(*) FROM traces").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}

func TestTraceExporter_ExportSpans_Single(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	span := createSingleSpan(t, "test-op",
		attribute.String("http.request.method", "GET"),
		attribute.Int64("http.response.status_code", 200),
	)

	if err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span}); err != nil {
		t.Fatalf("export: %v", err)
	}

	// Verify span stored
	var name string
	var statusCode string
	err = db.QueryRow("SELECT name, status_code FROM traces").Scan(&name, &statusCode)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if name != "test-op" {
		t.Errorf("expected name 'test-op', got %q", name)
	}
	// status_code 200 → codes.Ok → "Ok"
	if statusCode != "Ok" {
		t.Errorf("expected status_code 'Ok', got %q", statusCode)
	}

	// Verify attributes stored
	var attrCount int
	db.QueryRow("SELECT COUNT(*) FROM trace_attributes").Scan(&attrCount)
	if attrCount != 2 {
		t.Errorf("expected 2 attributes, got %d", attrCount)
	}

	// Verify scope stored
	var scopeName string
	db.QueryRow("SELECT name FROM trace_scopes").Scan(&scopeName)
	if scopeName != "test-tracer" {
		t.Errorf("expected scope name 'test-tracer', got %q", scopeName)
	}
}

func TestTraceExporter_ExportSpans_Multiple(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	spans := createTestSpans(t,
		[]string{"op-1", "op-2", "op-3"},
		[][]attribute.KeyValue{
			{attribute.String("key", "val1")},
			{attribute.Int64("count", 42)},
			nil,
		},
	)

	if err := exp.ExportSpans(context.Background(), spans); err != nil {
		t.Fatalf("export: %v", err)
	}

	var count int64
	db.QueryRow("SELECT COUNT(*) FROM traces").Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 spans, got %d", count)
	}
}

func TestTraceExporter_ExportSpans_WithHTTPError(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	span := createSingleSpan(t, "error-op",
		attribute.Int64("http.response.status_code", 500),
	)

	if err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span}); err != nil {
		t.Fatalf("export: %v", err)
	}

	var statusCode string
	db.QueryRow("SELECT status_code FROM traces").Scan(&statusCode)
	if statusCode != "Error" {
		t.Errorf("expected status 'Error' for 500, got %q", statusCode)
	}
}

func TestTraceExporter_Shutdown(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	if err := exp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestTraceExporter_MarshalLog(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	result := exp.MarshalLog()
	if result == nil {
		t.Fatal("expected non-nil marshal result")
	}
}

func TestTraceExporter_PrometheusMetrics_Empty(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	output, err := exp.PrometheusMetrics(context.Background())
	if err != nil {
		t.Fatalf("prometheus metrics: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output even with zero spans")
	}
	// Should have traces_total 0
	if !containsString(output, "traces_total 0") {
		t.Error("expected 'traces_total 0' in output")
	}
}

func TestTraceExporter_PrometheusMetrics_WithSpans(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewTraceExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	spans := createTestSpans(t,
		[]string{"GET /api/users", "POST /api/login"},
		[][]attribute.KeyValue{
			{
				attribute.String("http.request.method", "GET"),
				attribute.Int64("http.response.status_code", 200),
			},
			{
				attribute.String("http.request.method", "POST"),
				attribute.Int64("http.response.status_code", 401),
			},
		},
	)
	if err := exp.ExportSpans(context.Background(), spans); err != nil {
		t.Fatalf("export: %v", err)
	}

	output, err := exp.PrometheusMetrics(context.Background())
	if err != nil {
		t.Fatalf("prometheus metrics: %v", err)
	}

	checks := []string{
		"traces_total 2",
		"traces_by_status_total",
		"traces_by_operation_total",
		"traces_duration_seconds_avg",
		"traces_http_requests_total",
		"traces_http_responses_total",
	}
	for _, check := range checks {
		if !containsString(output, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}
}

// ─── MetricsExporter tests ─────────────────────────────────────────────────

func TestNewMetricsExporter_NilDB(t *testing.T) {
	_, err := NewMetricsExporter(nil)
	if err == nil {
		t.Fatal("expected error for nil db, got nil")
	}
}

func TestNewMetricsExporter_ValidDB(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
}

func TestMetricsExporter_SchemaCreation(t *testing.T) {
	db := newMemoryDB(t)
	_, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tables := []string{
		"metrics",
		"metric_attributes",
		"metric_histogram_buckets",
		"metric_exemplars",
	}
	for _, table := range tables {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query for table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestMetricsExporter_Export_Nil(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	if err := exp.Export(context.Background(), nil); err != nil {
		t.Fatalf("export nil: %v", err)
	}
}

func TestMetricsExporter_Export_Counter(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	now := time.Now()
	rm := &metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Metrics: []metricdata.Metrics{
					{
						Name:        "http.requests.total",
						Description: "Total HTTP requests",
						Unit:        "1",
						Data: metricdata.Sum[int64]{
							IsMonotonic: true,
							DataPoints: []metricdata.DataPoint[int64]{
								{
									Time:       now,
									Value:      42,
									Attributes: attribute.NewSet(attribute.String("method", "GET")),
								},
							},
						},
					},
				},
			},
		},
	}

	if err := exp.Export(context.Background(), rm); err != nil {
		t.Fatalf("export: %v", err)
	}

	var name, metricType string
	var value float64
	err = db.QueryRow("SELECT name, type, value FROM metrics").Scan(&name, &metricType, &value)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if name != "http.requests.total" {
		t.Errorf("expected name 'http.requests.total', got %q", name)
	}
	if metricType != "counter" {
		t.Errorf("expected type 'counter', got %q", metricType)
	}
	if value != 42 {
		t.Errorf("expected value 42, got %f", value)
	}

	// Check attribute
	var attrKey, attrVal string
	err = db.QueryRow("SELECT key, value FROM metric_attributes").Scan(&attrKey, &attrVal)
	if err != nil {
		t.Fatalf("query attributes: %v", err)
	}
	if attrKey != "method" || attrVal != "GET" {
		t.Errorf("expected method=GET, got %s=%s", attrKey, attrVal)
	}
}

func TestMetricsExporter_Export_FloatCounter(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	now := time.Now()
	rm := &metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Metrics: []metricdata.Metrics{
					{
						Name: "request.duration",
						Unit: "s",
						Data: metricdata.Sum[float64]{
							DataPoints: []metricdata.DataPoint[float64]{
								{
									Time:  now,
									Value: 3.14,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := exp.Export(context.Background(), rm); err != nil {
		t.Fatalf("export: %v", err)
	}

	var value float64
	db.QueryRow("SELECT value FROM metrics").Scan(&value)
	if value != 3.14 {
		t.Errorf("expected 3.14, got %f", value)
	}
}

func TestMetricsExporter_Export_Gauge(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	now := time.Now()
	rm := &metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Metrics: []metricdata.Metrics{
					{
						Name:        "system.memory.usage",
						Description: "Memory usage",
						Unit:        "By",
						Data: metricdata.Gauge[int64]{
							DataPoints: []metricdata.DataPoint[int64]{
								{
									Time:  now,
									Value: 1024000,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := exp.Export(context.Background(), rm); err != nil {
		t.Fatalf("export: %v", err)
	}

	var metricType string
	var value float64
	db.QueryRow("SELECT type, value FROM metrics").Scan(&metricType, &value)
	if metricType != "gauge" {
		t.Errorf("expected type 'gauge', got %q", metricType)
	}
	if value != 1024000 {
		t.Errorf("expected 1024000, got %f", value)
	}
}

func TestMetricsExporter_Export_FloatGauge(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	now := time.Now()
	rm := &metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Metrics: []metricdata.Metrics{
					{
						Name: "system.cpu.utilization",
						Unit: "1",
						Data: metricdata.Gauge[float64]{
							DataPoints: []metricdata.DataPoint[float64]{
								{
									Time:  now,
									Value: 0.75,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := exp.Export(context.Background(), rm); err != nil {
		t.Fatalf("export: %v", err)
	}

	var value float64
	db.QueryRow("SELECT value FROM metrics").Scan(&value)
	if value != 0.75 {
		t.Errorf("expected 0.75, got %f", value)
	}
}

func TestMetricsExporter_Export_Histogram(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	now := time.Now()
	rm := &metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Metrics: []metricdata.Metrics{
					{
						Name:        "http.request.duration",
						Description: "HTTP request duration",
						Unit:        "ms",
						Data: metricdata.Histogram[float64]{
							DataPoints: []metricdata.HistogramDataPoint[float64]{
								{
									Time:         now,
									Sum:          150.5,
									Count:        10,
									Bounds:       []float64{10, 50, 100, 250},
									BucketCounts: []uint64{2, 3, 3, 1, 1},
									Attributes:   attribute.NewSet(attribute.String("route", "/api")),
								},
							},
						},
					},
				},
			},
		},
	}

	if err := exp.Export(context.Background(), rm); err != nil {
		t.Fatalf("export: %v", err)
	}

	var metricType string
	var value float64
	db.QueryRow("SELECT type, value FROM metrics").Scan(&metricType, &value)
	if metricType != "histogram" {
		t.Errorf("expected type 'histogram', got %q", metricType)
	}
	if value != 150.5 {
		t.Errorf("expected sum 150.5, got %f", value)
	}

	// Verify buckets
	var bucketCount int
	db.QueryRow("SELECT COUNT(*) FROM metric_histogram_buckets").Scan(&bucketCount)
	if bucketCount != 5 {
		t.Errorf("expected 5 buckets, got %d", bucketCount)
	}
}

func TestMetricsExporter_Export_IntHistogram(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	now := time.Now()
	rm := &metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Metrics: []metricdata.Metrics{
					{
						Name: "request.size",
						Unit: "By",
						Data: metricdata.Histogram[int64]{
							DataPoints: []metricdata.HistogramDataPoint[int64]{
								{
									Time:         now,
									Sum:          5000,
									Count:        3,
									Bounds:       []float64{1000, 5000},
									BucketCounts: []uint64{1, 1, 1},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := exp.Export(context.Background(), rm); err != nil {
		t.Fatalf("export: %v", err)
	}

	var value float64
	db.QueryRow("SELECT value FROM metrics").Scan(&value)
	if value != 5000 {
		t.Errorf("expected 5000, got %f", value)
	}
}

func TestMetricsExporter_Export_MultipleMetrics(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	now := time.Now()
	rm := &metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Metrics: []metricdata.Metrics{
					{
						Name: "counter.one",
						Data: metricdata.Sum[int64]{
							DataPoints: []metricdata.DataPoint[int64]{
								{Time: now, Value: 10},
							},
						},
					},
					{
						Name: "gauge.one",
						Data: metricdata.Gauge[float64]{
							DataPoints: []metricdata.DataPoint[float64]{
								{Time: now, Value: 99.9},
							},
						},
					},
				},
			},
		},
	}

	if err := exp.Export(context.Background(), rm); err != nil {
		t.Fatalf("export: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM metrics").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 metrics, got %d", count)
	}
}

func TestMetricsExporter_Temporality(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	kinds := []metric.InstrumentKind{
		metric.InstrumentKindCounter,
		metric.InstrumentKindUpDownCounter,
		metric.InstrumentKindHistogram,
		metric.InstrumentKindGauge,
		metric.InstrumentKindObservableCounter,
		metric.InstrumentKindObservableUpDownCounter,
		metric.InstrumentKindObservableGauge,
	}

	for _, kind := range kinds {
		temp := exp.Temporality(kind)
		if temp != metricdata.CumulativeTemporality {
			t.Errorf("expected CumulativeTemporality for %v, got %v", kind, temp)
		}
	}
}

func TestMetricsExporter_Aggregation(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}

	// Should return default aggregation (non-nil) for each kind
	kinds := []metric.InstrumentKind{
		metric.InstrumentKindCounter,
		metric.InstrumentKindHistogram,
		metric.InstrumentKindGauge,
	}

	for _, kind := range kinds {
		agg := exp.Aggregation(kind)
		if agg == nil {
			t.Errorf("expected non-nil aggregation for %v", kind)
		}
	}
}

func TestMetricsExporter_ForceFlush(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	if err := exp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("force flush: %v", err)
	}
}

func TestMetricsExporter_Shutdown(t *testing.T) {
	db := newMemoryDB(t)
	exp, err := NewMetricsExporter(db)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	if err := exp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

// ─── SpanToJSON tests ───────────────────────────────────────────────────────

func TestSpanToJSON_Basic(t *testing.T) {
	span := createSingleSpan(t, "json-test-op",
		attribute.String("foo", "bar"),
		attribute.Int64("count", 7),
	)

	result, err := SpanToJSON(span)
	if err != nil {
		t.Fatalf("SpanToJSON: %v", err)
	}

	if result["Name"] != "json-test-op" {
		t.Errorf("expected Name 'json-test-op', got %v", result["Name"])
	}

	// Check SpanContext fields exist
	sc, ok := result["SpanContext"].(map[string]any)
	if !ok {
		t.Fatal("expected SpanContext to be map[string]any")
	}
	if sc["TraceID"] == "" {
		t.Error("expected non-empty TraceID")
	}
	if sc["SpanID"] == "" {
		t.Error("expected non-empty SpanID")
	}

	// Check attributes
	attrs, ok := result["Attributes"].([]map[string]any)
	if !ok {
		t.Fatal("expected Attributes to be []map[string]any")
	}
	if len(attrs) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(attrs))
	}

	// Check Status
	status, ok := result["Status"].(map[string]any)
	if !ok {
		t.Fatal("expected Status to be map[string]any")
	}
	if status["Code"] != "Unset" {
		t.Errorf("expected status code 'Unset', got %v", status["Code"])
	}

	// Events and Links should be nil for a bare span
	if result["Events"] != nil {
		t.Error("expected nil Events")
	}
	if result["Links"] != nil {
		t.Error("expected nil Links")
	}

	// InstrumentationScope
	scope, ok := result["InstrumentationScope"].(map[string]any)
	if !ok {
		t.Fatal("expected InstrumentationScope to be map[string]any")
	}
	if scope["Name"] != "test-tracer" {
		t.Errorf("expected scope name 'test-tracer', got %v", scope["Name"])
	}

	// InstrumentationLibrary should equal InstrumentationScope
	lib, ok := result["InstrumentationLibrary"].(map[string]any)
	if !ok {
		t.Fatal("expected InstrumentationLibrary to be map[string]any")
	}
	if lib["Name"] != scope["Name"] {
		t.Error("expected InstrumentationLibrary to match InstrumentationScope")
	}
}

func TestSpanToJSON_NoAttributes(t *testing.T) {
	span := createSingleSpan(t, "bare-span")

	result, err := SpanToJSON(span)
	if err != nil {
		t.Fatalf("SpanToJSON: %v", err)
	}

	attrs, ok := result["Attributes"].([]map[string]any)
	if !ok {
		t.Fatal("expected Attributes to be []map[string]any")
	}
	if len(attrs) != 0 {
		t.Errorf("expected 0 attributes, got %d", len(attrs))
	}
}

// ─── getStatusCodeFromAttributes tests ──────────────────────────────────────

func TestGetStatusCodeFromAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    []attribute.KeyValue
		expected codes.Code
	}{
		{
			name:     "no attributes",
			attrs:    nil,
			expected: codes.Unset,
		},
		{
			name:     "200 OK",
			attrs:    []attribute.KeyValue{attribute.Int64("http.response.status_code", 200)},
			expected: codes.Ok,
		},
		{
			name:     "404 OK",
			attrs:    []attribute.KeyValue{attribute.Int64("http.response.status_code", 404)},
			expected: codes.Ok,
		},
		{
			name:     "499 OK",
			attrs:    []attribute.KeyValue{attribute.Int64("http.response.status_code", 499)},
			expected: codes.Ok,
		},
		{
			name:     "500 Error",
			attrs:    []attribute.KeyValue{attribute.Int64("http.response.status_code", 500)},
			expected: codes.Error,
		},
		{
			name:     "503 Error",
			attrs:    []attribute.KeyValue{attribute.Int64("http.response.status_code", 503)},
			expected: codes.Error,
		},
		{
			name:     "unrelated attribute",
			attrs:    []attribute.KeyValue{attribute.String("foo", "bar")},
			expected: codes.Unset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusCodeFromAttributes(tt.attrs)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
