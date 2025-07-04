package domain

import "context"

// ConversationRepository defines the interface for conversation persistence operations
type ConversationRepository interface {
	// Schema management
	EnsureConversationSchema(ctx context.Context) error
	EnsureMessageSchema(ctx context.Context) error

	// Conversation operations
	CreateConversation(ctx context.Context, conversation *Conversation) error
	GetConversation(ctx context.Context, conversationID string) (*Conversation, error)
	GetConversationWithMessages(ctx context.Context, conversationID string) (*Conversation, error)
	UpdateConversation(ctx context.Context, conversation *Conversation) error
	DeleteConversation(ctx context.Context, conversationID string) error

	// Message operations
	AddMessage(ctx context.Context, conversationID string, message *ConversationMessage) error
	GetConversationMessages(ctx context.Context, conversationID string) ([]ConversationMessage, error)
	GetMessagesByRole(ctx context.Context, conversationID string, role MessageRole) ([]ConversationMessage, error)

	// Relationship operations
	LinkConversationToSession(ctx context.Context, conversationID, sessionID string) error
	LinkConversationToUser(ctx context.Context, conversationID, userID string) error
	LinkExecutionPlan(ctx context.Context, conversationID, planID string) error

	// Query operations
	FindConversationsByUser(ctx context.Context, userID string) ([]*Conversation, error)
	FindConversationsBySession(ctx context.Context, sessionID string) ([]*Conversation, error)
	FindActiveConversations(ctx context.Context) ([]*Conversation, error)
	FindConversationsByStatus(ctx context.Context, status ConversationStatus) ([]*Conversation, error)
}
