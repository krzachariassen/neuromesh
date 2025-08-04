package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	aiDomain "neuromesh/internal/ai/domain"
	"neuromesh/internal/logging"
	domain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

// MockAIPlanningEngine for testing the new planning interface
type MockAIPlanningEngine struct {
	mock.Mock
}

func (m *MockAIPlanningEngine) CreateExecutionPlan(ctx context.Context, userInput, userID, agentContext, requestID string, conversationHistory ...[]*aiDomain.AIConversationMessage) (*domain.ExecutionPlan, error) {
	args := m.Called(ctx, userInput, userID, agentContext, requestID, conversationHistory)
	return args.Get(0).(*domain.ExecutionPlan), args.Error(1)
}

func (m *MockAIPlanningEngine) LinkPlanningResultToConversation(ctx context.Context, planningResultID, conversationID string) error {
	args := m.Called(ctx, planningResultID, conversationID)
	return args.Error(0)
}

// MockGraphExplorer for testing graph exploration
type MockGraphExplorer struct {
	mock.Mock
}

func (m *MockGraphExplorer) GetAgentContext(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

// MockAIExecutionEngine for testing execution engine
type MockAIExecutionEngine struct {
	mock.Mock
}

func (m *MockAIExecutionEngine) ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext, planID string) (string, error) {
	args := m.Called(ctx, executionPlan, userInput, userID, agentContext, planID)
	return args.String(0), args.Error(1)
}

func TestOrchestratorService_PureOrchestrationPhase3(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewNoOpLogger()

	t.Run("should return execution plan ID immediately - pure orchestration", func(t *testing.T) {
		// Arrange: Mock components for Phase 3 pure orchestration
		mockPlanningEngine := &MockAIPlanningEngine{}
		mockGraphExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		mockConversationService := testHelpers.NewMockConversationService()
		mockPlanRepository := testHelpers.NewMockExecutionPlanRepository()

		// Set up mock expectations
		mockGraphExplorer.On("GetAgentContext", ctx).Return("Available agents: generic-agent", nil)

		executionPlan := &domain.ExecutionPlan{
			ID:             "planning-001",
			Type:           domain.PlanningTypeExecute,
			RequiredAgents: []string{"generic-agent"},
			Intent:         "weather_inquiry",
			Category:       "general_information",
			Confidence:     95,
			Reasoning:      "Simple weather question for generic agent",
		}

		mockPlanningEngine.On("CreateExecutionPlan", ctx, "What is the weather like today?", "user-123", "Available agents: generic-agent", "msg-001", mock.Anything).Return(executionPlan, nil)
		mockPlanningEngine.On("LinkPlanningResultToConversation", ctx, "planning-001", "conv-001").Return(nil)
		mockConversationService.On("GetConversationHistory", ctx, "conv-001").Return([]*aiDomain.AIConversationMessage{}, nil)
		mockConversationService.On("LinkExecutionPlan", ctx, "conv-001", "planning-001").Return(nil)

		// Mock the background execution that happens after returning the response
		mockExecutionEngine.On("ExecuteWithAgents", mock.Anything, mock.AnythingOfType("string"), "What is the weather like today?", "user-123", "Available agents: generic-agent", "planning-001").Return("Execution completed", nil)

		// Create orchestrator with Phase 3 pure orchestration
		orchestrator := NewOrchestratorService(
			mockPlanningEngine,
			mockGraphExplorer,
			mockExecutionEngine,
			mockConversationService,
			mockPlanRepository,
			logger,
		)

		request := &OrchestratorRequest{
			UserInput:      "What is the weather like today?",
			UserID:         "user-123",
			MessageID:      "msg-001",
			ConversationID: "conv-001",
		}

		// Act: Process request using pure orchestration
		result, err := orchestrator.ProcessUserRequest(ctx, request)

		// Assert: Should return execution plan ID immediately (pure orchestration)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "planning-001", result.ExecutionPlanID)
		assert.Equal(t, "executing", result.Status)
		assert.Empty(t, result.Message, "Pure orchestration should not return immediate message")

		// Verify all mocks were called
		mockGraphExplorer.AssertExpectations(t)
		mockPlanningEngine.AssertExpectations(t)
		mockConversationService.AssertExpectations(t)
	})

	t.Run("should handle clarification requests", func(t *testing.T) {
		// Arrange: Mock components for clarification scenario
		mockPlanningEngine := &MockAIPlanningEngine{}
		mockGraphExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		mockConversationService := testHelpers.NewMockConversationService()
		mockPlanRepository := testHelpers.NewMockExecutionPlanRepository()

		// Set up mock expectations for clarification
		mockGraphExplorer.On("GetAgentContext", ctx).Return("Available agents: generic-agent", nil)

		executionPlan := &domain.ExecutionPlan{
			ID:          "planning-002",
			Type:        domain.PlanningTypeClarify,
			Description: "What type of analysis would you like me to perform? Request is too vague to determine specific action",
			Reasoning:   "What type of analysis would you like me to perform? Request is too vague to determine specific action",
			Intent:      "unclear_request",
			Category:    "clarification",
			Confidence:  40,
		}

		mockPlanningEngine.On("CreateExecutionPlan", ctx, "Do something", "user-123", "Available agents: generic-agent", "msg-002", mock.Anything).Return(executionPlan, nil)
		mockPlanningEngine.On("LinkPlanningResultToConversation", ctx, "planning-002", "conv-001").Return(nil)
		mockConversationService.On("GetConversationHistory", ctx, "conv-001").Return([]*aiDomain.AIConversationMessage{}, nil)

		// Create orchestrator
		orchestrator := NewOrchestratorService(
			mockPlanningEngine,
			mockGraphExplorer,
			mockExecutionEngine,
			mockConversationService,
			mockPlanRepository,
			logger,
		)

		request := &OrchestratorRequest{
			UserInput:      "Do something",
			UserID:         "user-123",
			MessageID:      "msg-002",
			ConversationID: "conv-001",
		}

		// Act
		result, err := orchestrator.ProcessUserRequest(ctx, request)

		// Assert: Should return clarification
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, domain.PlanningTypeClarify, result.ExecutionPlan.Type)
		assert.Equal(t, "What type of analysis would you like me to perform? Request is too vague to determine specific action", result.Message)
		assert.Equal(t, "planning-002", result.ExecutionPlanID, "Clarification should also create execution plan in unified approach")

		// Verify mocks
		mockGraphExplorer.AssertExpectations(t)
		mockPlanningEngine.AssertExpectations(t)
	})
}
