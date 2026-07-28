package botelsqlite

import (
	"errors"
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// MetricsExporter is a SQLite-based metrics exporter that implements the metric.Exporter interface
type MetricsExporter struct {
	db *sql.DB
}

// NewMetricsExporter creates a new SQLite metrics exporter
func NewMetricsExporter(db *sql.DB) (*MetricsExporter, error) {
	if db == nil {
		return nil, errors.New("database cannot be nil")
	}

	exporter := &MetricsExporter{
		db: db,
	}

	// Initialize the database schema for metrics
	if err := exporter.initMetricsSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics schema: %w", err)
	}

	return exporter, nil
}

// initMetricsSchema creates the necessary tables for storing metrics
func (e *MetricsExporter) initMetricsSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		unit TEXT,
		type TEXT NOT NULL, -- 'counter', 'gauge', 'histogram'
		timestamp INTEGER NOT NULL,
		value REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
	CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics(type);

	CREATE TABLE IF NOT EXISTS metric_attributes (
		metric_id INTEGER NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		FOREIGN KEY (metric_id) REFERENCES metrics(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_metric_attributes_metric_id ON metric_attributes(metric_id);
	CREATE INDEX IF NOT EXISTS idx_metric_attributes_key ON metric_attributes(key);

	CREATE TABLE IF NOT EXISTS metric_histogram_buckets (
		metric_id INTEGER NOT NULL,
		bucket_boundary REAL NOT NULL,
		count INTEGER NOT NULL,
		FOREIGN KEY (metric_id) REFERENCES metrics(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_metric_histogram_buckets_metric_id ON metric_histogram_buckets(metric_id);

	CREATE TABLE IF NOT EXISTS metric_exemplars (
		metric_id INTEGER NOT NULL,
		value REAL NOT NULL,
		timestamp INTEGER NOT NULL,
		trace_id TEXT,
		span_id TEXT,
		FOREIGN KEY (metric_id) REFERENCES metrics(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_metric_exemplars_metric_id ON metric_exemplars(metric_id);
	`

	_, err := e.db.Exec(schema)
	return err
}

// Temporality returns the Temporality to use for an instrument kind
func (e *MetricsExporter) Temporality(kind metric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

// Aggregation returns the Aggregation to use for an instrument kind
func (e *MetricsExporter) Aggregation(kind metric.InstrumentKind) metric.Aggregation {
	return metric.DefaultAggregationSelector(kind)
}

// Export exports metrics data to SQLite
func (e *MetricsExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	if rm == nil {
		return nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if err := e.exportMetric(ctx, tx, m); err != nil {
				return fmt.Errorf("failed to export metric: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// exportMetric exports a single metric to the database
func (e *MetricsExporter) exportMetric(ctx context.Context, tx *sql.Tx, m metricdata.Metrics) error {
	var metricType string
	var err error

	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		metricType = "counter"
		err = e.exportSum(ctx, tx, m.Name, m.Description, m.Unit, data)
	case metricdata.Sum[float64]:
		metricType = "counter"
		err = e.exportSumFloat(ctx, tx, m.Name, m.Description, m.Unit, data)
	case metricdata.Gauge[int64]:
		metricType = "gauge"
		err = e.exportGauge(ctx, tx, m.Name, m.Description, m.Unit, data)
	case metricdata.Gauge[float64]:
		metricType = "gauge"
		err = e.exportGaugeFloat(ctx, tx, m.Name, m.Description, m.Unit, data)
	case metricdata.Histogram[int64]:
		metricType = "histogram"
		err = e.exportHistogram(ctx, tx, m.Name, m.Description, m.Unit, data)
	case metricdata.Histogram[float64]:
		metricType = "histogram"
		err = e.exportHistogramFloat(ctx, tx, m.Name, m.Description, m.Unit, data)
	default:
		return fmt.Errorf("unsupported metric type: %T", data)
	}

	if err != nil {
		return fmt.Errorf("failed to export %s metric: %w", metricType, err)
	}

	return nil
}

// exportSum exports counter/sum metrics (int64)
func (e *MetricsExporter) exportSum(ctx context.Context, tx *sql.Tx, name, description, unit string, data metricdata.Sum[int64]) error {
	for _, dp := range data.DataPoints {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO metrics (name, description, unit, type, timestamp, value)
			VALUES (?, ?, ?, 'counter', ?, ?)
		`, name, description, unit, dp.Time.UnixNano(), float64(dp.Value))
		if err != nil {
			return err
		}

		metricID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		if err := e.insertAttributes(ctx, tx, metricID, dp.Attributes); err != nil {
			return err
		}
	}
	return nil
}

// exportSumFloat exports counter/sum metrics (float64)
func (e *MetricsExporter) exportSumFloat(ctx context.Context, tx *sql.Tx, name, description, unit string, data metricdata.Sum[float64]) error {
	for _, dp := range data.DataPoints {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO metrics (name, description, unit, type, timestamp, value)
			VALUES (?, ?, ?, 'counter', ?, ?)
		`, name, description, unit, dp.Time.UnixNano(), dp.Value)
		if err != nil {
			return err
		}

		metricID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		if err := e.insertAttributes(ctx, tx, metricID, dp.Attributes); err != nil {
			return err
		}
	}
	return nil
}

// exportGauge exports gauge metrics (int64)
func (e *MetricsExporter) exportGauge(ctx context.Context, tx *sql.Tx, name, description, unit string, data metricdata.Gauge[int64]) error {
	for _, dp := range data.DataPoints {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO metrics (name, description, unit, type, timestamp, value)
			VALUES (?, ?, ?, 'gauge', ?, ?)
		`, name, description, unit, dp.Time.UnixNano(), float64(dp.Value))
		if err != nil {
			return err
		}

		metricID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		if err := e.insertAttributes(ctx, tx, metricID, dp.Attributes); err != nil {
			return err
		}
	}
	return nil
}

// exportGaugeFloat exports gauge metrics (float64)
func (e *MetricsExporter) exportGaugeFloat(ctx context.Context, tx *sql.Tx, name, description, unit string, data metricdata.Gauge[float64]) error {
	for _, dp := range data.DataPoints {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO metrics (name, description, unit, type, timestamp, value)
			VALUES (?, ?, ?, 'gauge', ?, ?)
		`, name, description, unit, dp.Time.UnixNano(), dp.Value)
		if err != nil {
			return err
		}

		metricID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		if err := e.insertAttributes(ctx, tx, metricID, dp.Attributes); err != nil {
			return err
		}
	}
	return nil
}

// exportHistogram exports histogram metrics (int64)
func (e *MetricsExporter) exportHistogram(ctx context.Context, tx *sql.Tx, name, description, unit string, data metricdata.Histogram[int64]) error {
	for _, dp := range data.DataPoints {
		// Store the sum as the main value
		result, err := tx.ExecContext(ctx, `
			INSERT INTO metrics (name, description, unit, type, timestamp, value)
			VALUES (?, ?, ?, 'histogram', ?, ?)
		`, name, description, unit, dp.Time.UnixNano(), float64(dp.Sum))
		if err != nil {
			return err
		}

		metricID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		if err := e.insertAttributes(ctx, tx, metricID, dp.Attributes); err != nil {
			return err
		}

		// Store bucket counts
		if err := e.insertHistogramBuckets(ctx, tx, metricID, dp.Bounds, dp.BucketCounts); err != nil {
			return err
		}
	}
	return nil
}

// exportHistogramFloat exports histogram metrics (float64)
func (e *MetricsExporter) exportHistogramFloat(ctx context.Context, tx *sql.Tx, name, description, unit string, data metricdata.Histogram[float64]) error {
	for _, dp := range data.DataPoints {
		// Store the sum as the main value
		result, err := tx.ExecContext(ctx, `
			INSERT INTO metrics (name, description, unit, type, timestamp, value)
			VALUES (?, ?, ?, 'histogram', ?, ?)
		`, name, description, unit, dp.Time.UnixNano(), dp.Sum)
		if err != nil {
			return err
		}

		metricID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		if err := e.insertAttributes(ctx, tx, metricID, dp.Attributes); err != nil {
			return err
		}

		// Store bucket counts
		if err := e.insertHistogramBuckets(ctx, tx, metricID, dp.Bounds, dp.BucketCounts); err != nil {
			return err
		}
	}
	return nil
}

// insertAttributes inserts metric attributes
func (e *MetricsExporter) insertAttributes(ctx context.Context, tx *sql.Tx, metricID int64, attrs attribute.Set) error {
	iter := attrs.Iter()
	for iter.Next() {
		attr := iter.Attribute()
		valueType := attr.Value.Type()
		value := attr.Value.AsString()
		switch valueType {
		case attribute.INT64:
			value = strconv.Itoa(int(attr.Value.AsInt64()))
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO metric_attributes (metric_id, key, value)
			VALUES (?, ?, ?)
		`, metricID, string(attr.Key), value)
		if err != nil {
			return err
		}
	}
	return nil
}

// insertHistogramBuckets inserts histogram bucket data
func (e *MetricsExporter) insertHistogramBuckets(ctx context.Context, tx *sql.Tx, metricID int64, bounds []float64, counts []uint64) error {
	for i, count := range counts {
		var boundary float64
		if i < len(bounds) {
			boundary = bounds[i]
		} else {
			boundary = -1 // Infinity bucket
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO metric_histogram_buckets (metric_id, bucket_boundary, count)
			VALUES (?, ?, ?)
		`, metricID, boundary, count)
		if err != nil {
			return err
		}
	}
	return nil
}

// ForceFlush flushes any pending metrics
func (e *MetricsExporter) ForceFlush(ctx context.Context) error {
	return nil
}

// Shutdown shuts down the exporter
func (e *MetricsExporter) Shutdown(ctx context.Context) error {
	return nil
}
