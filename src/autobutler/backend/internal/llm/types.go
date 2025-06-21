package llm

import "time"

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

func ErrorChatMessage() ChatMessage {
	return ChatMessage{
		Role:      ChatRoleError,
		Content:   "An error occurred while processing your request",
		Timestamp: GetTimestamp(time.Now()),
	}
}

func GetTimestamp(timestamp time.Time) string {
	// Matches JS new Date().toLocaleTimeString()
	return timestamp.Format("3:04:05 PM")
}

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

func (r ChatResponse) ToChatMessage() ChatMessage {
	if len(r.Choices) == 0 {
		return ChatMessage{
			Role:      ChatRoleError,
			Content:   "",
			Timestamp: GetTimestamp(time.Now()),
		}
	}
	return ChatMessage{
		Role:    ChatRoleSystem,
		Content: r.Choices[0].Message.Content,
		// Matches JS new Date().toLocaleTimeString()
		Timestamp: GetTimestamp(time.Now()),
	}
}
