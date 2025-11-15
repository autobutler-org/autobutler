package server

import (
	"autobutler/pkg/botel/exporters/botelsqlite"
	"autobutler/pkg/db"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var metricsExporter *botelsqlite.TraceExporter

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := botelsqlite.NewTraceExporter(db.HealthInstance.Db)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite exporter: %w", err)
	}

	// Store the exporter globally for metrics endpoint
	metricsExporter = exporter

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

func initMetrics() (*metric.MeterProvider, error) {
	metricsExp, err := botelsqlite.NewMetricsExporter(db.HealthInstance.Db)
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

func StartServer() error {
	tp, err := initTracer()
	if err != nil {
		return fmt.Errorf("failed to initialize otel trace: %w", err)
	}

	mp, err := initMetrics()
	if err != nil {
		return fmt.Errorf("failed to initialize otel metrics: %w", err)
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}()

	router := gin.Default()
	// IMPORTANT: UseMiddleware MUST be called before setupRoutes
	useMiddleware(router)
	setupRoutes(router)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		return err
	}

	return nil
}
