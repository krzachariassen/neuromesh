package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/orchestrator/infrastructure"
	planningDomain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

// TestAIExecutionEngine_AgentResultStorage tests that the AI execution engine
// automatically stores agent results during execution for graph-native synthesis
func TestAIExecutionEngine_AgentResultStorage(t *testing.T) {
	t.Run("should_store_agent_results_during_execution", func(t *testing.T) {
		// Setup: Create AI execution engine with repository dependency
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		mockMessageBus := testHelpers.NewMockAIMessageBus()
		mockAIProvider := testHelpers.NewMockAIProvider()
		correlationTracker := &infrastructure.CorrelationTracker{}

		// RED: This constructor doesn't exist yet - need to add repository dependency
		engine := NewAIExecutionEngineWithRepository(mockAIProvider, mockMessageBus, correlationTracker, mockRepo)
		require.NotNil(t, engine)

		// Create a test execution plan with multiple steps
		plan := createTestExecutionPlan()

		ctx := context.Background()

		// Execute: Run the execution engine with a simple plan
		executionPlan := createExecutionPlanText(plan)
		result, err := engine.ExecuteWithAgents(ctx, executionPlan, "Analyze healthcare data for patterns", "user-123", "Healthcare analysis context")

		// Verify: Execution completed successfully
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// RED: The key failing assertion - agent results should be stored automatically
		// This will fail because the current engine doesn't store results yet
		storedResults := mockRepo.GetStoredAgentResults()
		assert.Len(t, storedResults, 2, "Should store agent results for both execution steps")

		// Verify that results contain expected step IDs
		stepIDs := extractStepIDs(storedResults)
		assert.Contains(t, stepIDs, "step-1", "Should store result for step-1")
		assert.Contains(t, stepIDs, "step-2", "Should store result for step-2")
	})
}

// Helper function to create test execution plan
func createTestExecutionPlan() *planningDomain.ExecutionPlan {
	return &planningDomain.ExecutionPlan{
		ID:   "plan-123",
		Name: "Healthcare Data Analysis",
		Steps: []*planningDomain.ExecutionStep{
			{
				ID:            "step-1",
				PlanID:        "plan-123",
				StepNumber:    1,
				Description:   "Collect patient data",
				AssignedAgent: "data-collector-agent",
				Status:        planningDomain.ExecutionStepStatusPending,
			},
			{
				ID:            "step-2",
				PlanID:        "plan-123",
				StepNumber:    2,
				Description:   "Analyze data patterns",
				AssignedAgent: "analysis-agent",
				Status:        planningDomain.ExecutionStepStatusPending,
			},
		},
	}
}

// Helper function to create execution plan text representation
func createExecutionPlanText(plan *planningDomain.ExecutionPlan) string {
	text := "Execution Plan: " + plan.Name + "\n"
	text += "Plan ID: " + plan.ID + "\n\n"

	for _, step := range plan.Steps {
		text += "Step " + string(rune(step.StepNumber+'0')) + ": " + step.Description + "\n"
		text += "Agent: " + step.AssignedAgent + "\n"
		text += "Status: " + string(step.Status) + "\n\n"
	}

	return text
}

// Helper function to extract step IDs from agent results
func extractStepIDs(results []*executionDomain.AgentResult) []string {
	stepIDs := make([]string, len(results))
	for i, result := range results {
		stepIDs[i] = result.ExecutionStepID
	}
	return stepIDs
}
