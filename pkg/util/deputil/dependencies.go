package deputil

import (
	"autobutler/internal/db"
	"fmt"
)

type Dependencies interface {
	Database() *db.DatabaseSqlc
	HealthDatabase() *db.DatabaseRaw
	WithDatabase(database *db.DatabaseSqlc) Dependencies
	WithHealthDatabase(healthDatabase *db.DatabaseRaw) Dependencies
}

type dependencies struct {
	database       *db.DatabaseSqlc
	healthDatabase *db.DatabaseRaw
}

func NewDependencies() Dependencies {
	return &dependencies{}
}

func DefaultDependencies() (Dependencies, error) {
	deps := NewDependencies()
	if database, err := db.ConnectToDatabase(); err == nil {
		deps.WithDatabase(database)
	} else {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if database, err := db.ConnectToHealthDatabase(); err == nil {
		deps.WithHealthDatabase(database)
	} else {
		return nil, fmt.Errorf("failed to connect to health database: %w", err)
	}
	return deps, nil
}

func (d *dependencies) WithDatabase(database *db.DatabaseSqlc) Dependencies {
	d.database = database
	return d
}

func (d *dependencies) WithHealthDatabase(database *db.DatabaseRaw) Dependencies {
	d.healthDatabase = database
	return d
}

func (d *dependencies) Database() *db.DatabaseSqlc {
	return d.database
}

func (d *dependencies) HealthDatabase() *db.DatabaseRaw {
	return d.healthDatabase
}
