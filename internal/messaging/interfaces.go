package messaging

import (
	"context"
	"encoding/json"
	"time"
)

// DomainEvent represents a structured domain event with metadata
type DomainEvent struct {
	EventType string          `json:"event_type"`
	EventData json.RawMessage `json:"event_data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time       `json:"timestamp"`
}

// UnmarshalEventData unmarshals the event data into the provided struct
func (de *DomainEvent) UnmarshalEventData(v interface{}) error {
	return json.Unmarshal(de.EventData, v)
}

// DomainEventBus handles domain events with clean abstraction over infrastructure
type DomainEventBus interface {
	// PublishDomainEvent publishes a domain event with clean interface
	PublishDomainEvent(ctx context.Context, eventType string, event interface{}) error
	
	// SubscribeToDomainEvents subscribes to domain events matching pattern
	SubscribeToDomainEvents(ctx context.Context, subscriberID, eventPattern string) (<-chan *DomainEvent, error)
	
	// Close cleans up resources
	Close() error
}

// MessageBus handles natural language communication between AI, agents, and users
// This is the event-driven messaging system for conversational orchestration
type MessageBus interface {
	// Send message to specific recipient
	SendMessage(ctx context.Context, message *Message) error

	// Subscribe to messages for a specific participant
	Subscribe(ctx context.Context, participantID string) (<-chan *Message, error)

	// Unsubscribe from messages
	Unsubscribe(ctx context.Context, participantID string) error

	// Publish message to multiple recipients (broadcast)
	PublishMessage(ctx context.Context, message *Message, recipients []string) error

	// Get conversation history
	GetConversationHistory(ctx context.Context, conversationID string) ([]*Message, error)

	// Create new conversation context
	CreateConversation(ctx context.Context, participants []string, context map[string]interface{}) (*ConversationContext, error)

	// PrepareAgentQueue ensures queue and routing are set up for an agent without starting consumption
	PrepareAgentQueue(ctx context.Context, agentID string) error
	
	// Domain event support - extend existing MessageBus with domain events
	DomainEventBus
}

// MessageHandler handles incoming messages
type MessageHandler interface {
	// Handle incoming message
	HandleMessage(ctx context.Context, message *Message) error

	// Get handler ID for routing
	GetHandlerID() string
}

// ClarificationRequest represents a request for clarification
type ClarificationRequest struct {
	RequestID     string                 `json:"request_id"`
	AgentID       string                 `json:"agent_id"`
	Question      string                 `json:"question"`
	Context       map[string]interface{} `json:"context"`
	CorrelationID string                 `json:"correlation_id"`
}

// ClarificationResponse represents the response to a clarification
type ClarificationResponse struct {
	RequestID     string                 `json:"request_id"`
	Answer        string                 `json:"answer"`
	Context       map[string]interface{} `json:"context"`
	CorrelationID string                 `json:"correlation_id"`
}
