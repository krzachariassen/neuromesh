package application

import (
	"context"
	"os"
	"testing"

	aiInfrastructure "neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/logging"
	orchestratorDomain "neuromesh/internal/orchestrator/domain"
	planningApplication "neuromesh/internal/planning/application"
	planningDomain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing (but we'll use real AI provider)
type MockGraphExplorer struct {
	mock.Mock
}

func (m *MockGraphExplorer) GetAgentContext(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

type MockAIExecutionEngine struct {
	mock.Mock
}

func (m *MockAIExecutionEngine) ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext string) (string, error) {
	args := m.Called(ctx, executionPlan, userInput, userID, agentContext)
	return args.String(0), args.Error(1)
}

type MockAIDecisionEngine struct {
	mock.Mock
}

func (m *MockAIDecisionEngine) ExploreAndAnalyze(ctx context.Context, userInput, userID, agentContext, requestID string) (*planningDomain.Analysis, error) {
	args := m.Called(ctx, userInput, userID, agentContext, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*planningDomain.Analysis), args.Error(1)
}

func (m *MockAIDecisionEngine) MakeDecision(ctx context.Context, userInput, userID string, analysis *planningDomain.Analysis, requestID string) (*orchestratorDomain.Decision, error) {
	args := m.Called(ctx, userInput, userID, analysis, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orchestratorDomain.Decision), args.Error(1)
}

// setupRealAIProvider creates a real OpenAI provider for testing
func setupRealAIProviderForOrchestrator(t *testing.T) *aiInfrastructure.OpenAIProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set, skipping AI provider tests")
	}

	config := aiInfrastructure.DefaultOpenAIConfig()
	config.APIKey = apiKey
	config.Model = "gpt-3.5-turbo" // Use faster model for tests
	config.MaxTokens = 1000        // Limit tokens for faster tests

	logger, _ := logging.NewLogger(false) // Production logger for tests
	provider := aiInfrastructure.NewOpenAIProvider(config, logger)

	return provider
}

func TestOrchestratorService_ProcessUserRequest(t *testing.T) {
	t.Run("should process clarification request successfully", func(t *testing.T) {
		// Setup with real AI provider
		aiProvider := setupRealAIProviderForOrchestrator(t)
		aiEngine := planningApplication.NewAIDecisionEngine(aiProvider)

		// Setup mocks for other services
		mockExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		
		// Add missing dependencies for the updated constructor
		mockSynthesizer := testHelpers.NewMockResultSynthesizer()
		mockRepo := testHelpers.NewMockExecutionPlanRepository()

		logger, _ := logging.NewLogger(false)
		service := NewOrchestratorService(aiEngine, mockExplorer, mockExecutionEngine, mockSynthesizer, mockRepo, logger)

		// Test data
		request := &OrchestratorRequest{
			UserInput: "Deploy something unclear",
			UserID:    "user-123",
		}

		agentContext := "Deploy Agent available"

		// Setup expectations
		mockExplorer.On("GetAgentContext", mock.Anything).Return(agentContext, nil)
		mockExecutionEngine.On("ExecuteWithAgents", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("AI handled the request", nil).Maybe()

		// Execute
		result, err := service.ProcessUserRequest(context.Background(), request)

		// Verify
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Analysis)
		assert.NotNil(t, result.Decision)

		// The real AI should handle this request appropriately
		t.Logf("AI Response: %s", result.Message)
		t.Logf("Decision Type: %s", string(result.Decision.Type))

		// Verify mocks
		mockExplorer.AssertExpectations(t)
	})

	t.Run("should process execution request with action successfully", func(t *testing.T) {
		// Setup with real AI provider
		aiProvider := setupRealAIProviderForOrchestrator(t)
		aiEngine := planningApplication.NewAIDecisionEngine(aiProvider)

		// Setup mocks for other services
		mockExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		
		// Add missing dependencies for the updated constructor
		mockSynthesizer := testHelpers.NewMockResultSynthesizer()
		mockRepo := testHelpers.NewMockExecutionPlanRepository()

		logger, _ := logging.NewLogger(false)
		service := NewOrchestratorService(aiEngine, mockExplorer, mockExecutionEngine, mockSynthesizer, mockRepo, logger)

		// Test data
		request := &OrchestratorRequest{
			UserInput: "Deploy my application to production environment",
			UserID:    "user-123",
		}

		agentContext := "Deploy Agent available with deploy capability"

		// Setup expectations
		mockExplorer.On("GetAgentContext", mock.Anything).Return(agentContext, nil)
		mockExecutionEngine.On("ExecuteWithAgents", mock.Anything, mock.Anything, request.UserInput, request.UserID, agentContext).Return("AI orchestrated deployment successfully", nil)

		// Execute
		result, err := service.ProcessUserRequest(context.Background(), request)

		// Verify
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Analysis)
		assert.NotNil(t, result.Decision)

		// Log the AI's decision for inspection
		t.Logf("AI Response: %s", result.Message)
		t.Logf("Decision Type: %s", string(result.Decision.Type))
		t.Logf("Analysis Intent: %s", result.Analysis.Intent)
		t.Logf("Analysis Confidence: %d", result.Analysis.Confidence)

		// Verify mocks
		mockExplorer.AssertExpectations(t)

		// If AI made an execute decision with agents, execution engine should be called
		if result.Decision.Type == orchestratorDomain.DecisionTypeExecute && len(result.Analysis.RequiredAgents) > 0 {
			mockExecutionEngine.AssertExpectations(t)
		}
	})

	t.Run("should handle agent context error", func(t *testing.T) {
		// Setup with real AI provider
		aiProvider := setupRealAIProviderForOrchestrator(t)
		aiEngine := planningApplication.NewAIDecisionEngine(aiProvider)

		// Setup mocks for other services
		mockExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		
		// Add missing dependencies for the updated constructor
		mockSynthesizer := testHelpers.NewMockResultSynthesizer()
		mockRepo := testHelpers.NewMockExecutionPlanRepository()

		logger, _ := logging.NewLogger(false)
		service := NewOrchestratorService(aiEngine, mockExplorer, mockExecutionEngine, mockSynthesizer, mockRepo, logger)

		request := &OrchestratorRequest{
			UserInput: "Deploy app",
			UserID:    "user-123",
		}

		// Setup expectations
		mockExplorer.On("GetAgentContext", mock.Anything).Return("", assert.AnError)

		// Execute
		result, err := service.ProcessUserRequest(context.Background(), request)

		// Verify
		assert.NoError(t, err) // Service should not return Go error
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "Failed to get agent context")

		// Verify mocks
		mockExplorer.AssertExpectations(t)
	})
}
