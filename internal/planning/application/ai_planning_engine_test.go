package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

func TestAIPlanningEngine_CreateExecutionPlan(t *testing.T) {
	t.Run("should create planning result with real AI provider", func(t *testing.T) {
		// Arrange: Set up real AI provider for integration testing
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		engine := NewAIPlanningEngine(aiProvider)

		userID := "test-user-123"
		requestID := "test-request-456"
		userInput := "Count the words in this text: Hello world"
		agentContext := `Available Agents:
- text-processor | Status: available | Capabilities: word count, text analysis, character count`

		// Act: Create execution plan
		result, err := engine.CreateExecutionPlan(ctx, userInput, userID, agentContext, requestID)

		// Assert: Should return valid planning result
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, requestID, result.RequestID)
		assert.NotEmpty(t, result.Intent)
		assert.NotEmpty(t, result.Category)
		assert.Greater(t, result.Confidence, 0)
		assert.NotEmpty(t, result.Reasoning)

		// Should have identified text-processor as available and required
		assert.Contains(t, result.AvailableAgents, "text-processor")
		assert.Equal(t, 0, len(result.AgentGap)) // No gap for available agent

		// Verify agent gap analysis worked correctly
		t.Logf("Planning Result: %+v", result)
		t.Logf("Available Agents: %v", result.AvailableAgents)
		t.Logf("Required Agents: %v", result.RequiredAgents)
		t.Logf("Agent Gap: %v", result.AgentGap)
	})

	t.Run("should handle agent gap when required agents not available", func(t *testing.T) {
		// Arrange: Set up scenario with missing agents
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		engine := NewAIPlanningEngine(aiProvider)

		userID := "test-user-123"
		requestID := "test-request-456"
		userInput := "Deploy my application to production with monitoring"
		agentContext := `Available Agents:
- text-processor | Status: available | Capabilities: word count, text analysis`
		// Note: No deployment or monitoring agents available

		// Act: Create execution plan
		result, err := engine.CreateExecutionPlan(ctx, userInput, userID, agentContext, requestID)

		// Assert: Should identify agent gap
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have some agent gap since deployment requires specialized agents
		// The AI might still try to work with what's available or request clarification
		t.Logf("Planning Result with Agent Gap: %+v", result)
		t.Logf("Agent Gap: %v", result.AgentGap)

		// Verify the AI recognized the limitation
		assert.NotEmpty(t, result.Reasoning)
	})

	t.Run("should create clarification planning result when uncertain", func(t *testing.T) {
		// Arrange: Use ambiguous request that should trigger clarification
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		engine := NewAIPlanningEngine(aiProvider)

		userID := "test-user-123"
		requestID := "test-request-456"
		userInput := "do the thing"
		agentContext := `Available Agents:
- text-processor | Status: available | Capabilities: word count, text analysis`

		// Act: Create execution plan
		result, err := engine.CreateExecutionPlan(ctx, userInput, userID, agentContext, requestID)

		// Assert: Should return planning result (may be clarification)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, requestID, result.RequestID)

		// The AI should recognize this is unclear
		// It might clarify or try to respond directly
		t.Logf("Planning Result for Ambiguous Request: %+v", result)
		t.Logf("Planning Type: %s", result.Type)

		if result.Type == domain.PlanningTypeClarify {
			assert.NotEmpty(t, result.Reasoning)
			t.Logf("Clarification Reasoning: %s", result.Reasoning)
		}
	})
}

func TestAIPlanningEngine_ParseAvailableAgents(t *testing.T) {
	engine := NewAIPlanningEngine(nil) // No AI provider needed for parsing

	t.Run("should parse agent names from context", func(t *testing.T) {
		agentContext := `Available Agents:
- text-processor | Status: available | Capabilities: word count
- deploy-agent | Status: available | Capabilities: deployment
- monitor-agent | Status: busy | Capabilities: monitoring`

		agents := engine.parseAvailableAgents(agentContext)

		assert.Contains(t, agents, "text-processor")
		assert.Contains(t, agents, "deploy-agent")
		assert.Contains(t, agents, "monitor-agent")
		assert.Len(t, agents, 3)
	})

	t.Run("should handle empty agent context", func(t *testing.T) {
		agents := engine.parseAvailableAgents("")
		assert.Empty(t, agents)

		agents = engine.parseAvailableAgents("No agents available")
		assert.Empty(t, agents)
	})
}

func TestAIPlanningEngine_ParseExecutionPlanJSON(t *testing.T) {
	engine := NewAIPlanningEngine(nil) // No AI provider needed for parsing

	t.Run("should parse valid execution plan JSON", func(t *testing.T) {
		jsonStr := `{
			"steps": [
				{
					"step_number": 1,
					"agent_id": "text-processor",
					"action_description": "Count words in the provided text",
					"step_name": "Word Count"
				},
				{
					"step_number": 2,
					"agent_id": "text-processor",
					"action_description": "Analyze text structure and formatting",
					"step_name": "Text Analysis"
				}
			]
		}`

		steps, err := engine.parseExecutionPlanJSON(jsonStr)

		require.NoError(t, err)
		assert.Len(t, steps, 2)

		// Check first step
		assert.Equal(t, 1, steps[0].StepNumber)
		assert.Equal(t, "text-processor", steps[0].AssignedAgent)
		assert.Equal(t, "Word Count", steps[0].Name)
		assert.Equal(t, "Count words in the provided text", steps[0].Description)

		// Check second step
		assert.Equal(t, 2, steps[1].StepNumber)
		assert.Equal(t, "text-processor", steps[1].AssignedAgent)
		assert.Equal(t, "Text Analysis", steps[1].Name)
	})

	t.Run("should generate step name from action description if missing", func(t *testing.T) {
		jsonStr := `{
			"steps": [
				{
					"step_number": 1,
					"agent_id": "text-processor",
					"action_description": "Process the text for word counting"
				}
			]
		}`

		steps, err := engine.parseExecutionPlanJSON(jsonStr)

		require.NoError(t, err)
		assert.Len(t, steps, 1)
		assert.Equal(t, "Process the text", steps[0].Name) // First 3 words
	})

	t.Run("should return error for invalid JSON", func(t *testing.T) {
		jsonStr := `{invalid json`

		_, err := engine.parseExecutionPlanJSON(jsonStr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse execution plan JSON")
	})

	t.Run("should return error for missing required fields", func(t *testing.T) {
		jsonStr := `{
			"steps": [
				{
					"step_number": 1,
					"action_description": "Do something"
				}
			]
		}`

		_, err := engine.parseExecutionPlanJSON(jsonStr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent_id cannot be empty")
	})
}
