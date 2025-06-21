package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)


func RemoteLLMRequest(prompt string) (*ChatResponse, error) {
	// Set defaults as per Makefile exports
	llmURL := os.Getenv("LLM_URL")
	if llmURL == "" {
		llmURL = "https://autobutler-eus2.services.ai.azure.com/models/chat/completions"
	}
	llmArgs := os.Getenv("LLM_ARGS")
	if llmArgs == "" {
		llmArgs = "api-version=2024-05-01-preview"
	}
	apiKey := os.Getenv("LLM_AZURE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LLM_AZURE_API_KEY environment variable is not set")
	}
	systemPromptFile := os.Getenv("LLM_SYSTEM_PROMPT_FILE")
	if systemPromptFile == "" {
		systemPromptFile = "system.prompt"
	}
	maxTokens := os.Getenv("LLM_MAX_TOKENS")
	if maxTokens == "" {
		maxTokens = "2048"
	}
	temperature := os.Getenv("LLM_TEMP")
	if temperature == "" {
		temperature = "0.8"
	}
	topP := os.Getenv("LLM_TOP_P")
	if topP == "" {
		topP = "0.1"
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "autobutler_Ministral-3B"
	}

	systemPrompt, err := os.ReadFile(systemPromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read system prompt file: %w", err)
	}

	maxTokensInt := 2048
	fmt.Sscanf(maxTokens, "%d", &maxTokensInt)
	temperatureFloat := 0.8
	fmt.Sscanf(temperature, "%f", &temperatureFloat)
	topPFloat := 0.1
	fmt.Sscanf(topP, "%f", &topPFloat)

	reqBody := requestBody{
		Messages: []message{
			{Role: "system", Content: string(systemPrompt)},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   maxTokensInt,
		Temperature: temperatureFloat,
		TopP:        topPFloat,
		Model:       model,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := llmURL
	if llmArgs != "" {
		url = fmt.Sprintf("%s?%s", llmURL, llmArgs)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	return ParseResponseBytes(respBody)
}
