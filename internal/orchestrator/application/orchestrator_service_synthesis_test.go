package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executionApplication "neuromesh/internal/execution/application"
	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/logging"
	planningDomain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

// TestOrchestratorService_ResultSynthesis tests orchestrator integration with result synthesis
func TestOrchestratorService_ResultSynthesis(t *testing.T) {
	t.Run("should_create_orchestrator_with_result_synthesizer", func(t *testing.T) {
		// Setup dependencies
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		realAIProvider := testHelpers.SetupRealAIProvider(t)

		// Create all required dependencies
		mockAIDecisionEngine := &MockAIDecisionEngine{}
		mockGraphExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		resultSynthesizer := executionApplication.NewAIResultSynthesizer(realAIProvider, mockRepo)
		logger := logging.NewNoOpLogger()

		// RED: This constructor with synthesizer should include all dependencies
		orchestrator := NewOrchestratorService(mockAIDecisionEngine, mockGraphExplorer, mockExecutionEngine, resultSynthesizer, mockRepo, logger)
		require.NotNil(t, orchestrator)

		// Verify synthesizer is available
		assert.NotNil(t, orchestrator, "Orchestrator should be created with synthesizer")
	})

	t.Run("should_trigger_synthesis_after_execution_completion", func(t *testing.T) {
		// Setup
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		realAIProvider := testHelpers.SetupRealAIProvider(t)
		
		// Create all required dependencies
		mockAIDecisionEngine := &MockAIDecisionEngine{}
		mockGraphExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		resultSynthesizer := executionApplication.NewAIResultSynthesizer(realAIProvider, mockRepo)
		logger := logging.NewNoOpLogger()

		orchestrator := NewOrchestratorService(mockAIDecisionEngine, mockGraphExplorer, mockExecutionEngine, resultSynthesizer, mockRepo, logger)

		// Create test execution plan with completed steps
		planID := "healthcare-synthesis-plan"
		plan := createCompletedHealthcarePlan(planID)
		err := mockRepo.Create(context.Background(), plan)
		require.NoError(t, err)

		// Store completed agent results
		agentResults := createCompletedAgentResults(planID)
		for _, result := range agentResults {
			err := mockRepo.StoreAgentResult(context.Background(), result)
			require.NoError(t, err)
		}

		ctx := context.Background()

		// RED: ProcessWithSynthesis method doesn't exist yet
		synthesizedResult, err := orchestrator.ProcessWithSynthesis(ctx, planID, "Analyze healthcare data for patterns", "user-123")

		// Verify synthesis was triggered and completed
		require.NoError(t, err)
		assert.NotEmpty(t, synthesizedResult, "Should return synthesized result")

		// Verify the result contains synthesized information from multiple agents
		assert.Contains(t, synthesizedResult, "patient", "Should contain healthcare information")
		assert.Greater(t, len(synthesizedResult), 200, "Should be a substantial synthesized response")
	})

	t.Run("should_handle_incomplete_execution_plans", func(t *testing.T) {
		// Setup
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		realAIProvider := testHelpers.SetupRealAIProvider(t)
		
		// Create all required dependencies
		mockAIDecisionEngine := &MockAIDecisionEngine{}
		mockGraphExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		resultSynthesizer := executionApplication.NewAIResultSynthesizer(realAIProvider, mockRepo)
		logger := logging.NewNoOpLogger()

		orchestrator := NewOrchestratorService(mockAIDecisionEngine, mockGraphExplorer, mockExecutionEngine, resultSynthesizer, mockRepo, logger)

		// Create execution plan with incomplete steps
		planID := "incomplete-plan"
		plan := createIncompleteHealthcarePlan(planID)
		err := mockRepo.Create(context.Background(), plan)
		require.NoError(t, err)

		// Store only partial agent results
		partialResults := createPartialAgentResults(planID)
		for _, result := range partialResults {
			err := mockRepo.StoreAgentResult(context.Background(), result)
			require.NoError(t, err)
		}

		ctx := context.Background()

		// Should handle incomplete execution gracefully
		result, err := orchestrator.ProcessWithSynthesis(ctx, planID, "Process incomplete execution", "user-123")

		// Should return some result even with incomplete data
		require.NoError(t, err)
		assert.NotEmpty(t, result, "Should return result even with partial data")
	})

	t.Run("should_detect_execution_completion", func(t *testing.T) {
		// Setup
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		realAIProvider := testHelpers.SetupRealAIProvider(t)
		
		// Create all required dependencies
		mockAIDecisionEngine := &MockAIDecisionEngine{}
		mockGraphExplorer := &MockGraphExplorer{}
		mockExecutionEngine := &MockAIExecutionEngine{}
		resultSynthesizer := executionApplication.NewAIResultSynthesizer(realAIProvider, mockRepo)
		logger := logging.NewNoOpLogger()

		orchestrator := NewOrchestratorService(mockAIDecisionEngine, mockGraphExplorer, mockExecutionEngine, resultSynthesizer, mockRepo, logger)

		planID := "completion-detection-plan"
		plan := createCompletedHealthcarePlan(planID)
		err := mockRepo.Create(context.Background(), plan)
		require.NoError(t, err)

		ctx := context.Background()

		// RED: IsExecutionComplete method doesn't exist yet
		isComplete, err := orchestrator.IsExecutionComplete(ctx, planID)

		require.NoError(t, err)
		assert.True(t, isComplete, "Should detect when all steps are complete")
	})
}

// Helper functions for test setup

func createCompletedHealthcarePlan(planID string) *planningDomain.ExecutionPlan {
	return &planningDomain.ExecutionPlan{
		ID:     planID,
		Name:   "Healthcare Analysis Plan",
		Status: planningDomain.ExecutionPlanStatusCompleted,
		Steps: []*planningDomain.ExecutionStep{
			{
				ID:            "step-1",
				PlanID:        planID,
				StepNumber:    1,
				Description:   "Collect patient data",
				AssignedAgent: "data-collector-agent",
				Status:        planningDomain.ExecutionStepStatusCompleted,
			},
			{
				ID:            "step-2",
				PlanID:        planID,
				StepNumber:    2,
				Description:   "Analyze patterns",
				AssignedAgent: "pattern-analysis-agent",
				Status:        planningDomain.ExecutionStepStatusCompleted,
			},
			{
				ID:            "step-3",
				PlanID:        planID,
				StepNumber:    3,
				Description:   "Assess risks",
				AssignedAgent: "risk-assessment-agent",
				Status:        planningDomain.ExecutionStepStatusCompleted,
			},
		},
	}
}

func createIncompleteHealthcarePlan(planID string) *planningDomain.ExecutionPlan {
	return &planningDomain.ExecutionPlan{
		ID:     planID,
		Name:   "Incomplete Healthcare Plan",
		Status: planningDomain.ExecutionPlanStatusExecuting,
		Steps: []*planningDomain.ExecutionStep{
			{
				ID:            "step-1",
				PlanID:        planID,
				StepNumber:    1,
				Description:   "Collect data",
				AssignedAgent: "data-collector",
				Status:        planningDomain.ExecutionStepStatusCompleted,
			},
			{
				ID:            "step-2",
				PlanID:        planID,
				StepNumber:    2,
				Description:   "Process data",
				AssignedAgent: "processor",
				Status:        planningDomain.ExecutionStepStatusExecuting,
			},
		},
	}
}

func createCompletedAgentResults(planID string) []*executionDomain.AgentResult {
	return []*executionDomain.AgentResult{
		{
			ID:              "result-1",
			ExecutionStepID: "step-1",
			AgentID:         "data-collector-agent",
			Content:         "Successfully collected 1,250 patient records with comprehensive health data",
			Status:          executionDomain.AgentResultStatusSuccess,
			Timestamp:       time.Now(),
		},
		{
			ID:              "result-2",
			ExecutionStepID: "step-2",
			AgentID:         "pattern-analysis-agent",
			Content:         "Identified 5 significant patterns in cardiovascular health across age groups",
			Status:          executionDomain.AgentResultStatusSuccess,
			Timestamp:       time.Now(),
		},
		{
			ID:              "result-3",
			ExecutionStepID: "step-3",
			AgentID:         "risk-assessment-agent",
			Content:         "Risk assessment completed: 278 high-risk patients identified with actionable recommendations",
			Status:          executionDomain.AgentResultStatusSuccess,
			Timestamp:       time.Now(),
		},
	}
}

func createPartialAgentResults(planID string) []*executionDomain.AgentResult {
	return []*executionDomain.AgentResult{
		{
			ID:              "result-1",
			ExecutionStepID: "step-1",
			AgentID:         "data-collector",
			Content:         "Collected 500 records successfully",
			Status:          executionDomain.AgentResultStatusSuccess,
			Timestamp:       time.Now(),
		},
	}
}
