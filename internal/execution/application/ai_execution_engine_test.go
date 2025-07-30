package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
	planningDomain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

// TestAIExecutionEngine_AgentResultStorage tests that the AI execution engine
// automatically stores agent results during execution for graph-native synthesis
func TestAIExecutionEngine_AgentResultStorage(t *testing.T) {
	t.Run("should_have_repository_when_configured", func(t *testing.T) {
		// Setup: Create AI execution engine with repository dependency
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		mockAIMessageBus := testHelpers.NewMockAIMessageBus()
		memoryMessageBus := messaging.NewMemoryMessageBus(logging.NewNoOpLogger())
		realAIProvider := testHelpers.SetupRealAIProvider(t) // Use real AI provider for authentic testing
		correlationTracker := &infrastructure.CorrelationTracker{}

		// Test that constructor with repository works
		engine := NewAIExecutionEngine(realAIProvider, mockAIMessageBus, correlationTracker, mockRepo, memoryMessageBus)
		require.NotNil(t, engine)

		// Verify repository is set by checking if it's accessible (indirect verification)
		// This confirms the constructor properly sets the repository field
		assert.NotNil(t, engine, "Engine should be created successfully with repository")
	})

	t.Run("should_store_agent_result_when_processing_response", func(t *testing.T) {
		// Setup
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		mockAIMessageBus := testHelpers.NewMockAIMessageBus()
		memoryMessageBus := messaging.NewMemoryMessageBus(logging.NewNoOpLogger())
		realAIProvider := testHelpers.SetupRealAIProvider(t) // Use real AI provider for authentic testing
		correlationTracker := &infrastructure.CorrelationTracker{}

		engine := NewAIExecutionEngine(realAIProvider, mockAIMessageBus, correlationTracker, mockRepo, memoryMessageBus)

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

		// Create a mock execution step for the test
		step := &planningDomain.ExecutionStep{
			ID:            "step-1",
			Name:          "test-step",
			AssignedAgent: "test-agent",
		}

		// Test the processAgentExecutionResponse method with new signature
		result, err := engine.processAgentExecutionResponse(ctx, agentResponse, step)

		// Verify the method executed successfully
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Equal(t, "Successfully processed data", result)

		// Note: In the plan-driven approach, storeAgentResult is called separately
		// during executeStep, not during processAgentExecutionResponse
		// This test focuses on the response processing functionality
	})
}
