package deputil

import (
	"autobutler/internal/db"
	"autobutler/pkg/botel/exporters/botelsqlite"
	"testing"
)

func TestNewDependencies(t *testing.T) {
	deps := NewDependencies()
	if deps == nil {
		t.Fatal("NewDependencies() returned nil")
	}

	// Verify initial state
	if deps.Database() != nil {
		t.Error("Expected Database() to be nil initially")
	}
	if deps.HealthDatabase() != nil {
		t.Error("Expected HealthDatabase() to be nil initially")
	}
	if deps.MetricsExporter() != nil {
		t.Error("Expected MetricsExporter() to be nil initially")
	}
}

func TestWithDatabase(t *testing.T) {
	deps := NewDependencies()

	// Create a mock database (nil is fine for testing the setter/getter)
	var mockDB *db.DatabaseSqlc

	result := deps.WithDatabase(mockDB)

	// Verify it returns the same Dependencies instance (fluent interface)
	if result != deps {
		t.Error("WithDatabase() should return the same Dependencies instance")
	}

	// Verify the database was set
	if deps.Database() != mockDB {
		t.Error("Database() did not return the expected database")
	}
}

func TestWithHealthDatabase(t *testing.T) {
	deps := NewDependencies()

	// Create a mock health database
	var mockHealthDB *db.DatabaseRaw

	result := deps.WithHealthDatabase(mockHealthDB)

	// Verify it returns the same Dependencies instance (fluent interface)
	if result != deps {
		t.Error("WithHealthDatabase() should return the same Dependencies instance")
	}

	// Verify the health database was set
	if deps.HealthDatabase() != mockHealthDB {
		t.Error("HealthDatabase() did not return the expected database")
	}
}

func TestWithMetricsExporter(t *testing.T) {
	deps := NewDependencies()

	// Create a mock metrics exporter
	var mockExporter *botelsqlite.TraceExporter

	result := deps.WithMetricsExporter(mockExporter)

	// Verify it returns the same Dependencies instance (fluent interface)
	if result != deps {
		t.Error("WithMetricsExporter() should return the same Dependencies instance")
	}

	// Verify the metrics exporter was set
	if deps.MetricsExporter() != mockExporter {
		t.Error("MetricsExporter() did not return the expected exporter")
	}
}

func TestFluentChaining(t *testing.T) {
	deps := NewDependencies()

	var mockDB *db.DatabaseSqlc
	var mockHealthDB *db.DatabaseRaw
	var mockExporter *botelsqlite.TraceExporter

	// Test fluent chaining
	result := deps.
		WithDatabase(mockDB).
		WithHealthDatabase(mockHealthDB).
		WithMetricsExporter(mockExporter)

	// Verify all were set
	if result != deps {
		t.Error("Fluent chaining should return the same Dependencies instance")
	}
	if deps.Database() != mockDB {
		t.Error("Database was not set correctly during chaining")
	}
	if deps.HealthDatabase() != mockHealthDB {
		t.Error("HealthDatabase was not set correctly during chaining")
	}
	if deps.MetricsExporter() != mockExporter {
		t.Error("MetricsExporter was not set correctly during chaining")
	}
}
