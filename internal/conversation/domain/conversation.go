package domain

import (
	"fmt"
	"time"
)

// ConversationValidationError represents validation errors for conversations
type ConversationValidationError struct {
	Field   string
	Message string
}

func (e ConversationValidationError) Error() string {
	return fmt.Sprintf("conversation validation error - %s: %s", e.Field, e.Message)
}

// ConversationStatus represents the status of a conversation
type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusPaused   ConversationStatus = "paused"
	ConversationStatusClosed   ConversationStatus = "closed"
	ConversationStatusArchived ConversationStatus = "archived"
)

// MessageRole represents the role of a message sender
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleAgent     MessageRole = "agent"
)

// ConversationMessage represents a message within a conversation
type ConversationMessage struct {
	ID        string                 `json:"id"`
	Role      MessageRole            `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Conversation represents a multi-turn conversation between users and AI
// Foreign key relationships (user, session, project, execution plans) are handled via graph edges
type Conversation struct {
	ID        string                `json:"id"`
	Status    ConversationStatus    `json:"status"`
	Messages  []ConversationMessage `json:"messages"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// NewConversation creates a new conversation with only essential properties
// All relationships (user, session, project) are established via graph edges
func NewConversation(id string) (*Conversation, error) {
	if id == "" {
		return nil, ConversationValidationError{Field: "id", Message: "conversation ID cannot be empty"}
	}

	now := time.Now().UTC()

	conversation := &Conversation{
		ID:        id,
		Status:    ConversationStatusActive,
		Messages:  make([]ConversationMessage, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}

	return conversation, nil
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(messageID string, role MessageRole, content string, metadata map[string]interface{}) error {
	if messageID == "" {
		return ConversationValidationError{Field: "message_id", Message: "message ID cannot be empty"}
	}

	message := ConversationMessage{
		ID:        messageID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
		Metadata:  metadata,
	}

	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}

	c.Messages = append(c.Messages, message)
	c.UpdatedAt = time.Now().UTC()

	return nil
}

// LinkExecutionPlan is deprecated in graph-native architecture
// Execution plan relationships are now handled via graph edges
// This method is kept for backward compatibility but should not be used
func (c *Conversation) LinkExecutionPlan(planID string) error {
	if planID == "" {
		return ConversationValidationError{Field: "execution_plan_id", Message: "execution plan ID cannot be empty"}
	}

	// In graph-native architecture, this relationship is handled by the repository layer
	// This method now only updates the timestamp to maintain contract
	c.UpdatedAt = time.Now().UTC()

	return nil
}

// GetMessagesByRole returns all messages with the specified role
func (c *Conversation) GetMessagesByRole(role MessageRole) []ConversationMessage {
	var messages []ConversationMessage

	for _, message := range c.Messages {
		if message.Role == role {
			messages = append(messages, message)
		}
	}

	return messages
}

// Validate validates the conversation entity
// In graph-native architecture, relationship validation is handled by the repository layer
func (c *Conversation) Validate() error {
	if c.ID == "" {
		return ConversationValidationError{Field: "id", Message: "ID cannot be empty"}
	}

	// Foreign key validations removed - relationships are validated via graph edges
	return nil
}

// SetStatus updates the conversation status
func (c *Conversation) SetStatus(status ConversationStatus) {
	c.Status = status
	c.UpdatedAt = time.Now().UTC()
}

// GetLatestMessage returns the most recent message in the conversation
func (c *Conversation) GetLatestMessage() *ConversationMessage {
	if len(c.Messages) == 0 {
		return nil
	}

	latest := &c.Messages[0]
	for i := 1; i < len(c.Messages); i++ {
		if c.Messages[i].Timestamp.After(latest.Timestamp) {
			latest = &c.Messages[i]
		}
	}

	return latest
}

// GetMessageCount returns the total number of messages in the conversation
func (c *Conversation) GetMessageCount() int {
	return len(c.Messages)
}
