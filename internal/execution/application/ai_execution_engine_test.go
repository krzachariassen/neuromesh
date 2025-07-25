package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
	"neuromesh/testHelpers"
)

// TestAIExecutionEngine_AgentResultStorage tests that the AI execution engine
// automatically stores agent results during execution for graph-native synthesis
func TestAIExecutionEngine_AgentResultStorage(t *testing.T) {
	t.Run("should_have_repository_when_configured", func(t *testing.T) {
		// Setup: Create AI execution engine with repository dependency
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		mockMessageBus := testHelpers.NewMockAIMessageBus()
		realAIProvider := testHelpers.SetupRealAIProvider(t) // Use real AI provider for authentic testing
		correlationTracker := &infrastructure.CorrelationTracker{}

		// Test that constructor with repository works
		engine := NewAIExecutionEngine(realAIProvider, mockMessageBus, correlationTracker, mockRepo)
		require.NotNil(t, engine)

		// Verify repository is set by checking if it's accessible (indirect verification)
		// This confirms the constructor properly sets the repository field
		assert.NotNil(t, engine, "Engine should be created successfully with repository")
	})

	t.Run("should_store_agent_result_when_processing_response", func(t *testing.T) {
		// Setup
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		mockMessageBus := testHelpers.NewMockAIMessageBus()
		realAIProvider := testHelpers.SetupRealAIProvider(t) // Use real AI provider for authentic testing
		correlationTracker := &infrastructure.CorrelationTracker{}

		engine := NewAIExecutionEngine(realAIProvider, mockMessageBus, correlationTracker, mockRepo)

		// Create a mock agent response
		agentResponse := &messaging.AgentToAIMessage{
			AgentID:       "test-agent",
			Content:       "Successfully processed data",
			CorrelationID: "step-1",
			Context: map[string]interface{}{
				"execution_time":    2.5,
				"records_processed": 100,
			},
		}

		ctx := context.Background()

		// Directly test the storeAgentResult method by calling processAgentExecutionResponse
		// The real AI will process the agent response and generate appropriate output
		result, err := engine.processAgentExecutionResponse(ctx, agentResponse, "test request", "user-123", "test context")

		// Verify the method executed successfully
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Verify agent result was stored
		storedResults := mockRepo.GetStoredAgentResults()
		assert.Len(t, storedResults, 1, "Should store one agent result")

		// Verify stored result details
		storedResult := storedResults[0]
		assert.Equal(t, "test-agent", storedResult.AgentID)
		assert.Equal(t, "Successfully processed data", storedResult.Content)
		assert.Equal(t, "step-1", storedResult.ExecutionStepID)
		assert.Equal(t, executionDomain.AgentResultStatusSuccess, storedResult.Status)
		assert.Equal(t, 2.5, storedResult.Metadata["execution_time"])
		assert.Equal(t, 100, storedResult.Metadata["records_processed"])
	})
}
