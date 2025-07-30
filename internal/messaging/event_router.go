package messaging

import (
	"context"
	"encoding/json"
	"time"
)

// EventRouter manages event-driven message routing for decoupled coordination
// Provides dedicated exchanges and routing keys for different event types
type EventRouter interface {
	// PublishEvent publishes an event to a specific exchange with routing key
	PublishEvent(ctx context.Context, exchange, routingKey string, event interface{}) error

	// SubscribeToEvents subscribes to events matching routing key pattern
	SubscribeToEvents(ctx context.Context, exchange, queueName, routingKey string) (<-chan EventMessage, error)

	// SetupEventExchange ensures an exchange exists for event routing
	SetupEventExchange(ctx context.Context, exchangeName string) error

	// Close cleans up event router resources
	Close() error
}

// EventMessage wraps event data with metadata
type EventMessage struct {
	Exchange      string                 `json:"exchange"`
	RoutingKey    string                 `json:"routing_key"`
	EventType     string                 `json:"event_type"`
	EventData     json.RawMessage        `json:"event_data"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AgentCompletedEvent represents an agent completion for synthesis coordination
type AgentCompletedEvent struct {
	PlanID    string    `json:"plan_id"`
	StepID    string    `json:"step_id"`
	AgentID   string    `json:"agent_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// SynthesisRequestEvent represents a synthesis coordination request
type SynthesisRequestEvent struct {
	PlanID        string    `json:"plan_id"`
	TriggerStepID string    `json:"trigger_step_id"`
	RequestedBy   string    `json:"requested_by"`
	Timestamp     time.Time `json:"timestamp"`
}
