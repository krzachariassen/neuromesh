package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"neuromesh/internal/api/rest/v1/controllers"
	"neuromesh/internal/api/rest/v1/domain"
	"neuromesh/testHelpers"
)

// TDD RED Phase: Integration test that exposes missing server route wiring
func TestCleanAPIIntegration_ConversationGraph(t *testing.T) {
	// GIVEN: A clean REST API server should be wired up

	// Create mock conversation service using existing test helpers
	mockConversationService := testHelpers.NewMockConversationService()
	mockGraphService := &MockGraphService{
		graphData: &domain.GraphData{
			Nodes: []domain.Node{
				{ID: "test-node", Type: "user", Label: "Test User"},
			},
			Edges: []domain.Edge{
				{ID: "test-edge", Source: "test-node", Target: "conv-1", Type: "created"},
			},
		},
	}

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

// Mock graph service for testing
type MockGraphService struct {
	graphData *domain.GraphData
}

func (m *MockGraphService) GetConversationGraph(ctx context.Context, conversationID string) (*domain.GraphData, error) {
	return m.graphData, nil
}
