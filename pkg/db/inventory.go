package db

import (
	"context"
	"fmt"
)

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
