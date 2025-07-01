package db

import (
	"autobutler/pkg/util"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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

func (d *Database) QueryInventory(name string) (*Item, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := "SELECT * FROM inventory WHERE name = ?"
	rows, err := d.Db.Query(query, name)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %s", query)
	}
	defer rows.Close()
	items, err := NewItemsFromRows(rows)
	if err != nil {
		return nil, fmt.Errorf("error creating items from rows: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (d *Database) AddToInventory(newItem Item) (*Item, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	// Start a transaction
	{
		tx, err := d.Db.Begin()
		if err != nil {
			return nil, fmt.Errorf("error starting transaction: %w", err)
		}
		// Defer rollback or commit of transaction
		defer func() {
			if err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					fmt.Printf("error rolling back transaction: %v\n", rollbackErr)
				}
			} else {
				if commitErr := tx.Commit(); commitErr != nil {
					fmt.Printf("error committing transaction: %v\n", commitErr)
				}
			}
		}()
		item, err := d.QueryInventory(newItem.Name)
		if item != nil {
			// Item exists, update it
			item.Amount += newItem.Amount
			_, err = d.Exec("UPDATE inventory SET amount = ? WHERE id = ?", item.Amount, item.ID)
			if err != nil {
				return nil, fmt.Errorf("error updating item: %w", err)
			}
		} else {
			// Item does not exist, insert it
			_, err = d.Exec("INSERT INTO inventory (name, amount, unit) VALUES (?, ?, ?)", newItem.Name, newItem.Amount, newItem.Unit)
			if err != nil {
				return nil, fmt.Errorf("error inserting item: %w", err)
			}
		}
	}
	item, err := d.QueryInventory(newItem.Name)
	if err != nil {
		return nil, fmt.Errorf("error querying inventory after insert/update: %w", err)
	}
	return item, nil
}

func init() {
	dataDir := util.GetDataDir()
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		panic(fmt.Sprintf("failed to create data directory: %v", err))
	}

	dataFilePath := filepath.Join(dataDir, "autobutler.db")

	Instance.Db, err = sql.Open("sqlite", dataFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open database: %v", err))
	}

	initSchema()
}

func initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS inventory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		amount REAL NOT NULL,
		unit TEXT NOT NULL
	);
	`
	_, err := Instance.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return nil
}
