package main

import (
	"encoding/json"
	"os"
)

// Config holds all application configuration
type Config struct {
	ServerPort   string
	LLMServerURL string
	URLs         map[string]string
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	// Load URLs from file
	urls := loadURLs()

	// Get LLM server URL from environment or use default from URLs
	llmServerURL := getEnvOrDefault("LLM_URL", urls["llm_server"])

	return &Config{
		ServerPort:   getEnvOrDefault("PORT", "8080"),
		LLMServerURL: llmServerURL,
		URLs:         urls,
	}
}

// loadURLs reads the URLs from the urls.json file
func loadURLs() map[string]string {
	// Default URLs in case file can't be loaded
	defaultURLs := map[string]string{
		"chat":       "/api/chat",
		"health":     "/health",
		"dummy":      "/api/dummy",
		"llm_server": "http://127.0.0.1:8081/generate",
	}

	// Try to read the file
	data, err := os.ReadFile("urls.json")
	if err != nil {
		return defaultURLs
	}

	// Parse the JSON
	var config struct {
		URLs map[string]string `json:"urls"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return defaultURLs
	}

	return config.URLs
}

// getEnvOrDefault returns the value of the environment variable or the default value if not set
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
