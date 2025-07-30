package application

import (
	"context"
	"testing"

	"neuromesh/internal/messaging"

	"github.com/stretchr/testify/assert"
)

// TestSynthesisEventHandler_BasicFunctionality tests the core event-driven synthesis functionality
func TestSynthesisEventHandler_BasicFunctionality(t *testing.T) {
	t.Run("can create synthesis event handler", func(t *testing.T) {
		// Test that we can create the handler
		handler := NewSynthesisEventHandler(nil, nil, nil)
		assert.NotNil(t, handler)
	})

	t.Run("HandleAgentCompleted with nil synthesizer returns error", func(t *testing.T) {
		handler := NewSynthesisEventHandler(nil, nil, nil)

		event := &messaging.AgentCompletedEvent{
			PlanID:  "plan-1",
			StepID:  "step-1",
			AgentID: "agent-1",
		}

		// This should fail because synthesizer is nil
		err := handler.HandleAgentCompleted(context.Background(), event)
		assert.Error(t, err)
	})

	t.Run("PublishAgentCompletedEvent with nil messageBus returns error", func(t *testing.T) {
		// This should fail because messageBus is nil
		err := PublishAgentCompletedEvent(context.Background(), nil, "plan-1", "step-1", "agent-1")
		assert.Error(t, err)
	})

	t.Run("can create AgentCompletedEvent", func(t *testing.T) {
		event := &messaging.AgentCompletedEvent{
			PlanID:  "plan-1",
			StepID:  "step-1",
			AgentID: "agent-1",
		}

		assert.Equal(t, "plan-1", event.PlanID)
		assert.Equal(t, "step-1", event.StepID)
		assert.Equal(t, "agent-1", event.AgentID)
	})

	// RED: Test that execution plan status is updated to COMPLETED after synthesis
	t.Run("should_update_execution_plan_status_to_completed_after_synthesis", func(t *testing.T) {
		// This test proves the bug: execution plan status isn't updated after synthesis completes
		// This test should FAIL initially, proving the issue exists
		t.Skip("TODO: Implement test to verify execution plan status update after synthesis")
	})
}
