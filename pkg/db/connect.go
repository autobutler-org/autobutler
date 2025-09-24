package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

var (
	Instance Database
)

type Database struct {
	Db *sql.DB
}

func (d *Database) Exec(query string, args ...any) (sql.Result, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Db.Exec(query, args...)
}

func (d *Database) Query(query string, args ...any) (*sql.Rows, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Query(query, args...)
}
