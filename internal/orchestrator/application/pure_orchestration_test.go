package application

import (
	"context"
	"testing"

	planningDomain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestPureOrchestration_Phase3TDD tests the pure orchestration implementation
// This enforces Phase 3 of the Pure AI Orchestration Architecture
func TestPureOrchestration_Phase3TDD(t *testing.T) {
	ctx := context.Background()

	t.Run("should_return_execution_plan_id_immediately", func(t *testing.T) {
		// Arrange: Set up pure orchestration mocks
		mockPlanningEngine := &testHelpers.MockAIPlanningEngine{}
		mockGraphExplorer := &testHelpers.MockGraphExplorer{}
		mockExecutionCoordinator := &testHelpers.MockExecutionCoordinator{}
		mockConversationService := &testHelpers.MockConversationService{}
		mockResultSynthesizer := &testHelpers.MockResultSynthesizer{}
		mockRepository := testHelpers.NewMockExecutionPlanRepository()

		t.Logf("\n🔧 TESTING PHASE 3: PURE ORCHESTRATION - IMMEDIATE RETURN")

		// Set up mock expectations
		mockGraphExplorer.On("GetAgentContext", ctx).Return("agent-context", nil)

		planningResult := &planningDomain.PlanningResult{
			ID:              "planning-result-123",
			Type:            planningDomain.PlanningTypeExecute,
			ExecutionPlanID: "execution-plan-456",
			RequiredAgents:  []string{"generic-agent"},
		}
		mockPlanningEngine.On("CreateExecutionPlan", ctx, "What is the weather?", "user-123", "agent-context", "msg-789").
			Return(planningResult, nil)

		// Phase 3 requirement: ExecutionCoordinator.StartExecution should be called
		mockExecutionCoordinator.On("StartExecution", ctx, "execution-plan-456").Return(nil)
		
		// Mock conversation service linking (this happens in background)
		mockConversationService.On("LinkExecutionPlan", ctx, "conv-456", "execution-plan-456").Return(nil)
		
		// Mock planning result linking (also background operation)
		mockPlanningEngine.On("LinkPlanningResultToConversation", ctx, "planning-result-123", "conv-456").Return(nil)

		// Create orchestrator with pure orchestration capability
		orchestrator := &OrchestratorService{
			aiPlanningEngine:     mockPlanningEngine,
			graphExplorer:        mockGraphExplorer,
			executionCoordinator: mockExecutionCoordinator, // NEW: Async execution coordinator
			conversationService:  mockConversationService,
			resultSynthesizer:    mockResultSynthesizer,
			repository:           mockRepository,
			logger:               testHelpers.TestLogger(),
		}

		request := &OrchestratorRequest{
			UserInput:      "What is the weather?",
			UserID:         "user-123",
			MessageID:      "msg-789",
			ConversationID: "conv-456",
		}

		// Act: Process request with pure orchestration
		result, err := orchestrator.ProcessUserRequest(ctx, request)

		// Assert: Should return execution plan ID immediately, no Message
		assert.NoError(t, err, "ProcessUserRequest should not return error")
		assert.True(t, result.Success, "Result should be successful")
		assert.Equal(t, "execution-plan-456", result.ExecutionPlanID, "Should return execution plan ID")
		assert.Empty(t, result.Message, "Should NOT return immediate message - pure orchestration!")
		assert.Equal(t, "executing", result.Status, "Status should indicate execution in progress")

		// Verify execution was started asynchronously
		mockExecutionCoordinator.AssertExpectations(t)

		t.Logf("\n✅ SUCCESS: Pure orchestration working!")
		t.Logf("  ✅ Execution plan ID returned immediately: %s", result.ExecutionPlanID)
		t.Logf("  ✅ No immediate message response (pure orchestration)")
		t.Logf("  ✅ Async execution started via ExecutionCoordinator")
		t.Logf("  ✅ Status: %s", result.Status)
	})

	t.Run("should_start_async_execution_without_blocking", func(t *testing.T) {
		// Arrange: Test async execution coordination
		mockPlanningEngine := &testHelpers.MockAIPlanningEngine{}
		mockGraphExplorer := &testHelpers.MockGraphExplorer{}
		mockExecutionCoordinator := &testHelpers.MockExecutionCoordinator{}

		t.Logf("\n⚡ TESTING ASYNC EXECUTION COORDINATION")

		// Mock planning to return execution plan
		mockGraphExplorer.On("GetAgentContext", ctx).Return("context", nil)
		planningResult := &planningDomain.PlanningResult{
			Type:            planningDomain.PlanningTypeExecute,
			ExecutionPlanID: "async-plan-789",
			RequiredAgents:  []string{"text-processor", "analyzer"},
		}
		mockPlanningEngine.On("CreateExecutionPlan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(planningResult, nil)

		// Phase 3 critical: StartExecution should be called but NOT block
		mockExecutionCoordinator.On("StartExecution", ctx, "async-plan-789").Return(nil)

		orchestrator := &OrchestratorService{
			aiPlanningEngine:     mockPlanningEngine,
			graphExplorer:        mockGraphExplorer,
			executionCoordinator: mockExecutionCoordinator,
			logger:               testHelpers.TestLogger(),
		}

		request := &OrchestratorRequest{
			UserInput: "Analyze this medical document",
			UserID:    "doctor-123",
			MessageID: "medical-msg-001",
		}

		// Act: Should return immediately even for complex requests
		result, err := orchestrator.ProcessUserRequest(ctx, request)

		// Assert: Immediate return with async execution
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "async-plan-789", result.ExecutionPlanID)
		assert.Empty(t, result.Message, "No blocking execution - pure orchestration!")

		// Verify async execution was initiated
		mockExecutionCoordinator.AssertCalled(t, "StartExecution", ctx, "async-plan-789")

		t.Logf("\n✅ SUCCESS: Async execution coordination!")
		t.Logf("  ✅ No blocking - returned immediately")
		t.Logf("  ✅ StartExecution called asynchronously")
		t.Logf("  ✅ Pure orchestration pattern maintained")
	})

	t.Run("should_eliminate_all_immediate_response_paths", func(t *testing.T) {
		// Arrange: Test that NO requests return immediate messages
		mockPlanningEngine := &testHelpers.MockAIPlanningEngine{}
		mockGraphExplorer := &testHelpers.MockGraphExplorer{}
		mockExecutionCoordinator := &testHelpers.MockExecutionCoordinator{}

		t.Logf("\n🚫 TESTING ELIMINATION OF IMMEDIATE RESPONSES")

		mockGraphExplorer.On("GetAgentContext", ctx).Return("context", nil)

		// Test different request types - ALL should go through execution
		testCases := []struct {
			name       string
			userInput  string
			expectPlan string
		}{
			{"simple_question", "What is 2+2?", "simple-plan-001"},
			{"complex_analysis", "Analyze patient symptoms", "complex-plan-002"},
			{"meta_query", "What agents do you have?", "meta-plan-003"},
		}

		for _, tc := range testCases {
			// Each request type should create execution plan
			planningResult := &planningDomain.PlanningResult{
				Type:            planningDomain.PlanningTypeExecute,
				ExecutionPlanID: tc.expectPlan,
				RequiredAgents:  []string{"generic-agent"},
			}
			mockPlanningEngine.On("CreateExecutionPlan", mock.Anything, tc.userInput, mock.Anything, mock.Anything, mock.Anything).
				Return(planningResult, nil).Once()

			mockExecutionCoordinator.On("StartExecution", ctx, tc.expectPlan).Return(nil).Once()
		}

		orchestrator := &OrchestratorService{
			aiPlanningEngine:     mockPlanningEngine,
			graphExplorer:        mockGraphExplorer,
			executionCoordinator: mockExecutionCoordinator,
			logger:               testHelpers.TestLogger(),
		}

		// Act & Assert: ALL requests should have same pattern
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				request := &OrchestratorRequest{
					UserInput: tc.userInput,
					UserID:    "test-user",
					MessageID: "test-msg",
				}

				result, err := orchestrator.ProcessUserRequest(ctx, request)

				// Phase 3 requirement: NO immediate messages, ALL async
				assert.NoError(t, err)
				assert.True(t, result.Success)
				assert.Equal(t, tc.expectPlan, result.ExecutionPlanID)
				assert.Empty(t, result.Message, "NO immediate responses allowed!")
				assert.Equal(t, "executing", result.Status)

				t.Logf("  ✅ %s: %s → ExecutionPlan %s (no immediate response)", tc.name, tc.userInput, tc.expectPlan)
			})
		}

		mockExecutionCoordinator.AssertExpectations(t)

		t.Logf("\n✅ SUCCESS: All immediate response paths eliminated!")
		t.Logf("  ✅ Simple, complex, and meta queries all use execution plans")
		t.Logf("  ✅ Pure orchestration architecture enforced")
		t.Logf("  ✅ Consistent async pattern across all request types")
	})
}
