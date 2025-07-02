package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
)

// TestStatelessAIConversationEngine_TDD tests the new stateless design
func TestStatelessAIConversationEngine_TDD(t *testing.T) {
	t.Run("RED: should support concurrent conversations with different correlations", func(t *testing.T) {
		// ARRANGE - Create stateless engine with CorrelationTracker
		aiProvider := setupRealAIProvider(t)
		mockBus := &mockMessageBus{
			sentMessages:    make([]interface{}, 0),
			responseChannel: make(chan *messaging.Message, 10),
		}
		correlationTracker := infrastructure.NewCorrelationTracker()

		// Create NEW stateless engine
		engine := NewStatelessAIConversationEngine(aiProvider, mockBus, correlationTracker)

		// Create two concurrent conversation contexts
		ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel1()
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		agentContext := `Available agents:
- text-processor (ID: text-processor, Status: online)
  Capabilities: word-count, text-analysis`

		// Simulate agent responses for both conversations
		go func() {
			time.Sleep(500 * time.Millisecond)
			
			// Send responses for both correlation IDs
			for i := 0; i < 2; i++ {
				if len(mockBus.sentMessages) > i {
					if agentMsg, ok := mockBus.sentMessages[i].(*messaging.AIToAgentMessage); ok {
						response := &messaging.Message{
							MessageType:   messaging.MessageTypeAgentToAI,
							Content:       `Test response for correlation ` + agentMsg.CorrelationID,
							FromID:        "text-processor",
							ToID:          "orchestrator",
							CorrelationID: agentMsg.CorrelationID,
							Timestamp:     time.Now(),
						}
						
						select {
						case mockBus.responseChannel <- response:
							t.Logf("✅ Sent response for correlation: %s", agentMsg.CorrelationID)
						case <-time.After(1 * time.Second):
							t.Logf("⚠️ Failed to send response")
						}
					}
				}
			}
		}()

		// ACT - Process two conversations concurrently  
		var response1, response2 string
		var err1, err2 error

		done := make(chan bool, 2)

		go func() {
			response1, err1 = engine.ProcessWithAgents(ctx1, "First conversation", "user1", agentContext)
			done <- true
		}()

		go func() {
			response2, err2 = engine.ProcessWithAgents(ctx2, "Second conversation", "user2", agentContext)
			done <- true
		}()

		// Wait for both conversations to complete
		<-done
		<-done

		// ASSERT - Both conversations should succeed independently
		require.NoError(t, err1, "First conversation should succeed")
		require.NoError(t, err2, "Second conversation should succeed")
		assert.NotEmpty(t, response1, "First response should not be empty")
		assert.NotEmpty(t, response2, "Second response should not be empty")
		assert.NotEqual(t, response1, response2, "Responses should be different")

		t.Logf("✅ Concurrent conversations completed: %s | %s", response1, response2)
	})

	t.Run("RED: should handle correlation-based message routing", func(t *testing.T) {
		// ARRANGE
		aiProvider := setupRealAIProvider(t)
		mockBus := &mockMessageBus{
			sentMessages:    make([]interface{}, 0),
			responseChannel: make(chan *messaging.Message, 10),
		}
		correlationTracker := infrastructure.NewCorrelationTracker()
		engine := NewStatelessAIConversationEngine(aiProvider, mockBus, correlationTracker)

		agentContext := `Available agents:
- text-processor (ID: text-processor, Status: online)
  Capabilities: word-count, text-analysis`

		// ACT - Start conversation and get correlation ID
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start processing in background to capture correlation ID
		var correlationID string
		var response string
		var err error

		done := make(chan bool)
		go func() {
			defer func() { done <- true }()
			// Use input that clearly requires text-processor agent assistance
			response, err = engine.ProcessWithAgents(ctx, "Count the words in this text: Hello world testing", "user123", agentContext)
		}()

		// Wait for agent message to be sent and capture correlation ID
		time.Sleep(500 * time.Millisecond)
		require.Greater(t, len(mockBus.sentMessages), 0, "Should have sent message to agent")

		if agentMsg, ok := mockBus.sentMessages[0].(*messaging.AIToAgentMessage); ok {
			correlationID = agentMsg.CorrelationID
			assert.NotEmpty(t, correlationID, "CorrelationID should be set")

			// Send response with correct correlation ID
			agentResponse := &messaging.Message{
				MessageType:   messaging.MessageTypeAgentToAI,
				Content:       "Correlation test response",
				FromID:        "text-processor", 
				ToID:          "orchestrator",
				CorrelationID: correlationID,
				Timestamp:     time.Now(),
			}

			select {
			case mockBus.responseChannel <- agentResponse:
				t.Logf("✅ Sent correlated response: %s", correlationID)
			case <-time.After(1 * time.Second):
				t.Fatal("Failed to send correlated response")
			}
		} else {
			t.Fatal("Expected AIToAgentMessage")
		}

		// Wait for completion
		<-done

		// ASSERT
		require.NoError(t, err, "Should handle correlated response")
		assert.Contains(t, response, "Correlation test response", "Should include agent response")
	})
}
