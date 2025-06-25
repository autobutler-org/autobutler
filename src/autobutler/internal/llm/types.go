package llm

import (
	"fmt"
	"time"

	"github.com/openai/openai-go"
)

type ChatRole string

const (
	ChatRoleUser   ChatRole = "user"
	ChatRoleSystem ChatRole = "system"
	ChatRoleDummy  ChatRole = "dummy"
	ChatRoleError  ChatRole = "error"
)

type ChatMessage struct {
	Role      ChatRole `json:"role"`
	Content   string   `json:"content"`
	Timestamp string   `json:"timestamp"` // ISO 8601 format
}

func ErrorChatMessage(err error) ChatMessage {
	return ChatMessage{
		Role:      ChatRoleError,
		Content:   fmt.Sprintf("An error occurred while processing your request: %v", err),
		Timestamp: GetTimestamp(time.Now()),
	}
}

func GetTimestamp(timestamp time.Time) string {
	// Matches JS new Date().toLocaleTimeString()
	return timestamp.Format("3:04:05 PM")
}

func FromCompletionToChatMessage(completion openai.ChatCompletion) ChatMessage {
	if len(completion.Choices) == 0 {
		return ChatMessage{
			Role:      ChatRoleError,
			Content:   "",
			Timestamp: GetTimestamp(time.Now()),
		}
	}
	return ChatMessage{
		Role:    ChatRoleSystem,
		Content: completion.Choices[0].Message.Content,
		// Matches JS new Date().toLocaleTimeString()
		Timestamp: GetTimestamp(time.Now()),
	}
}
