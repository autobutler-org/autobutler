package llm

import (
	"encoding/json"
	"fmt"
)

func ParseResponseString(response string) (*ChatResponse, error) {
	return ParseResponseBytes([]byte(response))
}

func ParseResponseBytes(response []byte) (*ChatResponse, error) {
	var chatResponse ChatResponse
	err := json.Unmarshal(response, &chatResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(chatResponse.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned in response")
	}
	return &chatResponse, nil
}

func GetResponseText(response *ChatResponse) (string, error) {
	if response == nil || len(response.Choices) == 0 {
		return "", fmt.Errorf("invalid response: no choices available")
	}
	if response.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no content in the first choice message")
	}
	return response.Choices[0].Message.Content, nil
}

func DoChat(prompt string) (string, error) {
	response, err := RemoteLLMRequest(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to get response from LLM: %w", err)
	}
	text, err := GetResponseText(response)
	if err != nil {
		return "", fmt.Errorf("failed to get response text: %w", err)
	}
	return text, nil
}
