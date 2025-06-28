package db

import (
	"database/sql"
	"fmt"
)

type Item struct {
	ID    int
	Name  string
	Amount float64
	Unit string
}

func NewItem(id int, name string, amount float64, unit string) *Item {
	return &Item{
		ID: id,
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


