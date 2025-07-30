package messaging

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TDD: Define proper messaging abstraction that hides infrastructure details

func TestEventDrivenMessageBus_PublishDomainEvent(t *testing.T) {
	t.Run("should publish domain events without exposing infrastructure", func(t *testing.T) {
		// Given a message bus that supports domain events
		logger := logging.NewNoOpLogger()
		messageBus := NewMemoryMessageBus(logger)

		// When we publish a domain event
		event := AgentCompletedEvent{
			PlanID:    "plan-123",
			StepID:    "step-456",
			AgentID:   "agent-789",
			Status:    "completed",
			Timestamp: time.Now().UTC(),
		}

		err := messageBus.PublishDomainEvent(context.Background(), "execution.agent.completed", event)
		require.NoError(t, err)

		// Then the event should be published without exposing RabbitMQ details
		// The messaging domain handles all infrastructure internally
	})
}

func TestEventDrivenMessageBus_SubscribeToDomainEvents(t *testing.T) {
	t.Run("should subscribe to domain events using clean interface", func(t *testing.T) {
		// Given a message bus with domain event support
		logger := logging.NewNoOpLogger()
		messageBus := NewMemoryMessageBus(logger)
		ctx := context.Background()

		// When we subscribe to domain events
		eventChan, err := messageBus.SubscribeToDomainEvents(ctx, "synthesis-coordinator", "execution.agent.completed")
		require.NoError(t, err)

		// And publish an event
		event := AgentCompletedEvent{
			PlanID:  "plan-123",
			StepID:  "step-456",
			AgentID: "agent-789",
			Status:  "completed",
		}

		err = messageBus.PublishDomainEvent(ctx, "execution.agent.completed", event)
		require.NoError(t, err)

		// Then we should receive the event
		select {
		case receivedEvent := <-eventChan:
			assert.Equal(t, "execution.agent.completed", receivedEvent.EventType)

			var agentEvent AgentCompletedEvent
			err = receivedEvent.UnmarshalEventData(&agentEvent)
			require.NoError(t, err)
			assert.Equal(t, event.PlanID, agentEvent.PlanID)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected to receive domain event")
		}
	})
}

func TestEventDrivenMessageBus_AbstractsInfrastructure(t *testing.T) {
	t.Run("should hide all infrastructure details from application layer", func(t *testing.T) {
		// Given different message bus implementations
		logger := logging.NewNoOpLogger()
		memoryBus := NewMemoryMessageBus(logger)

		// When using the domain event interface
		var eventBus DomainEventBus = memoryBus

		// Then all infrastructure is hidden behind clean interface
		ctx := context.Background()
		event := AgentCompletedEvent{PlanID: "test"}

		err := eventBus.PublishDomainEvent(ctx, "test.event", event)
		require.NoError(t, err)

		_, err = eventBus.SubscribeToDomainEvents(ctx, "test-consumer", "test.event")
		require.NoError(t, err)

		// No RabbitMQ, AMQP, or infrastructure details exposed to application layer
	})
}
