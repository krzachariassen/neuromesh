package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResultSynthesizer tests the result synthesizer interface
func TestResultSynthesizer(t *testing.T) {
	t.Run("should_define_synthesizer_interface", func(t *testing.T) {
		// RED: This interface doesn't exist yet
		var synthesizer ResultSynthesizer
		assert.NotNil(t, &synthesizer, "ResultSynthesizer interface should be defined")
	})

	t.Run("should_define_synthesis_context", func(t *testing.T) {
		// RED: SynthesisContext doesn't exist yet
		ctx := &SynthesisContext{
			ExecutionPlanID: "plan-123",
			AgentResults:    []*AgentResult{},
			CreatedAt:       time.Now(),
		}

		require.NotNil(t, ctx)
		assert.Equal(t, "plan-123", ctx.ExecutionPlanID)
		assert.Empty(t, ctx.AgentResults)
		assert.False(t, ctx.CreatedAt.IsZero())
	})

	t.Run("should_validate_synthesis_context_fields", func(t *testing.T) {
		// RED: SynthesisContext validation doesn't exist yet
		ctx := &SynthesisContext{
			ExecutionPlanID: "",
			AgentResults:    nil,
		}

		err := ctx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ExecutionPlanID")

		// Valid context should pass validation
		validCtx := &SynthesisContext{
			ExecutionPlanID: "plan-123",
			AgentResults:    []*AgentResult{},
			CreatedAt:       time.Now(),
		}

		err = validCtx.Validate()
		assert.NoError(t, err)
	})

	t.Run("should_support_synthesis_metadata", func(t *testing.T) {
		// RED: SynthesisContext with metadata support doesn't exist yet
		ctx := &SynthesisContext{
			ExecutionPlanID: "plan-123",
			AgentResults:    []*AgentResult{},
			CreatedAt:       time.Now(),
			Metadata: map[string]interface{}{
				"user_request":   "Analyze healthcare data",
				"execution_type": "multi_agent",
				"priority":       "high",
			},
		}

		require.NotNil(t, ctx.Metadata)
		assert.Equal(t, "Analyze healthcare data", ctx.Metadata["user_request"])
		assert.Equal(t, "multi_agent", ctx.Metadata["execution_type"])
		assert.Equal(t, "high", ctx.Metadata["priority"])
	})

	t.Run("should_create_synthesis_context_with_constructor", func(t *testing.T) {
		// Test the new constructor
		results := []*AgentResult{
			{ID: "result-1", AgentID: "agent-1", Status: AgentResultStatusSuccess},
			{ID: "result-2", AgentID: "agent-2", Status: AgentResultStatusFailed},
		}

		ctx := NewSynthesisContext("plan-123", results)
		require.NotNil(t, ctx)
		assert.Equal(t, "plan-123", ctx.ExecutionPlanID)
		assert.Len(t, ctx.AgentResults, 2)
		assert.NotNil(t, ctx.Metadata)
		assert.False(t, ctx.CreatedAt.IsZero())
	})

	t.Run("should_manage_metadata", func(t *testing.T) {
		ctx := NewSynthesisContext("plan-123", []*AgentResult{})

		ctx.AddMetadata("user_request", "Analyze healthcare data")
		ctx.AddMetadata("priority", "high")

		assert.Equal(t, "Analyze healthcare data", ctx.Metadata["user_request"])
		assert.Equal(t, "high", ctx.Metadata["priority"])
	})

	t.Run("should_filter_successful_results", func(t *testing.T) {
		results := []*AgentResult{
			{ID: "result-1", AgentID: "agent-1", Status: AgentResultStatusSuccess},
			{ID: "result-2", AgentID: "agent-2", Status: AgentResultStatusFailed},
			{ID: "result-3", AgentID: "agent-3", Status: AgentResultStatusSuccess},
		}

		ctx := NewSynthesisContext("plan-123", results)
		successful := ctx.GetSuccessfulResults()

		assert.Len(t, successful, 2)
		assert.Equal(t, "result-1", successful[0].ID)
		assert.Equal(t, "result-3", successful[1].ID)
	})
}
