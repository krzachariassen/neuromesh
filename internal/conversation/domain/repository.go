package domain

import "context"

// ConversationContextData represents conversation context data retrieved from graph relationships
type ConversationContextData struct {
	ConversationID string
	ProjectID      string
	UserID         string
	SessionID      string
	ProjectName    string
}

// ConversationWithRelationships represents a conversation with all its related entities loaded
type ConversationWithRelationships struct {
	Conversation   *Conversation
	User           *UserInfo
	Session        *SessionInfo
	Project        *ProjectInfo
	ExecutionPlans []*ExecutionPlanInfo
}

// UserInfo represents user information for conversation context
type UserInfo struct {
	ID    string
	Email string
}

// SessionInfo represents session information for conversation context
type SessionInfo struct {
	ID     string
	Status string
}

// ProjectInfo represents project information for conversation context
type ProjectInfo struct {
	ID   string
	Name string
}

// ExecutionPlanInfo represents execution plan information for conversation context
type ExecutionPlanInfo struct {
	ID     string
	Status string
}

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
	LinkConversationToProject(ctx context.Context, conversationID, projectID string) error
	LinkExecutionPlan(ctx context.Context, conversationID, planID string) error

	// Query operations
	FindConversationsByUser(ctx context.Context, userID string) ([]*Conversation, error)
	FindConversationsBySession(ctx context.Context, sessionID string) ([]*Conversation, error)
	FindConversationsByProject(ctx context.Context, projectID string) ([]*Conversation, error)
	FindActiveConversations(ctx context.Context) ([]*Conversation, error)
	FindConversationsByStatus(ctx context.Context, status ConversationStatus) ([]*Conversation, error)
	GetAllConversations(ctx context.Context) ([]*Conversation, error)

	// Graph traversal operations for context
	GetConversationContext(ctx context.Context, conversationID string) (*ConversationContextData, error)
	GetConversationWithRelationships(ctx context.Context, conversationID string) (*ConversationWithRelationships, error)
}
