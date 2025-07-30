package application

import (
	"context"
	"testing"

	"neuromesh/internal/conversation/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestConversationService_GetConversationContext_Integration tests real repository integration
func TestConversationService_GetConversationContext_Integration(t *testing.T) {
	t.Run("should_call_repository_GetConversationContext_not_return_hardcoded_data", func(t *testing.T) {
		// Setup: Create service with mock repository
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)
		ctx := context.Background()
		conversationID := "test-conv-123"

		// Setup repository to return specific data that proves it was called
		expectedContextData := &domain.ConversationContextData{
			ConversationID: conversationID,
			ProjectID:      "real-project-456",
			UserID:         "real-user-789",
			SessionID:      "real-session-abc",
			ProjectName:    "Real Project Name",
		}
		mockRepo.On("GetConversationContext", ctx, conversationID).Return(expectedContextData, nil)

		// Act: Call service method
		result, err := service.GetConversationContext(ctx, conversationID)

		// Assert: Should get data from repository, not hardcoded mock data
		require.NoError(t, err)
		assert.Equal(t, conversationID, result.ConversationID)
		assert.Equal(t, "real-project-456", result.ProjectID, "Should get real data from repository, not hardcoded mock")
		assert.Equal(t, "real-user-789", result.UserID, "Should get real data from repository, not hardcoded mock")
		assert.Equal(t, "real-session-abc", result.SessionID, "Should get real data from repository, not hardcoded mock")
		assert.Equal(t, "Real Project Name", result.ProjectName, "Should get real data from repository, not hardcoded mock")

		// Verify repository was called
		mockRepo.AssertExpectations(t)
	})

	t.Run("should_link_conversation_to_project_in_graph", func(t *testing.T) {
		// RED: This test should FAIL initially to prove we're not linking to projects
		mockRepo := &MockConversationRepository{}
		service := NewConversationService(mockRepo)
		ctx := context.Background()

		conversationID := "test-conv-123"
		sessionID := "session-456"
		userID := "user-789"
		projectID := "project-abc"

		// Mock the conversation creation
		mockRepo.On("CreateConversation", ctx, mock.AnythingOfType("*domain.Conversation")).Return(nil)
		mockRepo.On("LinkConversationToSession", ctx, conversationID, sessionID).Return(nil)
		mockRepo.On("LinkConversationToUser", ctx, conversationID, userID).Return(nil)
		// This is the missing call that should cause the test to fail
		mockRepo.On("LinkConversationToProject", ctx, conversationID, projectID).Return(nil)

		// Act: Create conversation
		result, err := service.CreateConversation(ctx, conversationID, sessionID, userID, projectID)

		// Assert: Should create conversation successfully (graph-native approach)
		require.NoError(t, err)
		assert.Equal(t, conversationID, result.ID)
		assert.Equal(t, domain.ConversationStatusActive, result.Status)

		// In graph-native architecture, relationships are created separately, not stored as properties
		mockRepo.AssertExpectations(t)
	})
}

// MockConversationRepository provides a mock for the repository interface
type MockConversationRepository struct {
	mock.Mock
}

func (m *MockConversationRepository) GetConversationContext(ctx context.Context, conversationID string) (*domain.ConversationContextData, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).(*domain.ConversationContextData), args.Error(1)
}

// Implement all other required methods as no-ops for this test
func (m *MockConversationRepository) EnsureConversationSchema(ctx context.Context) error { return nil }
func (m *MockConversationRepository) EnsureMessageSchema(ctx context.Context) error      { return nil }
func (m *MockConversationRepository) CreateConversation(ctx context.Context, conversation *domain.Conversation) error {
	args := m.Called(ctx, conversation)
	return args.Error(0)
}
func (m *MockConversationRepository) GetConversation(ctx context.Context, conversationID string) (*domain.Conversation, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).(*domain.Conversation), args.Error(1)
}
func (m *MockConversationRepository) GetConversationWithMessages(ctx context.Context, conversationID string) (*domain.Conversation, error) {
	return nil, nil
}
func (m *MockConversationRepository) UpdateConversation(ctx context.Context, conversation *domain.Conversation) error {
	return nil
}
func (m *MockConversationRepository) DeleteConversation(ctx context.Context, conversationID string) error {
	return nil
}
func (m *MockConversationRepository) AddMessage(ctx context.Context, conversationID string, message *domain.ConversationMessage) error {
	return nil
}
func (m *MockConversationRepository) GetConversationMessages(ctx context.Context, conversationID string) ([]domain.ConversationMessage, error) {
	return nil, nil
}
func (m *MockConversationRepository) GetMessagesByRole(ctx context.Context, conversationID string, role domain.MessageRole) ([]domain.ConversationMessage, error) {
	return nil, nil
}
func (m *MockConversationRepository) LinkConversationToSession(ctx context.Context, conversationID, sessionID string) error {
	args := m.Called(ctx, conversationID, sessionID)
	return args.Error(0)
}
func (m *MockConversationRepository) LinkConversationToUser(ctx context.Context, conversationID, userID string) error {
	args := m.Called(ctx, conversationID, userID)
	return args.Error(0)
}
func (m *MockConversationRepository) LinkConversationToProject(ctx context.Context, conversationID, projectID string) error {
	args := m.Called(ctx, conversationID, projectID)
	return args.Error(0)
}
func (m *MockConversationRepository) LinkExecutionPlan(ctx context.Context, conversationID, planID string) error {
	return nil
}
func (m *MockConversationRepository) FindConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	return nil, nil
}
func (m *MockConversationRepository) FindConversationsBySession(ctx context.Context, sessionID string) ([]*domain.Conversation, error) {
	return nil, nil
}
func (m *MockConversationRepository) FindConversationsByProject(ctx context.Context, projectID string) ([]*domain.Conversation, error) {
	return nil, nil
}
func (m *MockConversationRepository) FindActiveConversations(ctx context.Context) ([]*domain.Conversation, error) {
	return nil, nil
}
func (m *MockConversationRepository) FindConversationsByStatus(ctx context.Context, status domain.ConversationStatus) ([]*domain.Conversation, error) {
	return nil, nil
}
func (m *MockConversationRepository) GetAllConversations(ctx context.Context) ([]*domain.Conversation, error) {
	return nil, nil
}
func (m *MockConversationRepository) GetConversationWithRelationships(ctx context.Context, conversationID string) (*domain.ConversationWithRelationships, error) {
	return nil, nil
}
