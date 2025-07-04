package application

import (
	"context"
	"strings"
	"testing"

	"neuromesh/internal/logging"
	orchestratorDomain "neuromesh/internal/orchestrator/domain"

	"github.com/stretchr/testify/mock"
)

// TestOrchestratorAgentRoutingEnforcement tests the CRITICAL principle that
// the orchestrator should NEVER answer user tasks directly - all user tasks
// must be routed through appropriate agents
func TestOrchestratorAgentRoutingEnforcement(t *testing.T) {
	// This test enforces the core AI-native orchestration principle:
	// ORCHESTRATOR SHOULD NEVER BYPASS AGENTS FOR USER TASKS

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	// Test cases that should ALWAYS route to agents
	userTasksRequiringAgents := []struct {
		name        string
		userInput   string
		description string
	}{
		{
			name:        "Word Count Task",
			userInput:   "Count the words in hello world",
			description: "Should route to text-processor agent, not answered directly by AI",
		},
		{
			name:        "Text Analysis Task",
			userInput:   "Analyze the sentiment of this text: I love programming",
			description: "Should route to text-processor agent for analysis",
		},
		{
			name:        "Mathematical Calculation",
			userInput:   "Calculate 15 * 24",
			description: "Should route to calculation agent, not computed by orchestrator",
		},
		{
			name:        "File Processing",
			userInput:   "Process this CSV file: data.csv",
			description: "Should route to file-processor agent",
		},
		{
			name:        "Data Transformation",
			userInput:   "Convert JSON to XML format",
			description: "Should route to data-processor agent",
		},
		{
			name:        "Code Generation",
			userInput:   "Generate a Python function to sort a list",
			description: "Should route to code-generator agent",
		},
	}

	// Test cases that ARE allowed to be answered directly (orchestrator meta-queries)
	orchestratorMetaQueries := []struct {
		name        string
		userInput   string
		description string
	}{
		{
			name:        "Agent Listing",
			userInput:   "What agents do you have available?",
			description: "Orchestrator meta-query - allowed direct response",
		},
		{
			name:        "System Status",
			userInput:   "What is the system status?",
			description: "Orchestrator meta-query - allowed direct response",
		},
		{
			name:        "Agent Capabilities",
			userInput:   "Show me agent capabilities",
			description: "Orchestrator meta-query - allowed direct response",
		},
		{
			name:        "Health Check",
			userInput:   "Are you healthy?",
			description: "Orchestrator meta-query - allowed direct response",
		},
	}

	t.Run("CRITICAL: User tasks must route to agents", func(t *testing.T) {
		for _, testCase := range userTasksRequiringAgents {
			t.Run(testCase.name, func(t *testing.T) {
				// Create mocks - use setup for user tasks (requires agents)
				mockAIDecisionEngine := setupMockAIDecisionEngineForUserTasks()
				mockGraphExplorer := &MockGraphExplorer{}
				mockAIConversationEngine := &MockAIConversationEngine{}
				mockLearningService := &MockLearningService{}

				orchestrator := NewOrchestratorService(
					mockAIDecisionEngine,
					mockGraphExplorer,
					mockAIConversationEngine,
					mockLearningService,
					logger,
				)

				// Setup: Mock agent context with available agents
				mockGraphExplorer.On("GetAgentContext", mock.Anything).Return(
					"Available agents:\n- text-processor-001 (word-count, text-analysis)\n- calc-agent-001 (math, calculations)", nil)

				// Setup: Mock AI conversation engine to track if it was called
				mockAIConversationEngine.On("ProcessWithAgents", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					"Agent processed the request successfully", nil)

				// Setup: Mock learning service
				mockLearningService.On("StoreInsights", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				request := &OrchestratorRequest{
					UserInput: testCase.userInput,
					UserID:    "test-user",
				}

				result, err := orchestrator.ProcessUserRequest(ctx, request)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				// CRITICAL ASSERTION: AI conversation engine (agent routing) must be called
				if !mockAIConversationEngine.AssertCalled(t, "ProcessWithAgents", mock.Anything, testCase.userInput, "test-user", mock.Anything) {
					t.Errorf("🚨 CRITICAL VIOLATION: User task '%s' was not routed to agents!", testCase.userInput)
					t.Errorf("Description: %s", testCase.description)
					t.Errorf("This violates the core AI-native orchestration principle!")
					t.Errorf("Result: %+v", result)
				}

				// Additional check: Response should not contain direct AI answers
				if result.Success && strings.Contains(strings.ToLower(result.Message), "contains") &&
					strings.Contains(strings.ToLower(result.Message), "word") {
					t.Errorf("🚨 SUSPECTED DIRECT AI RESPONSE: '%s'", result.Message)
					t.Errorf("This appears to be a direct AI answer, not an agent result")
				}
			})
		}
	})

	t.Run("ALLOWED: Orchestrator meta-queries can be answered directly", func(t *testing.T) {
		for _, testCase := range orchestratorMetaQueries {
			t.Run(testCase.name, func(t *testing.T) {
				// Create mocks - use setup for meta-queries (no agents required)
				mockAIDecisionEngine := setupMockAIDecisionEngineForMetaQueries()
				mockGraphExplorer := &MockGraphExplorer{}
				mockAIConversationEngine := &MockAIConversationEngine{}
				mockLearningService := &MockLearningService{}

				orchestrator := NewOrchestratorService(
					mockAIDecisionEngine,
					mockGraphExplorer,
					mockAIConversationEngine,
					mockLearningService,
					logger,
				)

				// Setup: Mock agent context
				mockGraphExplorer.On("GetAgentContext", mock.Anything).Return(
					"Available agents:\n- text-processor-001 (word-count, text-analysis)", nil)

				// Meta-queries should NOT call ProcessWithAgents - they should be handled directly
				// Do NOT setup mockAIConversationEngine.On("ProcessWithAgents") for meta-queries

				// Setup: Mock learning service
				mockLearningService.On("StoreInsights", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				request := &OrchestratorRequest{
					UserInput: testCase.userInput,
					UserID:    "test-user",
				}

				result, err := orchestrator.ProcessUserRequest(ctx, request)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				// Meta-queries can be answered directly by orchestrator
				// This is acceptable and expected behavior
				t.Logf("✅ Meta-query handled appropriately: %s", result.Message)

				// Verify that ProcessWithAgents was NOT called for meta-queries
				mockAIConversationEngine.AssertNotCalled(t, "ProcessWithAgents")
			})
		}
	})
}

// setupMockAIDecisionEngineForUserTasks creates a mock that requires agents (for user tasks)
func setupMockAIDecisionEngineForUserTasks() *MockAIDecisionEngine {
	mockEngine := &MockAIDecisionEngine{}

	// Mock analysis that identifies required agents for user tasks
	analysis := &orchestratorDomain.Analysis{
		Intent:         "user_task_requiring_agent",
		RequiredAgents: []string{"text-processor-001"}, // User tasks require agents
	}

	// Mock decision that forces execution (agent routing)
	decision := &orchestratorDomain.Decision{
		Type:          orchestratorDomain.DecisionTypeExecute, // Force execution through agents
		ExecutionPlan: "Route to appropriate agent for processing",
	}

	mockEngine.On("ExploreAndAnalyze", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(analysis, nil)
	mockEngine.On("MakeDecision", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(decision, nil)

	return mockEngine
}

// setupMockAIDecisionEngineForMetaQueries creates a mock that has no agent requirements (for meta-queries)
func setupMockAIDecisionEngineForMetaQueries() *MockAIDecisionEngine {
	mockEngine := &MockAIDecisionEngine{}

	// Mock analysis that identifies NO required agents for meta-queries
	analysis := &orchestratorDomain.Analysis{
		Intent:         "orchestrator_meta_query",
		RequiredAgents: []string{}, // Meta-queries require NO agents
	}

	// Mock decision that forces execution but without agents
	decision := &orchestratorDomain.Decision{
		Type:          orchestratorDomain.DecisionTypeExecute, // Still execute, but will be handled by meta-query logic
		ExecutionPlan: "Handle meta-query directly by orchestrator",
	}

	mockEngine.On("ExploreAndAnalyze", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(analysis, nil)
	mockEngine.On("MakeDecision", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(decision, nil)

	return mockEngine
}

// MockAIDecisionEngine for testing
type MockAIDecisionEngine struct {
	mock.Mock
}

func (m *MockAIDecisionEngine) ExploreAndAnalyze(ctx context.Context, userInput, userID, agentContext string) (*orchestratorDomain.Analysis, error) {
	args := m.Called(ctx, userInput, userID, agentContext)
	return args.Get(0).(*orchestratorDomain.Analysis), args.Error(1)
}

func (m *MockAIDecisionEngine) MakeDecision(ctx context.Context, userInput, userID string, analysis *orchestratorDomain.Analysis) (*orchestratorDomain.Decision, error) {
	args := m.Called(ctx, userInput, userID, analysis)
	return args.Get(0).(*orchestratorDomain.Decision), args.Error(1)
}
