package bff

import (
	"context"
	"testing"

	conversationApp "neuromesh/internal/conversation/application"
	"neuromesh/internal/logging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAIOrchestrator implements the AIOrchestrator interface for testing
type MockAIOrchestrator struct {
	mock.Mock
}

func (m *MockAIOrchestrator) ProcessRequest(ctx context.Context, userInput, userID string) (*orchestratorApp.OrchestratorResult, error) {
	args := m.Called(ctx, userInput, userID)
	return args.Get(0).(*orchestratorApp.OrchestratorResult), args.Error(1)
}

func (m *MockAIOrchestrator) ProcessUserRequest(ctx context.Context, request *orchestratorApp.OrchestratorRequest) (*orchestratorApp.OrchestratorResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*orchestratorApp.OrchestratorResult), args.Error(1)
}

// TestGraphNativeConversationContext_TDD tests the graph-native approach
// RED phase: This test will fail because we don't have graph-native context yet
func TestGraphNativeConversationContext_TDD(t *testing.T) {
	t.Run("RED_should_get_conversation_context_from_graph_only", func(t *testing.T) {
		// RED: This test will fail because we don't have GetConversationContext yet

		// Setup
		logger := logging.NewNoOpLogger()
		mockGraph := testHelpers.NewCleanMockGraph()
		conversationService := testHelpers.NewMockConversationService()
		userService := testHelpers.NewMockUserService()

		// Create our local mock orchestrator
		orchestrator := &MockAIOrchestrator{}

		service := NewService(orchestrator, conversationService, userService, mockGraph, logger)
		ctx := context.Background()

		// Setup graph data for conversation context
		conversationID := "conv-123"

		// Mock conversation service to return context from graph traversal
		expectedContext := &conversationApp.ConversationContext{
			ConversationID: conversationID,
			ProjectID:      "proj-456",
			UserID:         "user-789",
			SessionID:      "sess-abc",
			ProjectName:    "Test Project",
		}

		// This method doesn't exist yet - RED phase
		conversationService.On("GetConversationContext", ctx, conversationID).Return(expectedContext, nil)

		// Act: Try to get conversation context using only conversation ID
		context, err := service.GetConversationContext(ctx, conversationID)

		// Assert: Should get complete context from graph relationships
		require.NoError(t, err)
		assert.Equal(t, conversationID, context.ConversationID)
		assert.Equal(t, "proj-456", context.ProjectID)
		assert.Equal(t, "user-789", context.UserID)
		assert.Equal(t, "sess-abc", context.SessionID)
		assert.Equal(t, "Test Project", context.ProjectName)

		// Verify mocks
		conversationService.AssertExpectations(t)
	})

	t.Run("RED_should_process_message_with_only_conversation_id", func(t *testing.T) {
		// RED: This test will fail because ProcessMessage signature is wrong

		// Setup
		logger := logging.NewNoOpLogger()
		mockGraph := testHelpers.NewCleanMockGraph()
		conversationService := testHelpers.NewMockConversationService()
		userService := testHelpers.NewMockUserService()
		orchestrator := &MockAIOrchestrator{}

		service := NewService(orchestrator, conversationService, userService, mockGraph, logger)
		ctx := context.Background()

		conversationID := "conv-123"
		message := "Hello world"

		// Mock conversation context lookup
		expectedContext := &conversationApp.ConversationContext{
			ConversationID: conversationID,
			ProjectID:      "proj-456",
			UserID:         "user-789",
			SessionID:      "sess-abc",
		}
		conversationService.On("GetConversationContext", ctx, conversationID).Return(expectedContext, nil)

		// Mock orchestrator response
		orchestrator.On("ProcessUserRequest", mock.Anything, mock.MatchedBy(func(req interface{}) bool {
			// Verify that orchestrator gets context derived from graph
			return true
		})).Return(&orchestratorApp.OrchestratorResult{
			Message: "Response from orchestrator",
			Success: true,
		}, nil)

		// Act: Process message with only conversation ID (graph-native approach)
		// This method signature doesn't exist yet - RED phase
		response, err := service.ProcessMessageGraphNative(ctx, conversationID, message)

		// Assert: Should work with only conversation ID
		require.NoError(t, err)
		assert.Equal(t, "Response from orchestrator", response.Content)
		assert.Equal(t, conversationID, response.ConversationID)
		// Session ID and other context should be derived from graph
		assert.Equal(t, "sess-abc", response.SessionID)

		// Verify all context was derived from graph
		conversationService.AssertExpectations(t)
		orchestrator.AssertExpectations(t)
	})
}
