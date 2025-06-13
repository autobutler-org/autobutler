package profile

import (
	"github.com/gin-gonic/gin"
)

type Profile struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	Phone   string `json:"phone"`
}

func GetProfile(c *gin.Context) {
	defaultProfile := Profile{
		Name:    "Default User",
		Email:   "user@example.com", 
		Company: "Example Corp",
		Phone:   "+1-555-555-5555",
	}

	c.JSON(200, defaultProfile)
}

func SetupRoutes(router *gin.Engine) {
	router.GET("/profile", GetProfile)
}
