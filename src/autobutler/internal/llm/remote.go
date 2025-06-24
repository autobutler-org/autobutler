package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/openai/openai-go"
)

var (
	addFn *openai.FunctionDefinitionParam
)
func add(x float64, y float64) float64 {
	return x + y
}

func init() {
	var err error
	addFn, err = GenerateJSONSchema(add, "Adds two numbers together and returns the result.")
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for add function: %v", err))
	}
}

func RemoteLLMRequest(prompt string) (*openai.ChatCompletion, error) {
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
	systemPrompt := os.Getenv("LLM_SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = SYSTEM_PROMPT
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
		model = "autobutler_gpt-4.1-nano"
	}

	maxTokensInt := 2048
	fmt.Sscanf(maxTokens, "%d", &maxTokensInt)
	temperatureFloat := 0.8
	fmt.Sscanf(temperature, "%f", &temperatureFloat)
	topPFloat := 0.1
	fmt.Sscanf(topP, "%f", &topPFloat)

	reqBody := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.AssistantMessage(string(systemPrompt)),
			openai.UserMessage(prompt),
		},
		MaxTokens:   openai.Int(int64(maxTokensInt)),
		Temperature: openai.Float(temperatureFloat),
		TopP:        openai.Float(topPFloat),
		Model:       model,
		Tools: []openai.ChatCompletionToolParam{
			{
				Function: *addFn,
			},
		},
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
