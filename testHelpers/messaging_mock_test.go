package testHelpers

import (
	"context"
	"testing"

	"neuromesh/internal/messaging"
)

// Test CorrelationID validation in MockAIMessageBus
func TestMockAIMessageBus_CorrelationIDValidation(t *testing.T) {
	mock := NewMockAIMessageBus()
	ctx := context.Background()

	t.Run("SendToAgent should fail without CorrelationID", func(t *testing.T) {
		msg := &messaging.AIToAgentMessage{
			AgentID: "test-agent",
			Content: "test content",
			Intent:  "test-intent",
			// CorrelationID is missing
		}

		err := mock.SendToAgent(ctx, msg)
		if err == nil {
			t.Fatal("Expected error for missing CorrelationID")
		}
		if err.Error() != "correlation ID is required for all messages" {
			t.Errorf("Expected specific error message, got: %s", err.Error())
		}
	})

	t.Run("SendToAI should fail without CorrelationID", func(t *testing.T) {
		msg := &messaging.AgentToAIMessage{
			AgentID:     "test-agent",
			Content:     "test content",
			MessageType: messaging.MessageTypeResponse,
			// CorrelationID is missing
		}

		err := mock.SendToAI(ctx, msg)
		if err == nil {
			t.Fatal("Expected error for missing CorrelationID")
		}
		if err.Error() != "correlation ID is required for all messages" {
			t.Errorf("Expected specific error message, got: %s", err.Error())
		}
	})

	t.Run("SendBetweenAgents should fail without CorrelationID", func(t *testing.T) {
		msg := &messaging.AgentToAgentMessage{
			FromAgentID: "agent1",
			ToAgentID:   "agent2",
			Content:     "test content",
			// CorrelationID is missing
		}

		err := mock.SendBetweenAgents(ctx, msg)
		if err == nil {
			t.Fatal("Expected error for missing CorrelationID")
		}
		if err.Error() != "correlation ID is required for all messages" {
			t.Errorf("Expected specific error message, got: %s", err.Error())
		}
	})

	t.Run("SendUserToAI should fail without CorrelationID", func(t *testing.T) {
		msg := &messaging.UserToAIMessage{
			UserID:  "test-user",
			Content: "test content",
			// CorrelationID is missing
		}

		err := mock.SendUserToAI(ctx, msg)
		if err == nil {
			t.Fatal("Expected error for missing CorrelationID")
		}
		if err.Error() != "correlation ID is required for all messages" {
			t.Errorf("Expected specific error message, got: %s", err.Error())
		}
	})

	t.Run("All methods should work with valid CorrelationID", func(t *testing.T) {
		correlationID := "test-correlation-123"

		// Set up mock expectations
		mock.On("SendToAgent", ctx, &messaging.AIToAgentMessage{
			AgentID:       "test-agent",
			Content:       "test content",
			Intent:        "test-intent",
			CorrelationID: correlationID,
		}).Return(nil)

		mock.On("SendToAI", ctx, &messaging.AgentToAIMessage{
			AgentID:       "test-agent",
			Content:       "test content",
			MessageType:   messaging.MessageTypeResponse,
			CorrelationID: correlationID,
		}).Return(nil)

		mock.On("SendBetweenAgents", ctx, &messaging.AgentToAgentMessage{
			FromAgentID:   "agent1",
			ToAgentID:     "agent2",
			Content:       "test content",
			CorrelationID: correlationID,
		}).Return(nil)

		mock.On("SendUserToAI", ctx, &messaging.UserToAIMessage{
			UserID:        "test-user",
			Content:       "test content",
			CorrelationID: correlationID,
		}).Return(nil)

		// Test all methods with valid CorrelationID
		err := mock.SendToAgent(ctx, &messaging.AIToAgentMessage{
			AgentID:       "test-agent",
			Content:       "test content",
			Intent:        "test-intent",
			CorrelationID: correlationID,
		})
		if err != nil {
			t.Errorf("SendToAgent should succeed with CorrelationID: %v", err)
		}

		err = mock.SendToAI(ctx, &messaging.AgentToAIMessage{
			AgentID:       "test-agent",
			Content:       "test content",
			MessageType:   messaging.MessageTypeResponse,
			CorrelationID: correlationID,
		})
		if err != nil {
			t.Errorf("SendToAI should succeed with CorrelationID: %v", err)
		}

		err = mock.SendBetweenAgents(ctx, &messaging.AgentToAgentMessage{
			FromAgentID:   "agent1",
			ToAgentID:     "agent2",
			Content:       "test content",
			CorrelationID: correlationID,
		})
		if err != nil {
			t.Errorf("SendBetweenAgents should succeed with CorrelationID: %v", err)
		}

		err = mock.SendUserToAI(ctx, &messaging.UserToAIMessage{
			UserID:        "test-user",
			Content:       "test content",
			CorrelationID: correlationID,
		})
		if err != nil {
			t.Errorf("SendUserToAI should succeed with CorrelationID: %v", err)
		}

		// Verify all expectations were met
		mock.AssertExpectations(t)
	})
}
