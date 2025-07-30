package messaging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TDD: RED Phase - Define what we need for proper event-driven synthesis coordination

func TestSynthesisEventCoordination_PublishAgentCompletedEvent(t *testing.T) {
	t.Run("should publish agent completed event to execution exchange", func(t *testing.T) {
		// Given a memory-based event router for testing
		router := NewMemoryEventRouter()
		ctx := context.Background()

		// Setup execution exchange
		err := router.SetupEventExchange(ctx, "execution")
		require.NoError(t, err)

		// Subscribe to agent completion events
		eventChan, err := router.SubscribeToEvents(ctx, "execution", "synthesis-coordinator", "agent.completed")
		require.NoError(t, err)

		// When an agent completion event is published
		event := AgentCompletedEvent{
			PlanID:    "plan-123",
			StepID:    "step-456",
			AgentID:   "agent-789",
			Status:    "completed",
			Timestamp: time.Now().UTC(),
		}

		err = router.PublishEvent(ctx, "execution", "agent.completed", event)
		require.NoError(t, err)

		// Then the event should be received by synthesis coordinator
		select {
		case receivedMsg := <-eventChan:
			assert.Equal(t, "execution", receivedMsg.Exchange)
			assert.Equal(t, "agent.completed", receivedMsg.RoutingKey)
			assert.Equal(t, "agent.completed", receivedMsg.EventType)

			var receivedEvent AgentCompletedEvent
			err = json.Unmarshal(receivedMsg.EventData, &receivedEvent)
			require.NoError(t, err)
			assert.Equal(t, event.PlanID, receivedEvent.PlanID)
			assert.Equal(t, event.StepID, receivedEvent.StepID)
			assert.Equal(t, event.AgentID, receivedEvent.AgentID)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected to receive agent completed event")
		}
	})
}

func TestSynthesisEventCoordination_MultipleSubscribers(t *testing.T) {
	t.Run("should allow multiple services to subscribe to execution events", func(t *testing.T) {
		// Given a memory-based event router
		router := NewMemoryEventRouter()
		ctx := context.Background()

		// Setup execution exchange
		err := router.SetupEventExchange(ctx, "execution")
		require.NoError(t, err)

		// Multiple subscribers to execution events
		synthesisChan, err := router.SubscribeToEvents(ctx, "execution", "synthesis-coordinator", "agent.*")
		require.NoError(t, err)

		auditChan, err := router.SubscribeToEvents(ctx, "execution", "audit-logger", "agent.*")
		require.NoError(t, err)

		// When an agent completion event is published
		event := AgentCompletedEvent{
			PlanID:  "plan-123",
			StepID:  "step-456",
			AgentID: "agent-789",
			Status:  "completed",
		}

		err = router.PublishEvent(ctx, "execution", "agent.completed", event)
		require.NoError(t, err)

		// Then both subscribers should receive the event
		select {
		case msg := <-synthesisChan:
			assert.Equal(t, "agent.completed", msg.EventType)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Synthesis coordinator should receive event")
		}

		select {
		case msg := <-auditChan:
			assert.Equal(t, "agent.completed", msg.EventType)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Audit logger should receive event")
		}
	})
}

func TestSynthesisEventCoordination_EventFiltering(t *testing.T) {
	t.Run("should filter events by routing key patterns", func(t *testing.T) {
		// Given a memory-based event router
		router := NewMemoryEventRouter()
		ctx := context.Background()

		// Setup execution exchange
		err := router.SetupEventExchange(ctx, "execution")
		require.NoError(t, err)

		// Subscribe only to completed events, not started events
		completedChan, err := router.SubscribeToEvents(ctx, "execution", "synthesis-coordinator", "agent.completed")
		require.NoError(t, err)

		// Publish agent started event (should not be received)
		startedEvent := map[string]interface{}{
			"plan_id":  "plan-123",
			"step_id":  "step-456",
			"agent_id": "agent-789",
		}
		err = router.PublishEvent(ctx, "execution", "agent.started", startedEvent)
		require.NoError(t, err)

		// Publish agent completed event (should be received)
		completedEvent := AgentCompletedEvent{
			PlanID:  "plan-123",
			StepID:  "step-456",
			AgentID: "agent-789",
		}
		err = router.PublishEvent(ctx, "execution", "agent.completed", completedEvent)
		require.NoError(t, err)

		// Then only the completed event should be received
		select {
		case msg := <-completedChan:
			assert.Equal(t, "agent.completed", msg.EventType)
			var receivedEvent AgentCompletedEvent
			err = json.Unmarshal(msg.EventData, &receivedEvent)
			require.NoError(t, err)
			assert.Equal(t, completedEvent.PlanID, receivedEvent.PlanID)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Should receive completed event")
		}

		// Verify no additional messages (started event should be filtered out)
		select {
		case <-completedChan:
			t.Fatal("Should not receive additional events")
		case <-time.After(50 * time.Millisecond):
			// Expected - no additional events
		}
	})
}
