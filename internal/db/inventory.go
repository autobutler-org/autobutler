package db

import (
	"database/sql"
	"fmt"
)

type Item struct {
	ID     int
	Name   string
	Amount float64
	Unit   string
}

func NewItem(id int, name string, amount float64, unit string) *Item {
	return &Item{
		ID:     id,
		Name:   name,
		Amount: amount,
		Unit:   unit,
	}
}

func NewItemsFromRows(rows *sql.Rows) ([]*Item, error) {
	var items []*Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Amount, &item.Unit); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}
	return items, nil
}

func (d *Database) QueryInventory(id int) (*Item, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := "SELECT * FROM inventory WHERE id = ?"
	rows, err := d.Db.Query(query, id)
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

func (d *Database) QueryInventoryByName(name string) (*Item, error) {
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

func (d *Database) UpsertItem(newItem Item) (*Item, error) {
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
		item, err := d.QueryInventory(newItem.ID)
		if item != nil {
			// Item exists, update it
			item.Amount += newItem.Amount
			_, err = d.Exec(
				`UPDATE inventory SET
					amount = ?
						WHERE id = ?`,
				newItem.Amount,
				newItem.ID,
			)
			if err != nil {
				return nil, fmt.Errorf("error updating item: %w", err)
			}
		} else {
			// Item does not exist, insert it
			result, err := d.Exec(
				`INSERT INTO inventory (
					name,
					amount,
					unit
				) VALUES (
				 	?,
					?,
					?
				)`,
				newItem.Name,
				newItem.Amount,
				newItem.Unit,
			)
			if err != nil {
				return nil, fmt.Errorf("error inserting item: %w", err)
			}
			newId, err := result.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("error getting last insert id: %w", err)
			}
			newItem.ID = int(newId)
		}
	}
	item, err := d.QueryInventory(newItem.ID)
	if err != nil {
		return nil, fmt.Errorf("error querying inventory after insert/update: %w", err)
	}
	return item, nil
}
