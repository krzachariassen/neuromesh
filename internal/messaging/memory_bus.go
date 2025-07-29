package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"neuromesh/internal/logging"

	"github.com/google/uuid"
)

// MemoryMessageBus implements MessageBus using in-memory channels
// This is perfect for development and testing
type MemoryMessageBus struct {
	subscribers   map[string]chan *Message
	conversations map[string]*ConversationContext
	history       map[string][]*Message
	
	// Domain event support
	eventSubscribers map[string]chan *DomainEvent
	
	mutex         sync.RWMutex
	logger        logging.Logger
}

// NewMemoryMessageBus creates a new in-memory message bus
func NewMemoryMessageBus(logger logging.Logger) *MemoryMessageBus {
	return &MemoryMessageBus{
		subscribers:      make(map[string]chan *Message),
		conversations:    make(map[string]*ConversationContext),
		history:          make(map[string][]*Message),
		eventSubscribers: make(map[string]chan *DomainEvent),
		logger:           logger,
	}
}

// SendMessage sends a message to a specific recipient
func (mb *MemoryMessageBus) SendMessage(ctx context.Context, message *Message) error {
	// Validate CorrelationID is present
	if message.CorrelationID == "" {
		return fmt.Errorf("correlation ID is required for all messages")
	}

	mb.mutex.RLock()
	subscriber, exists := mb.subscribers[message.ToID]
	mb.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no subscriber found for participant %s", message.ToID)
	}

	// Store message in history
	mb.mutex.Lock()
	if message.CorrelationID != "" {
		mb.history[message.CorrelationID] = append(mb.history[message.CorrelationID], message)
	}
	mb.mutex.Unlock()

	// Send message (non-blocking)
	select {
	case subscriber <- message:
		// Log successful message delivery to subscriber
		if mb.logger != nil {
			mb.logger.Debug("📨 Message delivered to subscriber",
				"message_id", message.ID,
				"correlation_id", message.CorrelationID,
				"from", message.FromID,
				"to", message.ToID,
				"message_type", message.MessageType,
				"content_length", len(message.Content))
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("subscriber channel full for participant %s", message.ToID)
	}
}

// PrepareAgentQueue ensures queue and routing are set up for an agent without starting consumption
// For MemoryMessageBus, this is a no-op since queues are created on-demand
func (mb *MemoryMessageBus) PrepareAgentQueue(ctx context.Context, agentID string) error {
	// In-memory bus doesn't need explicit queue preparation
	mb.logger.Debug("Queue preparation not needed for memory bus", "agent_id", agentID)
	return nil
}

// Subscribe subscribes to messages for a specific participant
func (mb *MemoryMessageBus) Subscribe(ctx context.Context, participantID string) (<-chan *Message, error) {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	if _, exists := mb.subscribers[participantID]; exists {
		return nil, fmt.Errorf("participant %s already subscribed", participantID)
	}

	ch := make(chan *Message, 100) // Buffered channel
	mb.subscribers[participantID] = ch
	return ch, nil
}

// Unsubscribe unsubscribes from messages
func (mb *MemoryMessageBus) Unsubscribe(ctx context.Context, participantID string) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	if ch, exists := mb.subscribers[participantID]; exists {
		close(ch)
		delete(mb.subscribers, participantID)
		return nil
	}

	return fmt.Errorf("participant %s not subscribed", participantID)
}

// PublishMessage publishes a message to multiple recipients
func (mb *MemoryMessageBus) PublishMessage(ctx context.Context, message *Message, recipients []string) error {
	for _, recipient := range recipients {
		msg := *message // Copy message
		msg.ToID = recipient
		if err := mb.SendMessage(ctx, &msg); err != nil {
			return fmt.Errorf("failed to send to %s: %w", recipient, err)
		}
	}
	return nil
}

// GetConversationHistory returns the conversation history
func (mb *MemoryMessageBus) GetConversationHistory(ctx context.Context, conversationID string) ([]*Message, error) {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()

	history, exists := mb.history[conversationID]
	if !exists {
		return []*Message{}, nil
	}

	// Return copy of history
	result := make([]*Message, len(history))
	copy(result, history)
	return result, nil
}

// CreateConversation creates a new conversation context
func (mb *MemoryMessageBus) CreateConversation(ctx context.Context, participants []string, context map[string]interface{}) (*ConversationContext, error) {
	conversationID := uuid.New().String()

	conversation := &ConversationContext{
		ConversationID: conversationID,
		Participants:   participants,
		Context:        context,
		StartTime:      time.Now(),
		LastActivity:   time.Now(),
	}

	mb.mutex.Lock()
	mb.conversations[conversationID] = conversation
	mb.mutex.Unlock()

	return conversation, nil
}

// PublishDomainEvent publishes a domain event with clean interface
func (mb *MemoryMessageBus) PublishDomainEvent(ctx context.Context, eventType string, event interface{}) error {
	// Marshal the event data
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	domainEvent := &DomainEvent{
		EventType: eventType,
		EventData: json.RawMessage(eventData),
		Metadata: map[string]interface{}{
			"id": uuid.New().String(),
		},
		Timestamp: time.Now(),
	}

	mb.mutex.RLock()
	defer mb.mutex.RUnlock()

	// Send to all matching subscribers
	for subscriberKey, eventChan := range mb.eventSubscribers {
		if mb.eventPatternMatches(subscriberKey, eventType) {
			select {
			case eventChan <- domainEvent:
				mb.logger.Info("📨 Domain event delivered", "event_type", eventType, "subscriber", subscriberKey)
			default:
				mb.logger.Warn("Domain event channel full, dropping event", "event_type", eventType, "subscriber", subscriberKey)
			}
		}
	}

	return nil
}

// SubscribeToDomainEvents subscribes to domain events matching pattern
func (mb *MemoryMessageBus) SubscribeToDomainEvents(ctx context.Context, subscriberID, eventPattern string) (<-chan *DomainEvent, error) {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	
	// Create buffered channel for events
	eventChan := make(chan *DomainEvent, 100)
	
	// Store subscription with pattern key
	key := fmt.Sprintf("%s:%s", subscriberID, eventPattern)
	mb.eventSubscribers[key] = eventChan
	
	return eventChan, nil
}

// eventPatternMatches checks if an event type matches a subscription pattern
func (mb *MemoryMessageBus) eventPatternMatches(subscriberKey, eventType string) bool {
	// Extract pattern from subscriber key (format: "subscriberID:pattern")
	parts := strings.SplitN(subscriberKey, ":", 2)
	if len(parts) != 2 {
		return false
	}
	
	pattern := parts[1]
	
	// Simple pattern matching - supports exact match and wildcard patterns
	if pattern == eventType {
		return true
	}
	
	// Support simple wildcard patterns like "execution.*"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(eventType, prefix)
	}
	
	// Support filepath-style pattern matching
	matched, _ := filepath.Match(pattern, eventType)
	return matched
}

// Close cleans up resources
func (mb *MemoryMessageBus) Close() error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	
	// Close all subscription channels
	for _, ch := range mb.subscribers {
		close(ch)
	}
	
	// Close all event subscription channels
	for _, ch := range mb.eventSubscribers {
		close(ch)
	}
	
	// Clear maps
	mb.subscribers = make(map[string]chan *Message)
	mb.eventSubscribers = make(map[string]chan *DomainEvent)
	
	return nil
}
