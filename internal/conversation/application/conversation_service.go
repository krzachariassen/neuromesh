package application

import (
	"context"
	"fmt"

	aiDomain "neuromesh/internal/ai/domain"
	"neuromesh/internal/conversation/domain"
)

// ConversationContext represents complete conversation context derived from graph relationships
type ConversationContext struct {
	ConversationID string
	ProjectID      string
	UserID         string
	SessionID      string
	ProjectName    string
}

// ConversationService defines the application service interface for conversation management
type ConversationService interface {
	// Conversation management
	CreateConversation(ctx context.Context, id, sessionID, userID, projectID string) (*domain.Conversation, error)
	GetConversation(ctx context.Context, conversationID string) (*domain.Conversation, error)
	GetConversationWithMessages(ctx context.Context, conversationID string) (*domain.Conversation, error)
	UpdateConversationStatus(ctx context.Context, conversationID string, status domain.ConversationStatus) error
	DeleteConversation(ctx context.Context, conversationID string) error

	// Graph-native context operations
	GetConversationContext(ctx context.Context, conversationID string) (*ConversationContext, error)

	// Message management
	AddMessage(ctx context.Context, conversationID, messageID string, role domain.MessageRole, content string, metadata map[string]interface{}) error
	GetConversationMessages(ctx context.Context, conversationID string) ([]domain.ConversationMessage, error)
	GetMessagesByRole(ctx context.Context, conversationID string, role domain.MessageRole) ([]domain.ConversationMessage, error)
	GetConversationHistory(ctx context.Context, conversationID string) ([]*aiDomain.AIConversationMessage, error)

	// Execution plan linking
	LinkExecutionPlan(ctx context.Context, conversationID, planID string) error

	// Relationship management
	LinkConversationToSession(ctx context.Context, conversationID, sessionID string) error
	LinkConversationToUser(ctx context.Context, conversationID, userID string) error
	LinkConversationToProject(ctx context.Context, conversationID, projectID string) error

	// Query operations
	FindConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error)
	FindConversationsBySession(ctx context.Context, sessionID string) ([]*domain.Conversation, error)
	FindActiveConversations(ctx context.Context) ([]*domain.Conversation, error)
	GetAllConversations(ctx context.Context) ([]*domain.Conversation, error)

	// Schema management
	EnsureSchema(ctx context.Context) error
}

// ConversationServiceImpl implements the ConversationService interface
type ConversationServiceImpl struct {
	repo domain.ConversationRepository
}

// NewConversationService creates a new conversation service implementation
func NewConversationService(repo domain.ConversationRepository) ConversationService {
	return &ConversationServiceImpl{
		repo: repo,
	}
}

// CreateConversation creates a new conversation with graph-native relationships
func (s *ConversationServiceImpl) CreateConversation(ctx context.Context, id, sessionID, userID, projectID string) (*domain.Conversation, error) {
	// Create conversation entity without embedded foreign keys
	conversation, err := domain.NewConversation(id)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation domain object: %w", err)
	}

	if err := s.repo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to store conversation: %w", err)
	}

	// Create graph relationships (this is the graph-native approach)
	if err := s.repo.LinkConversationToSession(ctx, id, sessionID); err != nil {
		return nil, fmt.Errorf("failed to link conversation to session: %w", err)
	}

	if err := s.repo.LinkConversationToUser(ctx, id, userID); err != nil {
		return nil, fmt.Errorf("failed to link conversation to user: %w", err)
	}

	if err := s.repo.LinkConversationToProject(ctx, id, projectID); err != nil {
		return nil, fmt.Errorf("failed to link conversation to project: %w", err)
	}

	return conversation, nil
}

// GetConversation retrieves a conversation by ID
func (s *ConversationServiceImpl) GetConversation(ctx context.Context, conversationID string) (*domain.Conversation, error) {
	conversation, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	return conversation, nil
}

// GetConversationWithMessages retrieves a conversation with all its messages
func (s *ConversationServiceImpl) GetConversationWithMessages(ctx context.Context, conversationID string) (*domain.Conversation, error) {
	conversation, err := s.repo.GetConversationWithMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation with messages: %w", err)
	}
	return conversation, nil
}

// UpdateConversationStatus updates a conversation's status
func (s *ConversationServiceImpl) UpdateConversationStatus(ctx context.Context, conversationID string, status domain.ConversationStatus) error {
	conversation, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	conversation.SetStatus(status)

	if err := s.repo.UpdateConversation(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return nil
}

// DeleteConversation deletes a conversation
func (s *ConversationServiceImpl) DeleteConversation(ctx context.Context, conversationID string) error {
	if err := s.repo.DeleteConversation(ctx, conversationID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	return nil
}

// AddMessage adds a message to a conversation
func (s *ConversationServiceImpl) AddMessage(ctx context.Context, conversationID, messageID string, role domain.MessageRole, content string, metadata map[string]interface{}) error {
	// Get the conversation to ensure it exists and update it
	conversation, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Add message to conversation domain object
	if err := conversation.AddMessage(messageID, role, content, metadata); err != nil {
		return fmt.Errorf("failed to add message to conversation: %w", err)
	}

	// Get the newly added message
	messages := conversation.GetMessagesByRole(role)
	var newMessage *domain.ConversationMessage
	for _, msg := range messages {
		if msg.ID == messageID {
			newMessage = &msg
			break
		}
	}

	if newMessage == nil {
		return fmt.Errorf("failed to find newly added message")
	}

	// Store message in graph
	if err := s.repo.AddMessage(ctx, conversationID, newMessage); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Update conversation
	if err := s.repo.UpdateConversation(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return nil
}

// GetConversationMessages retrieves all messages for a conversation
func (s *ConversationServiceImpl) GetConversationMessages(ctx context.Context, conversationID string) ([]domain.ConversationMessage, error) {
	messages, err := s.repo.GetConversationMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}
	return messages, nil
}

// GetMessagesByRole retrieves messages by role for a conversation
func (s *ConversationServiceImpl) GetMessagesByRole(ctx context.Context, conversationID string, role domain.MessageRole) ([]domain.ConversationMessage, error) {
	messages, err := s.repo.GetMessagesByRole(ctx, conversationID, role)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by role: %w", err)
	}
	return messages, nil
}

// GetConversationHistory retrieves conversation messages formatted for AI conversation context
func (s *ConversationServiceImpl) GetConversationHistory(ctx context.Context, conversationID string) ([]*aiDomain.AIConversationMessage, error) {
	// Get conversation messages
	messages, err := s.repo.GetConversationMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	// Convert domain messages to AI conversation format
	var aiMessages []*aiDomain.AIConversationMessage
	for _, msg := range messages {
		role := string(msg.Role)
		if role == "user" || role == "assistant" || role == "system" {
			aiMessages = append(aiMessages, aiDomain.NewAIConversationMessage(role, msg.Content))
		}
	}

	return aiMessages, nil
}

// LinkExecutionPlan links an execution plan to a conversation
func (s *ConversationServiceImpl) LinkExecutionPlan(ctx context.Context, conversationID, planID string) error {
	// Get the conversation and update it
	conversation, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Add plan to conversation domain object
	if err := conversation.LinkExecutionPlan(planID); err != nil {
		return fmt.Errorf("failed to link execution plan to conversation: %w", err)
	}

	// Update conversation in graph
	if err := s.repo.UpdateConversation(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	// Create graph relationship
	if err := s.repo.LinkExecutionPlan(ctx, conversationID, planID); err != nil {
		return fmt.Errorf("failed to link execution plan in graph: %w", err)
	}

	return nil
}

// LinkConversationToSession links a conversation to a session
func (s *ConversationServiceImpl) LinkConversationToSession(ctx context.Context, conversationID, sessionID string) error {
	if err := s.repo.LinkConversationToSession(ctx, conversationID, sessionID); err != nil {
		return fmt.Errorf("failed to link conversation to session: %w", err)
	}
	return nil
}

// LinkConversationToUser links a conversation to a user
func (s *ConversationServiceImpl) LinkConversationToUser(ctx context.Context, conversationID, userID string) error {
	if err := s.repo.LinkConversationToUser(ctx, conversationID, userID); err != nil {
		return fmt.Errorf("failed to link conversation to user: %w", err)
	}
	return nil
}

// LinkConversationToProject links a conversation to a project
func (s *ConversationServiceImpl) LinkConversationToProject(ctx context.Context, conversationID, projectID string) error {
	if err := s.repo.LinkConversationToProject(ctx, conversationID, projectID); err != nil {
		return fmt.Errorf("failed to link conversation to project: %w", err)
	}
	return nil
}

// FindConversationsByUser finds conversations by user ID
func (s *ConversationServiceImpl) FindConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	conversations, err := s.repo.FindConversationsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversations by user: %w", err)
	}
	return conversations, nil
}

// FindConversationsBySession finds conversations by session ID
func (s *ConversationServiceImpl) FindConversationsBySession(ctx context.Context, sessionID string) ([]*domain.Conversation, error) {
	conversations, err := s.repo.FindConversationsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversations by session: %w", err)
	}
	return conversations, nil
}

// FindActiveConversations finds all active conversations
func (s *ConversationServiceImpl) FindActiveConversations(ctx context.Context) ([]*domain.Conversation, error) {
	conversations, err := s.repo.FindActiveConversations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find active conversations: %w", err)
	}
	return conversations, nil
}

// GetAllConversations retrieves all conversations in the system
func (s *ConversationServiceImpl) GetAllConversations(ctx context.Context) ([]*domain.Conversation, error) {
	conversations, err := s.repo.GetAllConversations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all conversations: %w", err)
	}
	return conversations, nil
}

// EnsureSchema ensures the conversation and message schemas are in place
func (s *ConversationServiceImpl) EnsureSchema(ctx context.Context) error {
	if err := s.repo.EnsureConversationSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure conversation schema: %w", err)
	}

	if err := s.repo.EnsureMessageSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure message schema: %w", err)
	}

	return nil
}

// GetConversationContext retrieves complete conversation context using graph relationships
// REFACTOR: Real implementation using repository graph traversal
func (s *ConversationServiceImpl) GetConversationContext(ctx context.Context, conversationID string) (*ConversationContext, error) {
	// Use repository to get context from graph relationships
	contextData, err := s.repo.GetConversationContext(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation context: %w", err)
	}

	// Convert domain context data to application context
	return &ConversationContext{
		ConversationID: contextData.ConversationID,
		ProjectID:      contextData.ProjectID,
		UserID:         contextData.UserID,
		SessionID:      contextData.SessionID,
		ProjectName:    contextData.ProjectName,
	}, nil
}
