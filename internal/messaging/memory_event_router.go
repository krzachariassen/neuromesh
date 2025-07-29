package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// MemoryEventRouter is an in-memory implementation of EventRouter for testing
type MemoryEventRouter struct {
	exchanges   map[string]bool
	subscribers map[string][]eventSubscription
	mu          sync.RWMutex
}

type eventSubscription struct {
	exchange   string
	queueName  string
	routingKey string
	channel    chan EventMessage
}

// NewMemoryEventRouter creates a new in-memory event router for testing
func NewMemoryEventRouter() *MemoryEventRouter {
	return &MemoryEventRouter{
		exchanges:   make(map[string]bool),
		subscribers: make(map[string][]eventSubscription),
	}
}

// SetupEventExchange ensures an exchange exists for event routing
func (r *MemoryEventRouter) SetupEventExchange(ctx context.Context, exchangeName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.exchanges[exchangeName] = true
	return nil
}

// PublishEvent publishes an event to a specific exchange with routing key
func (r *MemoryEventRouter) PublishEvent(ctx context.Context, exchange, routingKey string, event interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check if exchange exists
	if !r.exchanges[exchange] {
		return fmt.Errorf("exchange %s does not exist", exchange)
	}
	
	// Marshal event data
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	// Create event message
	eventMsg := EventMessage{
		Exchange:   exchange,
		RoutingKey: routingKey,
		EventType:  routingKey, // For simplicity, use routing key as event type
		EventData:  eventData,
		Metadata:   make(map[string]interface{}),
		Timestamp:  time.Now().UTC(),
	}
	
	// Send to matching subscribers
	for _, subscriptions := range r.subscribers {
		for _, sub := range subscriptions {
			if sub.exchange == exchange && r.routingKeyMatches(sub.routingKey, routingKey) {
				select {
				case sub.channel <- eventMsg:
				case <-ctx.Done():
					return ctx.Err()
				default:
					// Channel full, drop message (in production, this would be handled differently)
				}
			}
		}
	}
	
	return nil
}

// SubscribeToEvents subscribes to events matching routing key pattern
func (r *MemoryEventRouter) SubscribeToEvents(ctx context.Context, exchange, queueName, routingKey string) (<-chan EventMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if exchange exists
	if !r.exchanges[exchange] {
		return nil, fmt.Errorf("exchange %s does not exist", exchange)
	}
	
	// Create channel for events
	eventChan := make(chan EventMessage, 100) // Buffered channel
	
	// Create subscription
	subscription := eventSubscription{
		exchange:   exchange,
		queueName:  queueName,
		routingKey: routingKey,
		channel:    eventChan,
	}
	
	// Store subscription - use exchange as key since we iterate all subscriptions anyway
	key := exchange
	r.subscribers[key] = append(r.subscribers[key], subscription)
	
	return eventChan, nil
}

// Close cleans up event router resources
func (r *MemoryEventRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Close all subscription channels
	for _, subscriptions := range r.subscribers {
		for _, sub := range subscriptions {
			close(sub.channel)
		}
	}
	
	// Clear state
	r.exchanges = make(map[string]bool)
	r.subscribers = make(map[string][]eventSubscription)
	
	return nil
}

// routingKeyMatches checks if a subscription routing key pattern matches a published routing key
func (r *MemoryEventRouter) routingKeyMatches(pattern, routingKey string) bool {
	// Simple pattern matching - supports '*' wildcard
	matched, _ := filepath.Match(pattern, routingKey)
	return matched
}
