package deputil

import (
	"autobutler/internal/db"
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
