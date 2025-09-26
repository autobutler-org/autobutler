package mcp

import (
	"autobutler/pkg/db"
	"context"
	"fmt"
)

type UpsertInventoryParams struct {
	Name   string  `json:"param0"`
	Amount float64 `json:"param1"`
	Unit   string  `json:"param2"`
}

func (p UpsertInventoryParams) Output(response any) (string, []any) {
	resp := response.(UpsertInventoryResponse)
	return "Added %f %s of %s to the inventory, so now you have %f %s.", []any{
		p.Amount,
		resp.Inventory.Unit,
		resp.Inventory.Name,
		resp.Inventory.Amount,
		resp.Inventory.Unit,
	}
}

type UpsertInventoryResponse struct {
	Inventory db.Inventory `json:"inventory"`
}

func (r McpRegistry) UpsertInventory(name string, amount float64, unit string) UpsertInventoryResponse {
	inventory, err := db.Instance.UpsertInventory(db.Inventory{
		Name:   name,
		Amount: amount,
		Unit:   unit,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to add to inventory the item: %v", err))
	}
	return UpsertInventoryResponse{
		Inventory: *inventory,
	}
}

type QueryInventoryParams struct {
	ItemName string `json:"param0"`
}

func (p QueryInventoryParams) Output(response any) (string, []any) {
	resp := response.(QueryInventoryResponse)
	if resp.Inventory == 0 {
		return "You have no %s.", []any{resp.Item}
	}
	if resp.Inventory < 0 {
		return "You have a negative inventory of %f %s of %s, which is unusual.", []any{resp.Inventory, resp.Unit, resp.Item}
	}
	return "There are %f %s of %s in the inventory.", []any{resp.Inventory, resp.Unit, resp.Item}
}

type QueryInventoryResponse struct {
	Item      string  `json:"item"`
	Inventory float64 `json:"inventory"`
	Unit      string  `json:"unit,omitempty"`
}

func (r McpRegistry) QueryInventory(itemName string) QueryInventoryResponse {
	item, err := db.DatabaseQueries.GetInventoryByName(
		context.Background(),
		itemName,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to query inventory for item %s: %v", itemName, err))
	}
	return QueryInventoryResponse{
		Item:      item.Name,
		Inventory: float64(item.Amount),
		Unit:      item.Unit,
	}
}

type ReduceInventoryParams struct {
	Name   string  `json:"param0"`
	Amount float64 `json:"param1"`
	Unit   string  `json:"param2"`
}

func (p ReduceInventoryParams) Output(response any) (string, []any) {
	resp := response.(ReduceInventoryResponse)
	return "Reduced %f %s of %s from the inventory, so now you have %f %s.", []any{
		p.Amount,
		resp.Inventory.Unit,
		resp.Inventory.Name,
		resp.Inventory.Amount,
		resp.Inventory.Unit,
	}
}

type ReduceInventoryResponse struct {
	Inventory db.Inventory `json:"inventory"`
}

func (r McpRegistry) ReduceInventory(name string, amount float64, unit string) ReduceInventoryResponse {
	response := r.UpsertInventory(name, -amount, unit)
	return ReduceInventoryResponse(response)
}
