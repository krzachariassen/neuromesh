package web

import (
	"time"
)

// Enhanced WebSocket Message Types for React UI Integration
type EnhancedWebSocketMessageType string

const (
	EnhancedMessageTypeChatMessage    EnhancedWebSocketMessageType = "chat_message"
	EnhancedMessageTypeAgentUpdate    EnhancedWebSocketMessageType = "agent_update"
	EnhancedMessageTypeExecutionStart EnhancedWebSocketMessageType = "execution_start"
	EnhancedMessageTypeExecutionStep  EnhancedWebSocketMessageType = "execution_step"
	EnhancedMessageTypeError          EnhancedWebSocketMessageType = "error"
	EnhancedMessageTypeTyping         EnhancedWebSocketMessageType = "typing"
	EnhancedMessageTypePing           EnhancedWebSocketMessageType = "ping"
	EnhancedMessageTypePong           EnhancedWebSocketMessageType = "pong"
)

// EnhancedWebSocketMessage represents a structured message for the React UI
// This extends the basic WebSocketMessage with rich typing and metadata
type EnhancedWebSocketMessage struct {
	Type      EnhancedWebSocketMessageType `json:"type"`
	ID        string                       `json:"id"`
	Timestamp time.Time                    `json:"timestamp"`
	Data      interface{}                  `json:"data"`
	SessionID string                       `json:"session_id,omitempty"`
	Error     *ErrorData                   `json:"error,omitempty"`
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

// ExecutionStartData represents execution plan start information
type ExecutionStartData struct {
	ExecutionID    string    `json:"execution_id"`
	ConversationID string    `json:"conversation_id"`
	PlanID         string    `json:"plan_id"`
	StartTime      time.Time `json:"start_time"`
	EstimatedSteps int       `json:"estimated_steps"`
}

// EnhancedExecutionStepData extends ExecutionStepData with real-time fields
type EnhancedExecutionStepData struct {
	ExecutionStepData            // Embed existing type
	ExecutionID       string     `json:"execution_id"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	Result            string     `json:"result,omitempty"`
}

// ErrorData represents error information in WebSocket messages
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TypingData represents typing indicator information
type TypingData struct {
	UserID    string `json:"user_id,omitempty"`
	IsTyping  bool   `json:"is_typing"`
	SessionID string `json:"session_id"`
}
