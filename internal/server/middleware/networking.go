package middleware

import (
	"autobutler/pkg/networking"

	"github.com/gin-gonic/gin"
)

func NetworkingNode(node *networking.Node) gin.HandlerFunc {
	return func(c *gin.Context) {
		if node != nil {
			c.Set("networking_node", node)
		}
		c.Next()
	}
}
