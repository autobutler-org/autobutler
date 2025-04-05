package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

const (
	PI_SERVER_URL = "http://localhost:8081/generate"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Set up Gin router
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Chat endpoint
	r.POST("/api/chat", handleChat)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Get port from environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Forward request to Pi's TinyLlama server
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error marshaling request: %v", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	resp, err := client.Post(PI_SERVER_URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error forwarding request to Pi: %v", err)
		c.JSON(503, gin.H{"error": "TinyLlama service unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var piResponse ChatResponse
	if err := json.Unmarshal(body, &piResponse); err != nil {
		log.Printf("Error unmarshaling response: %v", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, piResponse)
}
