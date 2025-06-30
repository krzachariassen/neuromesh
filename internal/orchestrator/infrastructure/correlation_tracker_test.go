package infrastructure

import (
	"context"
	"fmt"
	"testing"
	"time"

	"neuromesh/internal/messaging"
)

// RED Phase: Write failing tests that expose the design requirements
func TestCorrelationTracker_RegisterRequest_ShouldReturnChannel(t *testing.T) {
	// Arrange
	tracker := NewCorrelationTracker()
	correlationID := "test-correlation-123"
	userID := "user-456"
	timeout := 5 * time.Second

	// Act
	responseChan := tracker.RegisterRequest(correlationID, userID, timeout)

	// Assert
	if responseChan == nil {
		t.Fatal("RegisterRequest should return a non-nil channel")
	}

	// Channel should be ready to receive
	select {
	case <-responseChan:
		t.Fatal("Channel should not have data initially")
	default:
		// Expected: channel is empty initially
	}
}

func TestCorrelationTracker_RouteResponse_ShouldDeliverToWaitingRequest(t *testing.T) {
	// Arrange
	tracker := NewCorrelationTracker()
	correlationID := "test-correlation-123"
	userID := "user-456"
	timeout := 5 * time.Second

	// Register a request first
	responseChan := tracker.RegisterRequest(correlationID, userID, timeout)

	// Create a mock agent response
	agentResponse := &messaging.AgentToAIMessage{
		AgentID:       "test-agent",
		Content:       "Test response",
		MessageType:   messaging.MessageTypeResponse,
		CorrelationID: correlationID,
		Context:       map[string]interface{}{"status": "completed"},
		NeedsHelp:     false,
	}

	// Act
	routed := tracker.RouteResponse(agentResponse)

	// Assert
	if !routed {
		t.Fatal("RouteResponse should return true for known correlation ID")
	}

	// Should receive the response on the channel
	select {
	case receivedResponse := <-responseChan:
		if receivedResponse.CorrelationID != correlationID {
			t.Errorf("Expected correlation ID %s, got %s", correlationID, receivedResponse.CorrelationID)
		}
		if receivedResponse.Content != "Test response" {
			t.Errorf("Expected content 'Test response', got %s", receivedResponse.Content)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Should have received response immediately")
	}
}

func TestCorrelationTracker_RouteResponse_ShouldReturnFalseForUnknownCorrelationID(t *testing.T) {
	// Arrange
	tracker := NewCorrelationTracker()
	
	// Create a response for unknown correlation ID
	agentResponse := &messaging.AgentToAIMessage{
		AgentID:       "test-agent",
		Content:       "Test response",
		MessageType:   messaging.MessageTypeResponse,
		CorrelationID: "unknown-correlation",
		Context:       map[string]interface{}{"status": "completed"},
		NeedsHelp:     false,
	}

	// Act
	routed := tracker.RouteResponse(agentResponse)

	// Assert
	if routed {
		t.Fatal("RouteResponse should return false for unknown correlation ID")
	}
}

func TestCorrelationTracker_CleanupRequest_ShouldRemovePendingRequest(t *testing.T) {
	// Arrange
	tracker := NewCorrelationTracker()
	correlationID := "test-correlation-123"
	userID := "user-456"
	timeout := 5 * time.Second

	// Register a request
	responseChan := tracker.RegisterRequest(correlationID, userID, timeout)
	if responseChan == nil {
		t.Fatal("Failed to register request")
	}

	// Act
	tracker.CleanupRequest(correlationID)

	// Assert
	// Try to route a response - should fail now
	agentResponse := &messaging.AgentToAIMessage{
		AgentID:       "test-agent",
		Content:       "Test response",
		MessageType:   messaging.MessageTypeResponse,
		CorrelationID: correlationID,
		Context:       map[string]interface{}{"status": "completed"},
		NeedsHelp:     false,
	}

	routed := tracker.RouteResponse(agentResponse)
	if routed {
		t.Fatal("RouteResponse should return false after cleanup")
	}
}

func TestCorrelationTracker_ConcurrentAccess_ShouldBeThreadSafe(t *testing.T) {
	// Arrange
	tracker := NewCorrelationTracker()
	done := make(chan bool, 2)

	// Act: Concurrent registration and cleanup
	go func() {
		for i := 0; i < 100; i++ {
			correlationID := fmt.Sprintf("test-correlation-%d", i)
			responseChan := tracker.RegisterRequest(correlationID, "user", 5*time.Second)
			if responseChan == nil {
				t.Errorf("Failed to register request %d", i)
			}
			tracker.CleanupRequest(correlationID)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			agentResponse := &messaging.AgentToAIMessage{
				AgentID:       "test-agent",
				Content:       "Test response",
				MessageType:   messaging.MessageTypeResponse,
				CorrelationID: fmt.Sprintf("test-correlation-%d", i),
				Context:       map[string]interface{}{"status": "completed"},
				NeedsHelp:     false,
			}
			tracker.RouteResponse(agentResponse) // May succeed or fail, both are OK
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we reach here without deadlock, the test passes
}

func TestCorrelationTracker_AutoCleanup_ShouldRemoveExpiredRequests(t *testing.T) {
	// Arrange
	tracker := NewCorrelationTracker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the cleanup worker
	tracker.StartCleanupWorker(ctx)

	correlationID := "test-correlation-123"
	userID := "user-456"
	timeout := 50 * time.Millisecond // Very short timeout for testing

	// Act
	responseChan := tracker.RegisterRequest(correlationID, userID, timeout)
	if responseChan == nil {
		t.Fatal("Failed to register request")
	}

	// Wait for auto-cleanup (should happen after timeout)
	time.Sleep(100 * time.Millisecond)

	// Assert
	agentResponse := &messaging.AgentToAIMessage{
		AgentID:       "test-agent",
		Content:       "Test response",
		MessageType:   messaging.MessageTypeResponse,
		CorrelationID: correlationID,
		Context:       map[string]interface{}{"status": "completed"},
		NeedsHelp:     false,
	}

	routed := tracker.RouteResponse(agentResponse)
	if routed {
		t.Fatal("Request should have been auto-cleaned up after timeout")
	}
}
