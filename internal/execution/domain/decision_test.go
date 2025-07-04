package domain

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestDecision_ExecuteCreation(t *testing.T) {
	t.Run("should create execute decision", func(t *testing.T) {
		executionPlan := "Step 1: Deploy to staging\nStep 2: Run tests"
		coordination := "Primary Agent: deploy-agent"
		decision := NewExecuteDecision(executionPlan, coordination, "Clear deployment request with sufficient details")
		assert.Equal(t, DecisionTypeExecute, decision.Type)
		assert.Equal(t, executionPlan, decision.ExecutionPlan)
		assert.Equal(t, coordination, decision.AgentCoordination)
		assert.Equal(t, "Clear deployment request with sufficient details", decision.Reasoning)
	})
}

func TestDecision_IsExecutable(t *testing.T) {
	t.Run("should return true for execute decision", func(t *testing.T) {
		decision := NewExecuteDecision("test plan", "test coordination", "test reasoning")
		assert.True(t, decision.IsExecutable())
	})
}

func TestDecision_HasAction(t *testing.T) {
	t.Run("should return false for decision without action", func(t *testing.T) {
		decision := NewExecuteDecision("test plan", "test coordination", "test reasoning")
		assert.False(t, decision.HasAction())
	})

	t.Run("should return true for decision with action", func(t *testing.T) {
		decision := &Decision{Type: DecisionTypeExecute, Action: "deploy"}
		assert.True(t, decision.HasAction())
	})
}
