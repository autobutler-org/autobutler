package botel

import (
	"fmt"
	"time"

	"github.com/autobutler-org/autobutler/pkg/botel/exporters/botelsqlite"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitTracer(deps deputil.Dependencies) (*sdktrace.TracerProvider, error) {
	exporter, err := botelsqlite.NewTraceExporter(deps.HealthDatabase().Db)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite exporter: %w", err)
	}
	deps.WithMetricsExporter(exporter)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp, nil
}

func InitMetrics(deps deputil.Dependencies) (*metric.MeterProvider, error) {
	metricsExp, err := botelsqlite.NewMetricsExporter(deps.HealthDatabase().Db)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricsExp)),
	)
	otel.SetMeterProvider(mp)

	// Start collecting runtime metrics (Go GC, memory, goroutines, etc.)
	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to start runtime metrics: %w", err)
	}

	return mp, nil
}
