package db

import (
	"context"
	"database/sql"
	"fmt"
)

func NewInventoryFromRows(rows *sql.Rows) ([]*Inventory, error) {
	var items []*Inventory
	for rows.Next() {
		var item Inventory
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

func (d *Database) QueryInventory(id int) (*Inventory, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := "SELECT * FROM inventory WHERE id = ?"
	rows, err := d.Db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %s", query)
	}
	defer rows.Close()
	items, err := NewInventoryFromRows(rows)
	if err != nil {
		return nil, fmt.Errorf("error creating items from rows: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (d *Database) QueryInventoryByName(name string) (*Inventory, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := "SELECT * FROM inventory WHERE name = ?"
	rows, err := d.Db.Query(query, name)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %s", query)
	}
	defer rows.Close()
	items, err := NewInventoryFromRows(rows)
	if err != nil {
		return nil, fmt.Errorf("error creating items from rows: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (d *Database) UpsertInventory(newInventory Inventory) (*Inventory, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	// Start a transaction
	inventory, err := DatabaseQueries.GetInventory(context.Background(), newInventory.ID)
	if err == nil {
		// Item exists, update it
		inventory.Amount += newInventory.Amount
		inventory, err = DatabaseQueries.UpdateInventory(
			context.Background(),
			UpdateInventoryParams{
				ID:     inventory.ID,
				Name:   inventory.Name,
				Amount: inventory.Amount,
				Unit:   inventory.Unit,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error updating item: %w", err)
		}
	} else {
		// Item does not exist, insert it
		inventory, err = DatabaseQueries.CreateInventory(
			context.Background(),
			CreateInventoryParams{
				Name:   inventory.Name,
				Amount: inventory.Amount,
				Unit:   inventory.Unit,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error inserting item: %w", err)
		}
		newInventory.ID = inventory.ID
	}
	return &inventory, nil
}
