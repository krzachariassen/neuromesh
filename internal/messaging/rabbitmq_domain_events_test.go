package messaging

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/logging"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRabbitMQDomainEvents_PublishAndSubscribe(t *testing.T) {
	// Skip if no RabbitMQ URL provided
	rabbitMQURL := getTestRabbitMQURL()
	if rabbitMQURL == "" {
		t.Skip("No RabbitMQ URL provided for integration test")
	}

	logger := logging.NewStructuredLogger(logging.LevelInfo)
	config := RabbitMQConfig{
		URL:            rabbitMQURL,
		ReconnectDelay: 1 * time.Second,
		MaxReconnects:  3,
		Heartbeat:      10 * time.Second,
	}

	bus := NewRabbitMQMessageBus(config, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := bus.Connect(ctx)
	require.NoError(t, err)
	defer bus.Close()

	// Test domain event subscription
	subscriptionCtx, subscriptionCancel := context.WithCancel(ctx)
	defer subscriptionCancel()
	
	eventsChan, err := bus.SubscribeToDomainEvents(subscriptionCtx, "synthesis-coordinator", "execution.agent.completed")
	require.NoError(t, err)

	// Publish a domain event
	testEvent := AgentCompletedEvent{
		PlanID:  "plan-123",
		StepID:  "step-456", 
		AgentID: "test-agent-123",
		Status:  "completed",
	}

	err = bus.PublishDomainEvent(ctx, "execution.agent.completed", testEvent)
	require.NoError(t, err)

	// Wait for the event
	select {
	case receivedEvent := <-eventsChan:
		assert.Equal(t, "execution.agent.completed", receivedEvent.EventType)
		
		var decodedEvent AgentCompletedEvent
		err := receivedEvent.UnmarshalEventData(&decodedEvent)
		require.NoError(t, err)
		
		assert.Equal(t, testEvent.AgentID, decodedEvent.AgentID)
		assert.Equal(t, testEvent.PlanID, decodedEvent.PlanID)
		assert.Equal(t, testEvent.StepID, decodedEvent.StepID)
		assert.Equal(t, testEvent.Status, decodedEvent.Status)
		
	case <-time.After(5 * time.Second):
		t.Fatal("Did not receive domain event within timeout")
	}
}

func TestRabbitMQDomainEvents_MultipleSubscribers(t *testing.T) {
	rabbitMQURL := getTestRabbitMQURL()
	if rabbitMQURL == "" {
		t.Skip("No RabbitMQ URL provided for integration test")
	}

	logger := logging.NewStructuredLogger(logging.LevelInfo)
	config := RabbitMQConfig{
		URL:            rabbitMQURL,
		ReconnectDelay: 1 * time.Second,
		MaxReconnects:  3,
		Heartbeat:      10 * time.Second,
	}

	bus := NewRabbitMQMessageBus(config, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := bus.Connect(ctx)
	require.NoError(t, err)
	defer bus.Close()

	// Create two subscribers for the same event
	subscriptionCtx, subscriptionCancel := context.WithCancel(ctx)
	defer subscriptionCancel()
	
	events1, err := bus.SubscribeToDomainEvents(subscriptionCtx, "subscriber-1", "test.event")
	require.NoError(t, err)
	
	events2, err := bus.SubscribeToDomainEvents(subscriptionCtx, "subscriber-2", "test.event")
	require.NoError(t, err)

	// Publish event
	testData := map[string]string{"message": "hello world"}
	err = bus.PublishDomainEvent(ctx, "test.event", testData)
	require.NoError(t, err)

	// Both should receive the event
	receivedCount := 0
	for i := 0; i < 2; i++ {
		select {
		case event := <-events1:
			assert.Equal(t, "test.event", event.EventType)
			receivedCount++
		case event := <-events2:
			assert.Equal(t, "test.event", event.EventType)
			receivedCount++
		case <-time.After(3 * time.Second):
			break
		}
	}
	
	assert.Equal(t, 2, receivedCount, "Both subscribers should receive the event")
}

func getTestRabbitMQURL() string {
	// Try common test URLs
	testURLs := []string{
		"amqp://guest:guest@localhost:5672/",
		"amqp://localhost:5672/",
	}
	
	for _, url := range testURLs {
		config := RabbitMQConfig{
			URL:            url,
			ReconnectDelay: 1 * time.Second,
			MaxReconnects:  1,
		}
		bus := NewRabbitMQMessageBus(config, logging.NewStructuredLogger(logging.LevelError))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := bus.Connect(ctx)
		cancel()
		if err == nil {
			bus.Close()
			return url
		}
	}
	return ""
}
