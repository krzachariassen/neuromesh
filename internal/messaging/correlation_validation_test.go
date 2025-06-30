package messaging

import (
	"context"
	"neuromesh/internal/logging"
	"testing"
)

func TestMessageBusCorrelationIDValidation(t *testing.T) {
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	bus := NewMemoryMessageBus(logger)
	ctx := context.Background()

	t.Run("SendMessage fails without CorrelationID", func(t *testing.T) {
		message := &Message{
			ID:          "test-msg-123",
			FromID:      "agent-1",
			ToID:        "orchestrator",
			Content:     "test message",
			MessageType: MessageTypeRequest,
			// CorrelationID: "", // Intentionally missing
		}

		err := bus.SendMessage(ctx, message)
		if err == nil {
			t.Error("Expected error when CorrelationID is missing, got nil")
		}
		if err.Error() != "correlation ID is required for all messages" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("SendMessage succeeds with CorrelationID", func(t *testing.T) {
		// Set up subscriber for orchestrator
		_, err := bus.Subscribe(ctx, "orchestrator")
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		defer bus.Unsubscribe(ctx, "orchestrator")

		message := &Message{
			ID:            "test-msg-123",
			FromID:        "agent-1",
			ToID:          "orchestrator",
			Content:       "test message",
			MessageType:   MessageTypeRequest,
			CorrelationID: "corr-123", // Present
		}

		err = bus.SendMessage(ctx, message)
		if err != nil {
			t.Errorf("Expected no error with CorrelationID, got: %v", err)
		}
	})

	t.Run("PublishMessage fails without CorrelationID", func(t *testing.T) {
		message := &Message{
			ID:          "test-msg-123",
			FromID:      "orchestrator",
			Content:     "broadcast message",
			MessageType: MessageTypeNotification,
			// CorrelationID: "", // Intentionally missing
		}

		err := bus.PublishMessage(ctx, message, []string{"agent-1", "agent-2"})
		if err == nil {
			t.Error("Expected error when CorrelationID is missing in PublishMessage, got nil")
		}
	})
}

func TestAIMessageBusCorrelationIDValidation(t *testing.T) {
	mockBus := &MockMessageBus{}
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	bus := NewAIMessageBus(mockBus, nil, logger)
	ctx := context.Background()

	t.Run("SendToAgent fails without CorrelationID", func(t *testing.T) {
		msg := &AIToAgentMessage{
			AgentID: "agent-1",
			Content: "test content",
			Intent:  "test intent",
			// CorrelationID: "", // Intentionally missing
		}

		err := bus.SendToAgent(ctx, msg)
		if err == nil {
			t.Error("Expected error when CorrelationID is missing, got nil")
		}
	})

	t.Run("SendToAI fails without CorrelationID", func(t *testing.T) {
		msg := &AgentToAIMessage{
			AgentID:     "agent-1",
			Content:     "test content",
			MessageType: MessageTypeResponse,
			// CorrelationID: "", // Intentionally missing
		}

		err := bus.SendToAI(ctx, msg)
		if err == nil {
			t.Error("Expected error when CorrelationID is missing, got nil")
		}
	})
}
