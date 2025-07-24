package application

import (
	"context"
	"testing"

	orchestratorDomain "neuromesh/internal/orchestrator/domain"
	"neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
)

func TestAIDecisionEngine_ExploreAndAnalyze(t *testing.T) {
	t.Run("should analyze user request with agent context using real AI", func(t *testing.T) {
		aiProvider := testHelpers.SetupRealAIProvider(t)
		engine := NewAIDecisionEngine(aiProvider)

		agentContext := "Agent: deploy-agent | Status: available | Capabilities: deploy, test"
		userInput := "Deploy my application to production"
		userID := "user123"
		requestID := "test-message-123"

		analysis, err := engine.ExploreAndAnalyze(context.Background(), userInput, userID, agentContext, requestID)

		assert.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.NotEmpty(t, analysis.Intent)
		assert.NotEmpty(t, analysis.Category)
		assert.Greater(t, analysis.Confidence, 0)
		assert.Less(t, analysis.Confidence, 101)
		assert.NotEmpty(t, analysis.Reasoning)

		// Since we're using real AI, we can't predict exact responses
		// but we can validate the structure and reasonable expectations
		t.Logf("AI Analysis - Intent: %s, Category: %s, Confidence: %d",
			analysis.Intent, analysis.Category, analysis.Confidence)
	})
}

func TestAIDecisionEngine_MakeDecision(t *testing.T) {
	t.Run("should make decision based on analysis using real AI", func(t *testing.T) {
		aiProvider := testHelpers.SetupRealAIProvider(t)
		engine := NewAIDecisionEngine(aiProvider)

		// Create a clear analysis that should result in execute decision
		requestID := "test-request-123"
		analysis := domain.NewAnalysis(requestID, "deploy_application", "deployment", 90,
			[]string{"deploy-agent"}, "Clear deployment request with specific target")

		decision, err := engine.MakeDecision(context.Background(),
			"Deploy my application to production", "user123", analysis, requestID)

		assert.NoError(t, err)
		assert.NotNil(t, decision)

		// Validate that we get either CLARIFY or EXECUTE decision
		assert.True(t, decision.Type == orchestratorDomain.DecisionTypeClarify ||
			decision.Type == orchestratorDomain.DecisionTypeExecute)

		if decision.Type == orchestratorDomain.DecisionTypeClarify {
			assert.NotEmpty(t, decision.ClarificationQuestion)
			t.Logf("AI Decision: CLARIFY - %s", decision.ClarificationQuestion)
		} else {
			assert.True(t, decision.IsExecutable())
			t.Logf("AI Decision: EXECUTE - ExecutionPlanID: %s", decision.ExecutionPlanID)
		}

		assert.NotEmpty(t, decision.Reasoning)
	})

	t.Run("should handle low confidence request appropriately", func(t *testing.T) {
		aiProvider := testHelpers.SetupRealAIProvider(t)
		engine := NewAIDecisionEngine(aiProvider)

		// Create an unclear analysis
		requestID := "unclear-request-123"
		analysis := domain.NewAnalysis(requestID, "unclear_request", "general", 30,
			[]string{}, "Request is vague and unclear")

		decision, err := engine.MakeDecision(context.Background(),
			"do something", "user123", analysis, requestID)

		assert.NoError(t, err)
		assert.NotNil(t, decision)
		assert.NotEmpty(t, decision.Reasoning)

		// With low confidence, AI might choose to clarify or still try to execute
		// We just validate the response is structured correctly
		t.Logf("AI Decision for unclear request: %s - %s",
			decision.Type, decision.Reasoning)
	})
}

// RED Phase: Tests for structured ExecutionPlan persistence with real AI
func TestAIDecisionEngine_MakeDecision_WithExecutionPlanPersistence(t *testing.T) {
	t.Run("should create and persist structured ExecutionPlan for execute decision using real AI", func(t *testing.T) {
		// This test represents the behavior we want but will fail initially
		// because we haven't implemented the ExecutionPlanRepository integration yet

		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)

		// Set up mock repository for structured plan persistence
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		engine := NewAIDecisionEngineWithRepository(aiProvider, mockRepo)

		// Create a clear analysis that should result in execute decision
		requestID := "test-request-123"
		analysis := domain.NewAnalysis(requestID, "deploy application", "deployment", 95,
			[]string{"deployment-agent", "monitoring-agent"}, "High confidence deployment request")

		decision, err := engine.MakeDecision(ctx,
			"Deploy my application to production with monitoring", "user123", analysis, requestID)

		// These assertions will pass with current implementation
		assert.NoError(t, err)
		assert.NotNil(t, decision)
		assert.NotEmpty(t, decision.Reasoning)

		if decision.Type == orchestratorDomain.DecisionTypeExecute {
			// GREEN: These assertions should now pass with structured ExecutionPlan persistence
			t.Logf("AI Decision: EXECUTE - ExecutionPlanID: '%s'", decision.ExecutionPlanID)
			t.Logf("ExecutionPlanID length: %d", len(decision.ExecutionPlanID))

			// Verify ExecutionPlanID is a proper UUID, not plan text
			assert.Len(t, decision.ExecutionPlanID, 36, "ExecutionPlanID should be a UUID, not execution plan text")
			assert.NotContains(t, decision.ExecutionPlanID, "Step", "ExecutionPlanID should not contain execution plan text")

			// Verify that the ExecutionPlan was created and persisted
			assert.Equal(t, 1, mockRepo.GetPlanCount(), "One ExecutionPlan should have been created")
			assert.Equal(t, 1, mockRepo.GetLinkCount(), "ExecutionPlan should be linked to analysis")

			// Verify repository method calls
			calls := mockRepo.GetCalls()
			assert.Contains(t, calls[0], "Create(", "Create should have been called")
			assert.Contains(t, calls[1], "LinkToAnalysis(", "LinkToAnalysis should have been called")
		} else {
			t.Logf("AI Decision: CLARIFY - %s", decision.ClarificationQuestion)
		}
	})
}
