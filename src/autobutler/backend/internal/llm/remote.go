package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type requestBody struct {
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	TopP        float64   `json:"top_p"`
	Model       string    `json:"model"`
}

type responseChoice struct {
	ContentFilterResults struct {
		Hate struct {
			Filtered bool   `json:"filtered"`
			Severity string `json:"severity"`
		} `json:"hate"`
		ProtectedMaterialCode struct {
			Filtered bool `json:"filtered"`
			Detected bool `json:"detected"`
		} `json:"protected_material_code"`
		ProtectedMaterialText struct {
			Filtered bool `json:"filtered"`
			Detected bool `json:"detected"`
		} `json:"protected_material_text"`
		SelfHarm struct {
			Filtered bool   `json:"filtered"`
			Severity string `json:"severity"`
		} `json:"self_harm"`
		Sexual struct {
			Filtered bool   `json:"filtered"`
			Severity string `json:"severity"`
		} `json:"sexual"`
		Violence struct {
			Filtered bool   `json:"filtered"`
			Severity string `json:"severity"`
		} `json:"violence"`
	} `json:"content_filter_results"`
	FinishReason string `json:"finish_reason"`
	Index        int    `json:"index"`
	Message      struct {
		Content   string      `json:"content"`
		Role      string      `json:"role"`
		ToolCalls interface{} `json:"tool_calls"` // Assuming tool_calls can be null or an object
	} `json:"message"`
}
type ChatResponse struct {
	Choices             []responseChoice `json:"choices"`
	Created             int64            `json:"created"`
	ID                  string           `json:"id"`
	Model               string           `json:"model"`
	Object              string           `json:"object"`
	PromptFilterResults []struct {
		PromptIndex          int `json:"prompt_index"`
		ContentFilterResults struct {
			Hate struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"hate"`
			Jailbreak struct {
				Filtered bool `json:"filtered"`
				Detected bool `json:"detected"`
			} `json:"jailbreak"`
			SelfHarm struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"self_harm"`
			Sexual struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"sexual"`
			Violence struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"violence"`
		} `json:"content_filter_results"`
	} `json:"prompt_filter_results"`
	Usage struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

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
