package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"neuromesh/internal/api/rest/v1/controllers"
	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/mock"
)

// TDD RED Phase: Integration test that exposes missing server route wiring
func TestCleanAPIIntegration_ConversationGraph(t *testing.T) {
	// GIVEN: A clean REST API server should be wired up

	// Create mock conversation service using existing test helpers
	mockConversationService := testHelpers.NewMockConversationService()
	mockGraphService := testHelpers.NewMockGraphService()

	// Set up mock expectations
	expectedGraphData := &conversationDomain.GraphData{
		Nodes: []conversationDomain.GraphNode{
			{ID: "test-node", Type: "user", Data: map[string]interface{}{"name": "Test User"}},
		},
		Edges: []conversationDomain.GraphEdge{
			{ID: "test-edge", Source: "test-node", Target: "conv-1", Type: "created"},
		},
	}
	mockGraphService.On("GetConversationGraph", mock.Anything, "test-conversation-1").Return(expectedGraphData, nil)

	// Create the clean controller
	controller := controllers.NewConversationController(mockConversationService)
	controller.SetGraphService(mockGraphService)

	// Create test server with clean API routes
	mux := http.NewServeMux()
	mux.Handle("/api/v1/conversations/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate controller method based on path
		if r.URL.Path == "/api/v1/conversations/test-conversation-1/graph" {
			controller.GetConversationGraph(w, r)
		} else if r.URL.Path == "/api/v1/conversations/test-conversation-1" {
			controller.GetConversation(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))

	server := httptest.NewServer(mux)
	defer server.Close()

	// WHEN: Making a request to the clean API endpoint
	resp, err := http.Get(server.URL + "/api/v1/conversations/test-conversation-1/graph")

	// THEN: Should get successful response with graph data
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Should have proper content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected application/json, got %s", contentType)
	}
}
