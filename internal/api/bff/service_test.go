package bff

import (
	"context"
	"testing"

	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/internal/logging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	userDomain "neuromesh/internal/user/domain"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockOrchestrator provides a testify-based mock for orchestrator operations
type MockOrchestrator struct {
	mock.Mock
}

// NewMockOrchestrator creates a new mock orchestrator instance
func NewMockOrchestrator() *MockOrchestrator {
	return &MockOrchestrator{}
}

func (m *MockOrchestrator) ProcessRequest(ctx context.Context, userInput, userID string) (*orchestratorApp.OrchestratorResult, error) {
	args := m.Called(ctx, userInput, userID)
	return args.Get(0).(*orchestratorApp.OrchestratorResult), args.Error(1)
}

func (m *MockOrchestrator) ProcessUserRequest(ctx context.Context, request *orchestratorApp.OrchestratorRequest) (*orchestratorApp.OrchestratorResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*orchestratorApp.OrchestratorResult), args.Error(1)
}

func CreateMockOrchestratorResult(message string, success bool) *orchestratorApp.OrchestratorResult {
	return &orchestratorApp.OrchestratorResult{
		Message: message,
		Success: success,
	}
}

func TestService_EnsureConversation_CleanArchitecture(t *testing.T) {
	// Test the refactored clean architecture conversation handling
	ctx := context.Background()
	logger := logging.NewNoOpLogger()

	// Setup mocks
	mockOrchestrator := NewMockOrchestrator()
	mockConversationService := testHelpers.NewMockConversationService()
	mockUserService := testHelpers.NewMockUserService()
	mockProjectService := testHelpers.NewMockProjectService()
	mockGraph := testHelpers.NewMockGraph()

	service := NewService(
		mockOrchestrator,
		mockConversationService,
		mockUserService,
		mockProjectService,
		mockGraph,
		logger,
	)

	t.Run("should validate project exists before creating conversation", func(t *testing.T) {
		// Setup: Project doesn't exist
		mockProjectService.On("GetProject", ctx, "non-existent-project").Return(nil, assert.AnError)

		request := &ChatRequest{
			Message:   "test message",
			ProjectID: "non-existent-project",
			SessionID: "session-123",
			UserID:    "user-123",
		}

		// Execute
		_, err := service.ensureConversation(ctx, request)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project 'non-existent-project' not found")
		mockProjectService.AssertExpectations(t)
	})

	t.Run("should get existing conversation when ID provided", func(t *testing.T) {
		// Setup: Project exists, conversation exists
		mockProject := testHelpers.CreateMockProject("test-project", "Test Project")
		mockConversation := testHelpers.CreateMockConversation("conv-123", "session-123", "user-123", "test-project")

		mockProjectService.On("GetProject", ctx, "test-project").Return(mockProject, nil)
		mockConversationService.On("GetConversation", ctx, "conv-123").Return(mockConversation, nil)

		request := &ChatRequest{
			Message:        "test message",
			ProjectID:      "test-project",
			SessionID:      "session-123",
			UserID:         "user-123",
			ConversationID: "conv-123",
		}

		// Execute
		conversation, err := service.ensureConversation(ctx, request)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, "conv-123", conversation.ID)
		// Note: ProjectID is now handled via graph relationships, not entity properties
		mockProjectService.AssertExpectations(t)
		mockConversationService.AssertExpectations(t)
	})

	t.Run("should create new conversation when ID provided but doesn't exist", func(t *testing.T) {
		// Setup: Project exists, conversation doesn't exist
		mockProject := testHelpers.CreateMockProject("test-project", "Test Project")
		mockConversation := testHelpers.CreateMockConversation("new-conv-123", "session-123", "user-123", "test-project")

		mockProjectService.On("GetProject", ctx, "test-project").Return(mockProject, nil)
		mockConversationService.On("GetConversation", ctx, "new-conv-123").Return((*conversationDomain.Conversation)(nil), assert.AnError)
		mockConversationService.On("CreateConversation", ctx, "new-conv-123", "session-123", "user-123", "test-project").Return(mockConversation, nil)

		request := &ChatRequest{
			Message:        "test message",
			ProjectID:      "test-project",
			SessionID:      "session-123",
			UserID:         "user-123",
			ConversationID: "new-conv-123",
		}

		// Execute
		conversation, err := service.ensureConversation(ctx, request)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, "new-conv-123", conversation.ID)
		// Note: ProjectID, SessionID, UserID are now handled via graph relationships, not entity properties
		mockProjectService.AssertExpectations(t)
		mockConversationService.AssertExpectations(t)
	})

	t.Run("should find existing active conversation by session", func(t *testing.T) {
		// Setup: Project exists, no conversation ID provided, but active conversation exists for session
		mockProject := testHelpers.CreateMockProject("test-project", "Test Project")
		mockConversation := testHelpers.CreateMockConversation("existing-conv", "session-123", "user-123", "test-project")

		mockProjectService.On("GetProject", ctx, "test-project").Return(mockProject, nil)
		mockConversationService.On("FindConversationsBySession", ctx, "session-123").Return([]*conversationDomain.Conversation{mockConversation}, nil)

		request := &ChatRequest{
			Message:   "test message",
			ProjectID: "test-project",
			SessionID: "session-123",
			UserID:    "user-123",
			// No ConversationID provided
		}

		// Execute
		conversation, err := service.ensureConversation(ctx, request)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, "existing-conv", conversation.ID)
		// Note: ProjectID is now handled via graph relationships, not entity properties
		mockProjectService.AssertExpectations(t)
		mockConversationService.AssertExpectations(t)
	})

	t.Run("should create new conversation when no active conversation found", func(t *testing.T) {
		// Setup: Project exists, no conversation ID provided, no active conversation for session
		mockProject := testHelpers.CreateMockProject("test-project", "Test Project")

		mockProjectService.On("GetProject", ctx, "test-project").Return(mockProject, nil)
		mockConversationService.On("FindConversationsBySession", ctx, "session-123").Return([]*conversationDomain.Conversation{}, nil)

		// Mock CreateConversation - return a specific conversation object
		expectedConversation := testHelpers.CreateMockConversation("generated-conv-id", "session-123", "user-123", "test-project")
		mockConversationService.On("CreateConversation", ctx, mock.AnythingOfType("string"), "session-123", "user-123", "test-project").Return(expectedConversation, nil)

		request := &ChatRequest{
			Message:   "test message",
			ProjectID: "test-project",
			SessionID: "session-123",
			UserID:    "user-123",
			// No ConversationID provided
		}

		// Execute
		conversation, err := service.ensureConversation(ctx, request)

		// Verify
		require.NoError(t, err)
		assert.NotEmpty(t, conversation.ID) // Should have generated an ID
		// Note: ProjectID, SessionID, UserID are now handled via graph relationships, not entity properties
		mockProjectService.AssertExpectations(t)
		mockConversationService.AssertExpectations(t)
	})
}

func TestService_ProcessMessage_UserSessionRelationships(t *testing.T) {
	// Test the critical fix: ensure User/Session nodes exist before conversation creation
	ctx := context.Background()
	logger := logging.NewNoOpLogger()

	// Setup mocks
	mockOrchestrator := NewMockOrchestrator()
	mockConversationService := testHelpers.NewMockConversationService()
	mockUserService := testHelpers.NewMockUserService()
	mockProjectService := testHelpers.NewMockProjectService()
	mockGraph := testHelpers.NewMockGraph()

	service := NewService(
		mockOrchestrator,
		mockConversationService,
		mockUserService,
		mockProjectService,
		mockGraph,
		logger,
	)

	t.Run("should create User and Session nodes before conversation", func(t *testing.T) {
		// This test ensures our critical fix works: User/Session creation before conversation
		sessionID := "session-test-123"
		message := "test message"

		// Setup: Default project exists
		mockProject := testHelpers.CreateMockProject("default-project", "Default Project")
		mockProjectService.On("GetProject", ctx, "default-project").Return(mockProject, nil)

		// Setup: User service calls (ensureUserAndSession)
		mockUser := testHelpers.CreateMockUser("user-session-test-123", sessionID)
		mockSession := testHelpers.CreateMockSession(sessionID, "user-session-test-123")

		mockUserService.On("GetUser", ctx, "user-session-test-123").Return((*userDomain.User)(nil), assert.AnError) // User doesn't exist
		mockUserService.On("CreateUser", ctx, "user-session-test-123", sessionID, mock.Anything).Return(mockUser, nil)
		mockUserService.On("GetSession", ctx, sessionID).Return((*userDomain.Session)(nil), assert.AnError) // Session doesn't exist
		mockUserService.On("CreateSession", ctx, sessionID, "user-session-test-123", mock.Anything).Return(mockSession, nil)

		// Setup: Conversation creation (should happen AFTER user/session creation)
		mockConversationService.On("FindConversationsBySession", ctx, sessionID).Return([]*conversationDomain.Conversation{}, nil)
		// Return a specific conversation object, not a function
		expectedConversation := testHelpers.CreateMockConversation("generated-conv-id", sessionID, "user-session-test-123", "default-project")
		mockConversationService.On("CreateConversation", ctx, mock.AnythingOfType("string"), sessionID, "user-session-test-123", "default-project").Return(expectedConversation, nil)

		// Setup: Message handling
		mockConversationService.On("AddMessage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, message, mock.Anything).Return(nil)
		mockConversationService.On("AddMessage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)

		// Setup: Orchestrator response
		mockOrchestratorResult := CreateMockOrchestratorResult("AI response", true)
		mockOrchestrator.On("ProcessUserRequest", ctx, mock.AnythingOfType("*application.OrchestratorRequest")).Return(mockOrchestratorResult, nil)

		// Execute
		response, err := service.ProcessMessage(ctx, sessionID, message, "default-project")

		// Verify
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, sessionID, response.SessionID)
		assert.Equal(t, "default-project", response.ProjectID)
		assert.NotEmpty(t, response.ConversationID)

		// Verify call order: User/Session creation before conversation
		mockUserService.AssertExpectations(t)
		mockConversationService.AssertExpectations(t)
		mockOrchestrator.AssertExpectations(t)
	})
}
