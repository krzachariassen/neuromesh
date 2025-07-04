package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
	"neuromesh/testHelpers"
)

// TestAIConversationEngine_TDD tests the new stateless design
func TestAIConversationEngine_TDD(t *testing.T) {
	t.Run("RED: should support concurrent conversations with different correlations", func(t *testing.T) {
		// ARRANGE - Create stateless engine with CorrelationTracker
		aiProvider := testHelpers.SetupRealAIProvider(t)
		mockBus := &mockMessageBus{
			sentMessages:    make([]interface{}, 0),
			responseChannel: make(chan *messaging.Message, 10),
		}
		correlationTracker := infrastructure.NewCorrelationTracker()

		// Create NEW stateless engine
		engine := NewAIConversationEngine(aiProvider, mockBus, correlationTracker)

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
				sentMessages := mockBus.GetSentMessages()
				if len(sentMessages) > i {
					if agentMsg, ok := sentMessages[i].(*messaging.AIToAgentMessage); ok {
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
		aiProvider := testHelpers.SetupRealAIProvider(t)
		mockBus := &mockMessageBus{
			sentMessages:    make([]interface{}, 0),
			responseChannel: make(chan *messaging.Message, 10),
		}
		correlationTracker := infrastructure.NewCorrelationTracker()
		engine := NewAIConversationEngine(aiProvider, mockBus, correlationTracker)

		agentContext := `Available agents:
- text-processor (ID: text-processor, Status: online)
  Capabilities: word-count, text-analysis`

		// ACT - Start conversation and get correlation ID
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
		time.Sleep(2 * time.Second) // Increased from 500ms to 2s to allow AI call to complete
		require.Greater(t, mockBus.GetSentMessageCount(), 0, "Should have sent message to agent")

		sentMessages := mockBus.GetSentMessages()
		if agentMsg, ok := sentMessages[0].(*messaging.AIToAgentMessage); ok {
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
		assert.NotEmpty(t, response, "Should have a response")
		// The AI should process the agent response and provide a meaningful answer
		// (not just echo the mock response)
		assert.Contains(t, response, "3", "Should include word count result")
		t.Logf("✅ Final AI response: %s", response)
	})

	t.Run("SCALE: should handle many concurrent conversations with proper correlation isolation", func(t *testing.T) {
		// ARRANGE - Setup for scale testing
		aiProvider := testHelpers.SetupRealAIProvider(t)

		// Larger channel buffer for scale testing
		mockBus := &mockMessageBus{
			sentMessages:    make([]interface{}, 0),
			responseChannel: make(chan *messaging.Message, 200), // Larger buffer for concurrent load
		}
		correlationTracker := infrastructure.NewCorrelationTracker()
		engine := NewAIConversationEngine(aiProvider, mockBus, correlationTracker)

		agentContext := `Available agents:
- text-processor (ID: text-processor, Status: online)
  Capabilities: word-count, text-analysis`

		// Scale parameters
		numConcurrentUsers := 10 // Reduced for more reliable testing
		requestsPerUser := 2     // Each user makes 2 requests
		totalRequests := numConcurrentUsers * requestsPerUser

		// Track correlation IDs as they're created (thread-safe)
		var correlationMutex sync.Mutex
		correlationTrackingMap := make(map[string]bool)

		// Simulate agent responses for all requests
		go func() {
			respondedCount := 0
			for respondedCount < totalRequests {
				time.Sleep(100 * time.Millisecond) // Check every 100ms

				// Respond to any new messages
				sentMessages := mockBus.GetSentMessages()
				currentMessageCount := len(sentMessages)
				for i := respondedCount; i < currentMessageCount && i < totalRequests; i++ {
					if agentMsg, ok := sentMessages[i].(*messaging.AIToAgentMessage); ok {
						// Track this correlation ID
						correlationMutex.Lock()
						correlationTrackingMap[agentMsg.CorrelationID] = true
						correlationMutex.Unlock()

						response := &messaging.Message{
							MessageType:   messaging.MessageTypeAgentToAI,
							Content:       fmt.Sprintf("Scale test response %d for correlation %s", i+1, agentMsg.CorrelationID),
							FromID:        "text-processor",
							ToID:          "orchestrator",
							CorrelationID: agentMsg.CorrelationID,
							Timestamp:     time.Now(),
						}

						select {
						case mockBus.responseChannel <- response:
							respondedCount++
						case <-time.After(100 * time.Millisecond):
							// Continue trying
						}
					}
				}
			}
		}()

		// ACT - Launch many concurrent conversations
		var wg sync.WaitGroup
		results := make(chan string, totalRequests)
		errors := make(chan error, totalRequests)

		startTime := time.Now()

		for userID := 1; userID <= numConcurrentUsers; userID++ {
			for reqID := 1; reqID <= requestsPerUser; reqID++ {
				wg.Add(1)

				go func(uID, rID int) {
					defer wg.Done()

					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()

					userIDStr := fmt.Sprintf("user%d", uID)
					requestText := fmt.Sprintf("Count words in text %d from user %d: Hello world test", rID, uID)

					response, err := engine.ProcessWithAgents(ctx, requestText, userIDStr, agentContext)

					if err != nil {
						errors <- fmt.Errorf("user%d-req%d: %w", uID, rID, err)
					} else {
						results <- fmt.Sprintf("user%d-req%d: %s", uID, rID, response)
					}
				}(userID, reqID)
			}
		}

		// Wait for all conversations to complete
		wg.Wait()
		close(results)
		close(errors)

		duration := time.Since(startTime)

		// ASSERT - Validate scale test results
		var successCount int
		var errorCount int
		responses := make([]string, 0)

		// Collect results
		for result := range results {
			responses = append(responses, result)
			successCount++
		}

		// Collect errors
		for err := range errors {
			t.Logf("❌ Error: %v", err)
			errorCount++
		}

		// Get the final correlation tracking map
		correlationMutex.Lock()
		finalCorrelationIDs := make(map[string]bool)
		for k, v := range correlationTrackingMap {
			finalCorrelationIDs[k] = v
		}
		correlationMutex.Unlock()

		// Validate results
		assert.Equal(t, totalRequests, successCount, "All requests should succeed")
		assert.Equal(t, 0, errorCount, "No errors should occur")
		assert.Len(t, responses, totalRequests, "Should have response for each request")

		// Validate correlation ID uniqueness - this is the key test
		assert.Len(t, finalCorrelationIDs, totalRequests, "All correlation IDs should be unique")

		// Validate no correlation ID format issues
		for correlationID := range finalCorrelationIDs {
			assert.True(t, strings.HasPrefix(correlationID, "conv-user"), "Correlation ID should have correct format: %s", correlationID)
			assert.Contains(t, correlationID, "-", "Correlation ID should contain separator: %s", correlationID)
		}

		// Performance validation
		avgTimePerRequest := duration / time.Duration(totalRequests)
		assert.Less(t, avgTimePerRequest, 10*time.Second, "Average response time should be reasonable")

		// Log scale test results
		t.Logf("✅ SCALE TEST RESULTS:")
		t.Logf("📊 Concurrent Users: %d", numConcurrentUsers)
		t.Logf("📊 Requests per User: %d", requestsPerUser)
		t.Logf("📊 Total Requests: %d", totalRequests)
		t.Logf("📊 Successful Responses: %d", successCount)
		t.Logf("📊 Errors: %d", errorCount)
		t.Logf("📊 Unique Correlation IDs: %d", len(finalCorrelationIDs))
		t.Logf("📊 Total Duration: %v", duration)
		t.Logf("📊 Average Time per Request: %v", avgTimePerRequest)
		t.Logf("📊 Requests per Second: %.2f", float64(totalRequests)/duration.Seconds())

		// Log some correlation IDs for verification
		count := 0
		for correlationID := range finalCorrelationIDs {
			if count < 3 {
				t.Logf("📊 Sample Correlation ID: %s", correlationID)
				count++
			}
		}

		// Validate some sample responses for content correctness
		sampleCount := 0
		for _, response := range responses {
			if sampleCount < 3 { // Check first 3 responses
				assert.NotEmpty(t, response, "Response should not be empty")
				assert.Contains(t, response, "user", "Response should contain user context")
				sampleCount++
			}
		}
	})
}

// mockMessageBus implements AIMessageBus for testing
type mockMessageBus struct {
	mu              sync.Mutex
	sentMessages    []interface{}
	responseChannel chan *messaging.Message
}

func (m *mockMessageBus) SendToAgent(ctx context.Context, msg *messaging.AIToAgentMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func (m *mockMessageBus) SendToAI(ctx context.Context, msg *messaging.AgentToAIMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func (m *mockMessageBus) SendBetweenAgents(ctx context.Context, msg *messaging.AgentToAgentMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func (m *mockMessageBus) SendUserToAI(ctx context.Context, msg *messaging.UserToAIMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func (m *mockMessageBus) Subscribe(ctx context.Context, participantID string) (<-chan *messaging.Message, error) {
	// If subscribing as orchestrator or ai-orchestrator and we have a response channel, return it
	if (participantID == "orchestrator" || participantID == "ai-orchestrator") && m.responseChannel != nil {
		return m.responseChannel, nil
	}

	// Otherwise return a closed channel
	ch := make(chan *messaging.Message)
	close(ch)
	return ch, nil
}

func (m *mockMessageBus) GetConversationHistory(ctx context.Context, correlationID string) ([]*messaging.Message, error) {
	return []*messaging.Message{}, nil
}

func (m *mockMessageBus) PrepareAgentQueue(ctx context.Context, agentID string) error {
	// Mock implementation - just return nil
	return nil
}

// GetSentMessages returns a copy of sent messages in a thread-safe way
func (m *mockMessageBus) GetSentMessages() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]interface{}, len(m.sentMessages))
	copy(result, m.sentMessages)
	return result
}

// GetSentMessageCount returns the count of sent messages in a thread-safe way
func (m *mockMessageBus) GetSentMessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sentMessages)
}
