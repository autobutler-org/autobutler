package mcp

import (
	"autobutler/internal/db"
	"fmt"
)

type AddToInventoryParams struct {
	Name   string  `json:"param0"`
	Amount float64 `json:"param1"`
	Unit   string  `json:"param2"`
}

func (p AddToInventoryParams) Output(response any) (string, []any) {
	resp := response.(AddToInventoryResponse)
	return "Added %f %s of %s to the inventory, so now you have %f %s.", []any{p.Amount, resp.Item.Unit, resp.Item.Name, resp.Item.Amount, resp.Item.Unit}
}

type AddToInventoryResponse struct {
	Item db.Item `json:"item"`
}

func (r McpRegistry) AddToInventory(name string, amount float64, unit string) AddToInventoryResponse {
	item, err := db.Instance.AddToInventory(db.Item{
		Name:   name,
		Amount: amount,
		Unit:   unit,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to add to inventory the item: %v", err))
	}
	return AddToInventoryResponse{
		Item: *item,
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
	item, err := db.Instance.QueryInventory(itemName)
	if err != nil {
		panic(fmt.Sprintf("failed to query inventory for item %s: %v", itemName, err))
	}
	if item == nil {
		return QueryInventoryResponse{
			Item:      itemName,
			Inventory: 0.0,
			Unit:      "",
		}
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
	return "Reduced %f %s of %s from the inventory, so now you have %f %s.", []any{p.Amount, resp.Item.Unit, resp.Item.Name, resp.Item.Amount, resp.Item.Unit}
}

type ReduceInventoryResponse struct {
	Item db.Item `json:"item"`
}

func (r McpRegistry) ReduceInventory(name string, amount float64, unit string) ReduceInventoryResponse {
	response := r.AddToInventory(name, -amount, unit)
	return ReduceInventoryResponse{
		Item: response.Item,
	}
}
