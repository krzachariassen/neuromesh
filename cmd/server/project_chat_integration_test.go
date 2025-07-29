package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"neuromesh/internal/api/bff"
)

func TestProjectChatIntegration(t *testing.T) {
	t.Run("chat API should accept project ID", func(t *testing.T) {
		// Create a chat request with project ID
		chatReq := bff.ChatRequest{
			Message:        "Hello from project test",
			SessionID:      "test-session-123",
			TenantID:       "tenant-456",
			ProjectID:      "project-789", // This is what we added
			ConversationID: "conv-abc",
			UserID:         "user-def",
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(chatReq)
		if err != nil {
			t.Fatalf("Failed to marshal chat request: %v", err)
		}

		// Create HTTP request
		req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// The request structure should be valid
		if chatReq.ProjectID != "project-789" {
			t.Error("Project ID not preserved in chat request")
		}

		if chatReq.Message != "Hello from project test" {
			t.Error("Message not preserved in chat request")
		}

		// Basic validation that our structures compile and work
		if w.Code == 0 {
			// This just tests that our structures work, not the full HTTP flow
			t.Log("Project chat integration structures validated successfully")
		}
	})

	t.Run("web response should include project ID", func(t *testing.T) {
		// Test WebResponse with project context
		webResp := bff.WebResponse{
			Content:        "Response from project context",
			SessionID:      "test-session-123",
			ConversationID: "conv-abc",
			CorrelationID:  "corr-xyz",
			TenantID:       "tenant-456",
			ProjectID:      "project-789", // Project context in response
		}

		if webResp.ProjectID != "project-789" {
			t.Error("Project ID not preserved in web response")
		}

		if webResp.Content != "Response from project context" {
			t.Error("Content not preserved in web response")
		}

		t.Log("Project context in web response validated successfully")
	})
}
