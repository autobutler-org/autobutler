package purchases

import (
	"github.com/gin-gonic/gin"
)

// TODO update this to use the actual product types
type PurchaseItem struct {
	ProductType  string `json:"productType"`
	Description  string `json:"description"`
	Amount       int    `json:"amount"`
}

type Purchase struct {
	ID       string         `json:"id"`
	Amount   int           `json:"amount"`
	Currency string        `json:"currency"`
	Status   string        `json:"status"`
	Items    []PurchaseItem `json:"items"`
}

func GetPurchases(c *gin.Context) {
	defaultPurchases := []Purchase{
		{
			ID:       "purchase_123",
			Amount:   35900,
			Currency: "usd",
			Status:   "completed",
			Items: []PurchaseItem{
				{
					ProductType:  "base_product",
					Description: "Base Software License",
					Amount:     35900,
				},
			},
		},
	}

	c.JSON(200, defaultPurchases)
}

func SetupRoutes(router *gin.Engine) {
	router.GET("/purchases", GetPurchases)
}
