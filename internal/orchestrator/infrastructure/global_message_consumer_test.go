package infrastructure

import (
	"context"
	"fmt"
	"testing"
	"time"

	"neuromesh/internal/messaging"
)

// RED Phase: Write failing tests for GlobalMessageConsumer

func TestGlobalMessageConsumer_StartConsumption_ShouldStartConsumingFromQueue(t *testing.T) {
	// Arrange
	mockBus := &MockMessageBus{
		messages: make(chan *messaging.Message, 10),
	}
	tracker := NewCorrelationTracker()
	consumer := NewGlobalMessageConsumer(mockBus, tracker)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Act
	err := consumer.StartConsumption(ctx, "orchestrator")

	// Assert
	if err != nil {
		t.Fatalf("StartConsumption should not return error: %v", err)
	}

	// Should have called Subscribe on the message bus
	if !mockBus.SubscribeCalled {
		t.Fatal("StartConsumption should call Subscribe on message bus")
	}
}

func TestGlobalMessageConsumer_RouteMessage_ShouldRouteToCorrelationTracker(t *testing.T) {
	// Arrange
	mockBus := &MockMessageBus{
		messages: make(chan *messaging.Message, 10),
	}
	tracker := NewCorrelationTracker()
	consumer := NewGlobalMessageConsumer(mockBus, tracker)

	correlationID := "test-correlation-123"
	userID := "user-456"

	// Register a request in the tracker
	responseChan := tracker.RegisterRequest(correlationID, userID, 5*time.Second)

	// Create an agent response message
	agentResponse := &messaging.Message{
		MessageType:   messaging.MessageTypeAgentToAI,
		Content:       "Test response from agent",
		FromID:        "test-agent",
		ToID:          "orchestrator",
		CorrelationID: correlationID,
		Metadata: map[string]interface{}{
			"userID": userID,
		},
	}

	// Act
	routed := consumer.RouteMessage(agentResponse)

	// Assert
	if !routed {
		t.Fatal("RouteMessage should return true for known correlation ID")
	}

	// Should receive the response on the tracker's channel
	select {
	case receivedResponse := <-responseChan:
		if receivedResponse.CorrelationID != correlationID {
			t.Errorf("Expected correlation ID %s, got %s", correlationID, receivedResponse.CorrelationID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Should have received response through correlation tracker")
	}
}

func TestGlobalMessageConsumer_RouteMessage_ShouldHandleUnknownCorrelationID(t *testing.T) {
	// Arrange
	mockBus := &MockMessageBus{
		messages: make(chan *messaging.Message, 10),
	}
	tracker := NewCorrelationTracker()
	consumer := NewGlobalMessageConsumer(mockBus, tracker)

	// Create an agent response message with unknown correlation ID
	agentResponse := &messaging.Message{
		MessageType:   messaging.MessageTypeAgentToAI,
		Content:       "Test response from agent",
		FromID:        "test-agent",
		ToID:          "orchestrator",
		CorrelationID: "unknown-correlation",
		Metadata: map[string]interface{}{
			"userID": "user-456",
		},
	}

	// Act
	routed := consumer.RouteMessage(agentResponse)

	// Assert
	if routed {
		t.Fatal("RouteMessage should return false for unknown correlation ID")
	}
}

func TestGlobalMessageConsumer_EndToEndMessageFlow_ShouldConsumeAndRoute(t *testing.T) {
	// Arrange
	mockBus := &MockMessageBus{
		messages: make(chan *messaging.Message, 10),
	}
	tracker := NewCorrelationTracker()
	consumer := NewGlobalMessageConsumer(mockBus, tracker)

	correlationID := "test-correlation-123"
	userID := "user-456"

	// Register a request in the tracker
	responseChan := tracker.RegisterRequest(correlationID, userID, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Act: Start consumption
	err := consumer.StartConsumption(ctx, "orchestrator")
	if err != nil {
		t.Fatalf("StartConsumption failed: %v", err)
	}

	// Send a message to the mock bus (simulates agent response)
	agentResponse := &messaging.Message{
		MessageType:   messaging.MessageTypeAgentToAI,
		Content:       "Test response from agent",
		FromID:        "test-agent",
		ToID:          "orchestrator",
		CorrelationID: correlationID,
		Metadata: map[string]interface{}{
			"userID": userID,
		},
	}

	// Simulate message arriving on the bus
	mockBus.messages <- agentResponse

	// Assert: Should receive the response through correlation tracker
	select {
	case receivedResponse := <-responseChan:
		if receivedResponse.CorrelationID != correlationID {
			t.Errorf("Expected correlation ID %s, got %s", correlationID, receivedResponse.CorrelationID)
		}
		if receivedResponse.Content != "Test response from agent" {
			t.Errorf("Expected content 'Test response from agent', got %s", receivedResponse.Content)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Should have received response through end-to-end flow")
	}
}

func TestGlobalMessageConsumer_ConcurrentMessageProcessing_ShouldHandleMultipleMessages(t *testing.T) {
	// Arrange
	mockBus := &MockMessageBus{
		messages: make(chan *messaging.Message, 100),
	}
	tracker := NewCorrelationTracker()
	consumer := NewGlobalMessageConsumer(mockBus, tracker)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start consumption
	err := consumer.StartConsumption(ctx, "orchestrator")
	if err != nil {
		t.Fatalf("StartConsumption failed: %v", err)
	}

	// Act: Send multiple messages concurrently
	numMessages := 10
	responseChans := make([]chan *messaging.AgentToAIMessage, numMessages)

	for i := 0; i < numMessages; i++ {
		correlationID := fmt.Sprintf("test-correlation-%d", i)
		userID := fmt.Sprintf("user-%d", i)

		// Register request
		responseChans[i] = tracker.RegisterRequest(correlationID, userID, 5*time.Second)

		// Send message
		agentResponse := &messaging.Message{
			MessageType:   messaging.MessageTypeAgentToAI,
			Content:       fmt.Sprintf("Response %d", i),
			FromID:        "test-agent",
			ToID:          "orchestrator",
			CorrelationID: correlationID,
			Metadata: map[string]interface{}{
				"userID": userID,
			},
		}

		mockBus.messages <- agentResponse
	}

	// Assert: All messages should be routed correctly
	for i := 0; i < numMessages; i++ {
		select {
		case receivedResponse := <-responseChans[i]:
			expectedCorrelationID := fmt.Sprintf("test-correlation-%d", i)
			if receivedResponse.CorrelationID != expectedCorrelationID {
				t.Errorf("Message %d: Expected correlation ID %s, got %s", i, expectedCorrelationID, receivedResponse.CorrelationID)
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("Message %d: Should have received response", i)
		}
	}
}

// MockMessageBus for testing GlobalMessageConsumer
type MockMessageBus struct {
	messages        chan *messaging.Message
	SubscribeCalled bool
}

func (m *MockMessageBus) SendToAgent(ctx context.Context, msg *messaging.AIToAgentMessage) error {
	return nil
}

func (m *MockMessageBus) SendToAI(ctx context.Context, msg *messaging.AgentToAIMessage) error {
	return nil
}

func (m *MockMessageBus) SendBetweenAgents(ctx context.Context, msg *messaging.AgentToAgentMessage) error {
	return nil
}

func (m *MockMessageBus) SendUserToAI(ctx context.Context, msg *messaging.UserToAIMessage) error {
	return nil
}

func (m *MockMessageBus) Subscribe(ctx context.Context, participantID string) (<-chan *messaging.Message, error) {
	m.SubscribeCalled = true
	return m.messages, nil
}

func (m *MockMessageBus) GetConversationHistory(ctx context.Context, correlationID string) ([]*messaging.Message, error) {
	return []*messaging.Message{}, nil
}

func (m *MockMessageBus) PrepareAgentQueue(ctx context.Context, agentID string) error {
	return nil
}
