package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

func main() {
	// Load configuration
	cfg := NewConfig()

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

	r.POST(cfg.URLs["chat"], func(c *gin.Context) {
		handleChat(c, cfg)
	})

	// Dummy endpoint for testing
	r.POST(cfg.URLs["dummy"], func(c *gin.Context) {
		handleDummy(c)
	})

	// Health check endpoint
	r.GET(cfg.URLs["health"], func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// handleDummy returns a simple "Hello World" response for testing
func handleDummy(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Simulate a delay to mimic LLM processing
	time.Sleep(500 * time.Millisecond)

	// Return a dummy response
	response := ChatResponse{
		Response: "Hello World! This is a dummy response from the backend. Your message was: " + req.Message,
	}

	c.JSON(200, response)
}

func handleChat(c *gin.Context, cfg *Config) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Forward request to LLM server
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error marshaling request: %v", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	// Use the LLM server URL from config
	log.Printf("Forwarding request to LLM server at %s", cfg.LLMServerURL)
	resp, err := client.Post(cfg.LLMServerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error forwarding request to LLM: %v", err)
		c.JSON(503, gin.H{"error": "LLM service unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var llmResponse ChatResponse
	if err := json.Unmarshal(body, &llmResponse); err != nil {
		log.Printf("Error unmarshaling response: %v", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, llmResponse)
}
