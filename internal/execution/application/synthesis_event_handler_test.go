package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSynthesisEventHandler_BasicFunctionality tests the core event-driven synthesis functionality
func TestSynthesisEventHandler_BasicFunctionality(t *testing.T) {
	t.Run("can create synthesis event handler", func(t *testing.T) {
		// Test that we can create the handler
		handler := NewSynthesisEventHandler(nil, nil, nil, nil)
		assert.NotNil(t, handler)
	})

	t.Run("HandleAgentCompleted with nil coordinator returns error", func(t *testing.T) {
		handler := NewSynthesisEventHandler(nil, nil, nil, nil)

		event := &AgentCompletedEvent{
			PlanID:  "plan-1",
			StepID:  "step-1",
			AgentID: "agent-1",
		}

		// This should fail because coordinator is nil
		err := handler.HandleAgentCompleted(context.Background(), event)
		assert.Error(t, err)
	})

	t.Run("PublishAgentCompletedEvent with nil messageBus returns error", func(t *testing.T) {
		// This should fail because messageBus is nil
		err := PublishAgentCompletedEvent(context.Background(), nil, "plan-1", "step-1", "agent-1")
		assert.Error(t, err)
	})

	t.Run("can create AgentCompletedEvent", func(t *testing.T) {
		event := &AgentCompletedEvent{
			PlanID:  "plan-1",
			StepID:  "step-1",
			AgentID: "agent-1",
		}

		assert.Equal(t, "plan-1", event.PlanID)
		assert.Equal(t, "step-1", event.StepID)
		assert.Equal(t, "agent-1", event.AgentID)
	})
}
