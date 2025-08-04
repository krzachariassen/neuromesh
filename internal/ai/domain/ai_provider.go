package domain

import (
	"context"
)

// AIProvider defines the core domain interface for AI inference
// This is a pure domain interface - no infrastructure concerns
type AIProvider interface {
	// CallAI performs AI inference with system and user prompts
	CallAI(ctx context.Context, systemPrompt, userPrompt string) (string, error)

	// CallAIWithConversation performs AI inference with full conversation history
	// This is essential for clarification flows where AI needs context
	CallAIWithConversation(ctx context.Context, conversation []*AIConversationMessage) (string, error)

	// GetProviderInfo returns metadata about the provider
	GetProviderInfo() *ProviderInfo

	// Close releases provider resources
	Close() error
}

// AIConversationMessage represents a single message in a conversation for AI context
// This maps directly to OpenAI's message format for conversation history
type AIConversationMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // The message content
}

// NewAIConversationMessage creates a new conversation message for AI context
func NewAIConversationMessage(role, content string) *AIConversationMessage {
	return &AIConversationMessage{
		Role:    role,
		Content: content,
	}
}

// ConversationAIRequest represents a request with full conversation history
type ConversationAIRequest struct {
	Messages   []*AIConversationMessage `json:"messages"`
	Parameters map[string]interface{}   `json:"parameters,omitempty"`
}

// NewConversationAIRequest creates a new conversation-aware AI request
func NewConversationAIRequest(messages []*AIConversationMessage) *ConversationAIRequest {
	return &ConversationAIRequest{
		Messages:   messages,
		Parameters: make(map[string]interface{}),
	}
}

// ProviderInfo contains metadata about an AI provider
type ProviderInfo struct {
	Name    string `json:"name"`    // Provider name (e.g., "openai", "ollama")
	Model   string `json:"model"`   // Model name (e.g., "gpt-4", "llama2")
	Version string `json:"version"` // Provider version
}

// AIRequest represents a request for AI inference
type AIRequest struct {
	SystemPrompt string                 `json:"system_prompt"`
	UserPrompt   string                 `json:"user_prompt"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// AIResponse represents the response from AI inference
type AIResponse struct {
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence,omitempty"`
	Model      string  `json:"model"`
	TokensUsed int     `json:"tokens_used,omitempty"`
}

// NewAIRequest creates a new AI request
func NewAIRequest(systemPrompt, userPrompt string) *AIRequest {
	return &AIRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Parameters:   make(map[string]interface{}),
	}
}

// WithParameter adds a parameter to the AI request
func (r *AIRequest) WithParameter(key string, value interface{}) *AIRequest {
	if r.Parameters == nil {
		r.Parameters = make(map[string]interface{})
	}
	r.Parameters[key] = value
	return r
}
