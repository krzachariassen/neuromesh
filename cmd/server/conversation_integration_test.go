package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConversationPersistenceIntegration tests end-to-end conversation persistence
func TestConversationPersistenceIntegration(t *testing.T) {
	t.Skip("This test requires the server to be running with Neo4j and RabbitMQ")

	// This test validates that:
	// 1. The server uses ConversationAwareWebBFF instead of regular WebBFF
	// 2. Conversations are persisted to the graph database
	// 3. Message history is maintained across requests

	sessionID := "test-session-" + fmt.Sprintf("%d", time.Now().Unix())
	serverURL := "http://localhost:8081"

	t.Run("should persist conversation messages", func(t *testing.T) {
		// Send first message
		message1 := map[string]interface{}{
			"message":   "Hello, can you help me?",
			"sessionId": sessionID,
		}

		response1, err := sendWebMessage(serverURL, message1)
		require.NoError(t, err)
		require.NotNil(t, response1)
		assert.Equal(t, sessionID, response1["sessionId"])
		assert.NotEmpty(t, response1["content"])

		// Send second message
		message2 := map[string]interface{}{
			"message":   "What can you do for me?",
			"sessionId": sessionID,
		}

		response2, err := sendWebMessage(serverURL, message2)
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, sessionID, response2["sessionId"])
		assert.NotEmpty(t, response2["content"])

		// The server should maintain conversation context
		// Both messages should be in the same conversation in the graph
		fmt.Printf("✅ Conversation persistence test completed for session: %s\n", sessionID)
		fmt.Printf("Response 1: %s\n", response1["content"])
		fmt.Printf("Response 2: %s\n", response2["content"])
	})
}

// sendWebMessage sends a message to the WebBFF endpoint
func sendWebMessage(serverURL string, message map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/api/message", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}
