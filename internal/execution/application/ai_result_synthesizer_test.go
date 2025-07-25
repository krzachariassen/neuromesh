package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executionDomain "neuromesh/internal/execution/domain"
	planningDomain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

// TestAIResultSynthesizer tests the AI-powered result synthesizer
func TestAIResultSynthesizer(t *testing.T) {
	t.Run("should_create_synthesizer_with_dependencies", func(t *testing.T) {
		// Setup: Real AI provider and mock repository
		realAIProvider := testHelpers.SetupRealAIProvider(t)
		mockRepo := testHelpers.NewMockExecutionPlanRepository()

		// RED: AIResultSynthesizer doesn't exist yet
		synthesizer := NewAIResultSynthesizer(realAIProvider, mockRepo)
		require.NotNil(t, synthesizer)

		// Verify it implements the interface
		var _ executionDomain.ResultSynthesizer = synthesizer
	})

	t.Run("should_synthesize_results_using_real_AI", func(t *testing.T) {
		// Setup
		realAIProvider := testHelpers.SetupRealAIProvider(t)
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		synthesizer := NewAIResultSynthesizer(realAIProvider, mockRepo)

		// Create test execution plan with agent results
		planID := "healthcare-plan-123"
		
		// First create and store the execution plan with steps
		plan := createTestExecutionPlan(planID)
		err := mockRepo.Create(context.Background(), plan)
		require.NoError(t, err)
		
		agentResults := createHealthcareAgentResults(planID)

		// Store agent results in mock repository
		for _, result := range agentResults {
			err := mockRepo.StoreAgentResult(context.Background(), result)
			require.NoError(t, err)
		}

		ctx := context.Background()

		// RED: SynthesizeResults method doesn't exist yet
		synthesizedResult, err := synthesizer.SynthesizeResults(ctx, planID)

		// Verify synthesis completed successfully
		require.NoError(t, err)
		assert.NotEmpty(t, synthesizedResult, "Synthesized result should not be empty")

		// Verify synthesized result contains information from multiple agents
		assert.Contains(t, synthesizedResult, "patient", "Should reference patient data")
		assert.Contains(t, synthesizedResult, "analysis", "Should reference analysis results")

		// Verify it's a cohesive response, not just concatenated results
		assert.Greater(t, len(synthesizedResult), 100, "Should be a substantial synthesized response")
	})

	t.Run("should_get_synthesis_context", func(t *testing.T) {
		// Setup
		realAIProvider := testHelpers.SetupRealAIProvider(t)
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		synthesizer := NewAIResultSynthesizer(realAIProvider, mockRepo)

		// Create test data
		planID := "plan-456"
		
		// Create and store execution plan with steps
		plan := createTestExecutionPlan(planID)
		err := mockRepo.Create(context.Background(), plan)
		require.NoError(t, err)
		
		agentResults := createHealthcareAgentResults(planID)

		// Store agent results
		for _, result := range agentResults {
			err := mockRepo.StoreAgentResult(context.Background(), result)
			require.NoError(t, err)
		}

		ctx := context.Background()

		// RED: GetSynthesisContext method doesn't exist yet
		synthCtx, err := synthesizer.GetSynthesisContext(ctx, planID)

		// Verify context retrieval
		require.NoError(t, err)
		require.NotNil(t, synthCtx)
		assert.Equal(t, planID, synthCtx.ExecutionPlanID)
		assert.Len(t, synthCtx.AgentResults, 3, "Should have all 3 agent results")
		assert.False(t, synthCtx.CreatedAt.IsZero())
	})

	t.Run("should_handle_partial_results", func(t *testing.T) {
		// Setup with mixed success/failure results
		realAIProvider := testHelpers.SetupRealAIProvider(t)
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		synthesizer := NewAIResultSynthesizer(realAIProvider, mockRepo)

		planID := "partial-plan-789"
		
		// Create execution plan with steps
		plan := &planningDomain.ExecutionPlan{
			ID:   planID,
			Name: "Partial Results Test",
			Steps: []*planningDomain.ExecutionStep{
				{
					ID:            "step-1",
					PlanID:        planID,
					StepNumber:    1,
					Description:   "Collect data",
					AssignedAgent: "data-collector",
					Status:        planningDomain.ExecutionStepStatusPending,
				},
				{
					ID:            "step-2",
					PlanID:        planID,
					StepNumber:    2,
					Description:   "Analyze data",
					AssignedAgent: "analyzer",
					Status:        planningDomain.ExecutionStepStatusPending,
				},
			},
		}
		
		err := mockRepo.Create(context.Background(), plan)
		require.NoError(t, err)
		
		agentResults := []*executionDomain.AgentResult{
			{
				ID:              "result-1",
				ExecutionStepID: "step-1",
				AgentID:         "data-collector",
				Content:         "Successfully collected 1,250 patient records",
				Status:          executionDomain.AgentResultStatusSuccess,
				Timestamp:       time.Now(),
			},
			{
				ID:              "result-2",
				ExecutionStepID: "step-2",
				AgentID:         "analyzer",
				Content:         "Analysis failed: insufficient data quality",
				Status:          executionDomain.AgentResultStatusFailed,
				Timestamp:       time.Now(),
			},
		}

		// Store mixed results
		for _, result := range agentResults {
			err := mockRepo.StoreAgentResult(context.Background(), result)
			require.NoError(t, err)
		}

		ctx := context.Background()

		// Should still synthesize even with partial results
		synthesizedResult, err := synthesizer.SynthesizeResults(ctx, planID)

		require.NoError(t, err)
		assert.NotEmpty(t, synthesizedResult)

		// Should acknowledge both success and failure
		assert.Contains(t, synthesizedResult, "collected", "Should mention successful data collection")
		assert.Contains(t, synthesizedResult, "analysis", "Should address analysis issue")
	})
}

// Helper function to create test execution plan with steps
func createTestExecutionPlan(planID string) *planningDomain.ExecutionPlan {
	return &planningDomain.ExecutionPlan{
		ID:   planID,
		Name: "Healthcare Data Analysis",
		Steps: []*planningDomain.ExecutionStep{
			{
				ID:            "step-1",
				PlanID:        planID,
				StepNumber:    1,
				Description:   "Collect patient data",
				AssignedAgent: "data-collector-agent",
				Status:        planningDomain.ExecutionStepStatusPending,
			},
			{
				ID:            "step-2",
				PlanID:        planID,
				StepNumber:    2,
				Description:   "Analyze patterns",
				AssignedAgent: "pattern-analysis-agent",
				Status:        planningDomain.ExecutionStepStatusPending,
			},
			{
				ID:            "step-3",
				PlanID:        planID,
				StepNumber:    3,
				Description:   "Assess risks",
				AssignedAgent: "risk-assessment-agent",
				Status:        planningDomain.ExecutionStepStatusPending,
			},
		},
	}
}

// Helper function to create realistic healthcare agent results
func createHealthcareAgentResults(planID string) []*executionDomain.AgentResult {
	return []*executionDomain.AgentResult{
		{
			ID:              "result-data-collection",
			ExecutionStepID: "step-1",
			AgentID:         "data-collector-agent",
			Content:         "Successfully collected 1,250 patient records with demographics, vital signs, and medical history. Data quality is high with 98% completeness rate.",
			Status:          executionDomain.AgentResultStatusSuccess,
			Metadata: map[string]interface{}{
				"records_collected": 1250,
				"data_quality":      "high",
				"completeness_rate": 0.98,
			},
			Timestamp: time.Now(),
		},
		{
			ID:              "result-pattern-analysis",
			ExecutionStepID: "step-2",
			AgentID:         "pattern-analysis-agent",
			Content:         "Identified 5 significant patterns in blood pressure trends across different age groups. Notable finding: 35% increase in hypertension cases in patients over 50.",
			Status:          executionDomain.AgentResultStatusSuccess,
			Metadata: map[string]interface{}{
				"patterns_found":      5,
				"hypertension_rate":   0.35,
				"confidence_level":    0.87,
			},
			Timestamp: time.Now(),
		},
		{
			ID:              "result-risk-assessment",
			ExecutionStepID: "step-3",
			AgentID:         "risk-assessment-agent",
			Content:         "Risk assessment completed. High-risk patients identified: 278 individuals with multiple cardiovascular risk factors. Recommend immediate follow-up for 45 critical cases.",
			Status:          executionDomain.AgentResultStatusSuccess,
			Metadata: map[string]interface{}{
				"high_risk_patients": 278,
				"critical_cases":     45,
				"risk_score_avg":     6.8,
			},
			Timestamp: time.Now(),
		},
	}
}
