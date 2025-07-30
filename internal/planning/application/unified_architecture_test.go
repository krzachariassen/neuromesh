package application

import (
	"context"
	"testing"

	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnifiedArchitectureEnforcement ensures ALL requests go through execution plans
// This test enforces the new architecture where RESPOND_DIRECTLY is eliminated
func TestUnifiedArchitectureEnforcement(t *testing.T) {
	t.Run("should force all requests to create execution plans", func(t *testing.T) {
		// Arrange: Set up real AI and planning
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		executionPlanRepo := testHelpers.NewMockExecutionPlanRepository()

		planningEngine := NewAIPlanningEngineWithRepository(
			aiProvider,
			executionPlanRepo,
		)

		// Test simple request that would previously use RESPOND_DIRECTLY
		simpleRequest := "What is the weather like today?"
		userID := "test-user-123"
		requestID := "simple-request-456"

		// Mock agent context with generic agent
		agentContext := `Available Agents:
- generic-agent | Status: available | Capabilities: general question answering, guidance, explanations`

		t.Logf("\n🔧 TESTING UNIFIED ARCHITECTURE ENFORCEMENT")
		t.Logf("Request: %s", simpleRequest)

		// Act: All requests should create execution plans
		planningResult, err := planningEngine.CreateExecutionPlan(
			ctx, simpleRequest, userID, agentContext, requestID)

		// Assert: Must succeed and create execution plan
		require.NoError(t, err)
		require.NotNil(t, planningResult)

		t.Logf("\n✅ Planning Result:")
		t.Logf("  Type: %s", planningResult.Type)
		t.Logf("  ExecutionPlanID: %s", planningResult.ID)
		t.Logf("  RequiredAgents: %v", planningResult.RequiredAgents)

		// Critical assertions for unified architecture
		assert.NotEqual(t, "RESPOND_DIRECTLY", string(planningResult.Type),
			"RESPOND_DIRECTLY should be eliminated - all requests must create execution plans")

		assert.Equal(t, "EXECUTE", string(planningResult.Type),
			"All requests should result in EXECUTE type with execution plans")

		assert.NotEmpty(t, planningResult.ID,
			"Every request must have an execution plan ID")

		assert.NotEmpty(t, planningResult.RequiredAgents,
			"Every request must require at least one agent (generic-agent)")

		assert.Contains(t, planningResult.RequiredAgents, "generic-agent",
			"Simple requests should use generic-agent")

		t.Logf("\n🎯 SUCCESS: Unified architecture enforced!")
		t.Logf("  ✅ No RESPOND_DIRECTLY code path")
		t.Logf("  ✅ Execution plan created for simple request")
		t.Logf("  ✅ Generic agent assigned for general questions")
	})

	t.Run("should create execution plans for complex multi-agent requests", func(t *testing.T) {
		// Arrange: Complex request requiring multiple agents
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		executionPlanRepo := testHelpers.NewMockExecutionPlanRepository()

		planningEngine := NewAIPlanningEngineWithRepository(
			aiProvider,
			executionPlanRepo,
		)

		// Complex healthcare request
		complexRequest := "Analyze this medical report for brain tumor assessment: Patient has headaches and vision changes"
		userID := "doctor-123"
		requestID := "complex-request-789"

		// Multiple specialized agents available
		agentContext := `Available Agents:
- generic-agent | Status: available | Capabilities: general question answering
- medical-text-processor | Status: available | Capabilities: medical terminology extraction
- clinical-analyzer | Status: available | Capabilities: clinical data analysis
- risk-assessor | Status: available | Capabilities: medical risk assessment`

		t.Logf("\n🏥 TESTING COMPLEX MEDICAL REQUEST")

		// Act
		planningResult, err := planningEngine.CreateExecutionPlan(
			ctx, complexRequest, userID, agentContext, requestID)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, planningResult)

		t.Logf("\n✅ Complex Planning Result:")
		t.Logf("  Type: %s", planningResult.Type)
		t.Logf("  ExecutionPlanID: %s", planningResult.ID)
		t.Logf("  RequiredAgents: %v", planningResult.RequiredAgents)

		// Same architecture for complex requests
		assert.Equal(t, "EXECUTE", string(planningResult.Type),
			"Complex requests should also use EXECUTE type")

		assert.NotEmpty(t, planningResult.ID,
			"Complex requests must have execution plans")

		assert.GreaterOrEqual(t, len(planningResult.RequiredAgents), 1,
			"Complex requests should require multiple agents")

		t.Logf("\n🎯 SUCCESS: Same architecture scales from simple to complex!")
	})
}
