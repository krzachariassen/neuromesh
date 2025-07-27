package websocket

import (
	"time"
)

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	MessageTypeChatMessage    MessageType = "chat_message"
	MessageTypeAgentUpdate    MessageType = "agent_update"
	MessageTypeExecutionStart MessageType = "execution_start"
	MessageTypeExecutionStep  MessageType = "execution_step"
	MessageTypeError          MessageType = "error"
	MessageTypeTyping         MessageType = "typing"
	MessageTypePing           MessageType = "ping"
	MessageTypePong           MessageType = "pong"
)

// Message represents a structured WebSocket message for the React UI
type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	SessionID string      `json:"session_id,omitempty"`
	Error     *ErrorData  `json:"error,omitempty"`
}

// ChatMessageData represents chat-specific data matching React UI types
type ChatMessageData struct {
	Content        string                 `json:"content"`
	Role           string                 `json:"role"` // 'user', 'assistant', 'system'
	ConversationID string                 `json:"conversation_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// AgentUpdateData represents agent status updates for AgentMonitor component
type AgentUpdateData struct {
	AgentName    string   `json:"agent_name"`
	Type         string   `json:"type"`
	Status       string   `json:"status"` // 'active', 'busy', 'idle', 'error'
	Capabilities []string `json:"capabilities,omitempty"`
	Metadata     struct {
		LastActive string `json:"last_active"`
	} `json:"metadata"`
}

// ExecutionStepData represents execution step updates for ExecutionVisualization
type ExecutionStepData struct {
	PlanID       string                 `json:"plan_id"`
	StepID       string                 `json:"step_id"`
	StepType     string                 `json:"step_type"`
	AgentName    string                 `json:"agent_name"`
	Status       string                 `json:"status"`
	Input        map[string]interface{} `json:"input,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// ErrorData represents error information in WebSocket messages
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TypingData represents typing indicator data
type TypingData struct {
	UserID   string `json:"user_id"`
	IsTyping bool   `json:"is_typing"`
}
