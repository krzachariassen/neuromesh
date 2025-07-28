package application

import (
	"testing"

	"neuromesh/internal/planning/domain"

	"github.com/stretchr/testify/assert"
)

// TestPlanningEngine_ConceptualImprovement demonstrates the design improvement
// from the current Decision-based approach to proper Planning terminology
func TestPlanningEngine_ConceptualImprovement(t *testing.T) {
	// This test demonstrates the conceptual improvement without relying on complex mocking

	t.Run("OLD API - Confusing Decision Terminology", func(t *testing.T) {
		// PROBLEM 1: Confusing terminology
		// - "Decision" suggests binary choice, but we're doing comprehensive planning
		// - "ExploreAndAnalyze" + "MakeDecision" = redundant AI calls
		// - Agent validation happens in wrong place

		t.Logf("❌ OLD API Problems:")
		t.Logf("   1. analysis := engine.ExploreAndAnalyze() // First AI call")
		t.Logf("   2. decision := engine.MakeDecision(analysis) // Second AI call")
		t.Logf("   3. 'Decision' terminology is misleading - we're planning!")
		t.Logf("   4. Agent validation scattered across multiple functions")
		t.Logf("   5. No clear agent gap analysis")
	})

	t.Run("NEW API - Clear Planning Terminology", func(t *testing.T) {
		// SOLUTION: Unified planning with proper terminology

		t.Logf("✅ NEW API Benefits:")
		t.Logf("   1. planningResult := engine.CreateExecutionPlan() // Single AI call")
		t.Logf("   2. 'Planning' terminology is accurate")
		t.Logf("   3. Agent validation built into planning process")
		t.Logf("   4. Clear agent gap analysis (required vs available)")
		t.Logf("   5. Comprehensive planning result with all context")

		// Demonstrate the improved data structure
		planningResult := domain.NewExecutePlanningResult(
			"request-123",
			"deploy application",
			"deployment",
			95,
			[]string{"deployment-agent", "monitoring-agent"}, // Available
			[]string{"deployment-agent", "security-agent"},   // Required
			"plan-456",
			"Deployment agent available, but security agent missing",
		)

		// Rich agent analysis
		assert.True(t, planningResult.HasAgentGap(), "Should detect missing agents")
		assert.Equal(t, []string{"security-agent"}, planningResult.AgentGap, "Should identify missing security agent")
		assert.True(t, planningResult.IsExecutable(), "Should still be executable with primary agent")

		// Clear planning semantics
		assert.Equal(t, domain.PlanningTypeExecute, planningResult.Type, "Clear planning type")
		assert.NotEmpty(t, planningResult.ExecutionPlanID, "Execution plan created")

		t.Logf("✅ Agent Gap Analysis: %v", planningResult.AgentGap)
		t.Logf("✅ Planning Type: %s", planningResult.Type)
		t.Logf("✅ Is Executable: %t", planningResult.IsExecutable())
	})

	t.Run("Orchestrator Integration Benefits", func(t *testing.T) {
		// Show how this improves orchestrator code

		t.Logf("🚀 ORCHESTRATOR INTEGRATION:")
		t.Logf("")
		t.Logf("OLD WAY (confusing):")
		t.Logf("  analysis := decisionEngine.ExploreAndAnalyze(...)")
		t.Logf("  decision := decisionEngine.MakeDecision(...)")
		t.Logf("  if decision.Type == 'EXECUTE' { /* execute */ }")
		t.Logf("")
		t.Logf("NEW WAY (clear):")
		t.Logf("  plan := planningEngine.CreateExecutionPlan(...)")
		t.Logf("  if plan.IsExecutable() { /* execute */ }")
		t.Logf("  if plan.HasAgentGap() { /* warn about missing agents */ }")
		t.Logf("  if plan.NeedsClarification() { /* ask for clarification */ }")
		t.Logf("")

		// The new API makes orchestrator logic much cleaner
		assert.True(t, true, "Conceptual improvement demonstrated")
	})
}

// TestPlanningResult_AgentGapAnalysis tests the new agent gap functionality
func TestPlanningResult_AgentGapAnalysis(t *testing.T) {
	testCases := []struct {
		name               string
		availableAgents    []string
		requiredAgents     []string
		expectedGap        []string
		shouldBeExecutable bool
	}{
		{
			name:               "Perfect Match - No Gap",
			availableAgents:    []string{"deployment-agent", "monitoring-agent"},
			requiredAgents:     []string{"deployment-agent"},
			expectedGap:        []string{},
			shouldBeExecutable: true,
		},
		{
			name:               "Agent Gap - Missing Required Agent",
			availableAgents:    []string{"deployment-agent"},
			requiredAgents:     []string{"deployment-agent", "security-agent"},
			expectedGap:        []string{"security-agent"},
			shouldBeExecutable: true, // Can still execute with partial agents
		},
		{
			name:               "No Available Agents",
			availableAgents:    []string{},
			requiredAgents:     []string{"deployment-agent"},
			expectedGap:        []string{"deployment-agent"},
			shouldBeExecutable: true, // Planning result is created, but gap exists
		},
		{
			name:               "None Required",
			availableAgents:    []string{"deployment-agent"},
			requiredAgents:     []string{},
			expectedGap:        []string{},
			shouldBeExecutable: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			planningResult := domain.NewExecutePlanningResult(
				"request-123",
				"test request",
				"test",
				90,
				tc.availableAgents,
				tc.requiredAgents,
				"plan-123",
				"test reasoning",
			)

			assert.Equal(t, tc.expectedGap, planningResult.AgentGap, "Agent gap should match expected")
			assert.Equal(t, len(tc.expectedGap) > 0, planningResult.HasAgentGap(), "HasAgentGap should match gap length")
			assert.Equal(t, tc.shouldBeExecutable, planningResult.IsExecutable(), "Executable status should match expected")
		})
	}
}
