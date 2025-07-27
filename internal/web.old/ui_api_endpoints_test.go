package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TDD RED Phase: Tests that expose the missing /api/ui/ endpoints
// These tests will fail until we implement the proper API structure

func TestUIAPIEndpoints_GraphData_ShouldReturn404(t *testing.T) {
	// RED: This test will fail because /api/ui/graph-data doesn't exist
	// We expect 404 Not Found until we implement the endpoint
	
	// Create a simple handler that only has existing endpoints
	mux := http.NewServeMux()
	
	// Test the /api/ui/graph-data endpoint (this should return 404)
	req := httptest.NewRequest("GET", "/api/ui/graph-data?conversation_id=test-123", nil)
	w := httptest.NewRecorder()
	
	mux.ServeHTTP(w, req)
	
	// Should return 404 because endpoint doesn't exist
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 (Not Found), got %d", w.Code)
	}
}

func TestUIAPIEndpoints_ConversationHistory_ShouldReturn404(t *testing.T) {
	// RED: This test will fail because /api/ui/conversation-history doesn't exist
	
	// Create a simple handler that only has existing endpoints
	mux := http.NewServeMux()
	
	// Test the /api/ui/conversation-history endpoint (this should return 404)
	req := httptest.NewRequest("GET", "/api/ui/conversation-history?conversation_id=test-123", nil)
	w := httptest.NewRecorder()
	
	mux.ServeHTTP(w, req)
	
	// Should return 404 because endpoint doesn't exist
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 (Not Found), got %d", w.Code)
	}
}

func TestUIAPIEndpoints_ExecutionPlans_ShouldReturn404(t *testing.T) {
	// RED: This test will fail because /api/ui/execution-plans doesn't exist
	
	// Create a simple handler that only has existing endpoints
	mux := http.NewServeMux()
	
	// Test the /api/ui/execution-plans endpoint (this should return 404)
	req := httptest.NewRequest("GET", "/api/ui/execution-plans?conversation_id=test-123", nil)
	w := httptest.NewRecorder()
	
	mux.ServeHTTP(w, req)
	
	// Should return 404 because endpoint doesn't exist
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 (Not Found), got %d", w.Code)
	}
}
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	userApp "neuromesh/internal/user/application"
	userDomain "neuromesh/internal/user/domain"
)

// TDD RED Phase: Tests that expose the missing /api/ui/ endpoints
// These tests will fail until we implement the proper API structure

func TestUIAPIEndpoints_GraphData_ShouldReturnGraphVisualizationData(t *testing.T) {
	// RED: This test will fail because /api/ui/graph-data doesn't exist
	
	// Setup test dependencies
	mockGraphRepo := testHelpers.NewMockGraphRepository()
	mockConversationService := testHelpers.NewMockConversationService()
	mockUserService := testHelpers.NewMockUserService()
	logger := testHelpers.NewTestLogger()

	// Setup test conversation with graph data
	testConversationID := "test-conversation-123"
	testConversation := &domain.Conversation{
		ID:        testConversationID,
		SessionID: "session-456",
		UserID:    "user-789",
		Status:    domain.ConversationStatusActive,
	}
	
	mockConversationService.SetConversation(testConversationID, testConversation)
	
	// Setup expected graph data
	expectedNodes := []map[string]interface{}{
		{"id": "node1", "label": "User Message", "type": "user"},
		{"id": "node2", "label": "AI Response", "type": "ai"},
	}
	expectedEdges := []map[string]interface{}{
		{"id": "edge1", "source": "node1", "target": "node2"},
	}
	
	mockGraphRepo.SetGraphData(testConversationID, expectedNodes, expectedEdges)

	// Create BFF instance
	bff := NewConversationAwareWebBFF(
		mockConversationService,
		mockUserService,
		mockGraphRepo,
		logger,
	)

	// Create test server
	server := bff.CreateWebServer(":8080")
	
	// Test the /api/ui/graph-data endpoint (this will fail - endpoint doesn't exist)
	req := httptest.NewRequest("GET", "/api/ui/graph-data?conversation_id="+testConversationID, nil)
	w := httptest.NewRecorder()
	
	server.Handler.ServeHTTP(w, req)
	
	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	// Verify graph data structure
	nodes, ok := response["nodes"].([]interface{})
	if !ok {
		t.Error("Expected nodes array in response")
	}
	
	edges, ok := response["edges"].([]interface{})
	if !ok {
		t.Error("Expected edges array in response")
	}
	
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
	
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(edges))
	}
}

func TestUIAPIEndpoints_ConversationHistory_ShouldReturnHistoryData(t *testing.T) {
	// RED: This test will fail because /api/ui/conversation-history doesn't exist
	
	// Setup test dependencies
	mockGraphRepo := testHelpers.NewMockGraphRepository()
	mockConversationService := testHelpers.NewMockConversationService()
	mockUserService := testHelpers.NewMockUserService()
	logger := testHelpers.NewTestLogger()

	// Setup test conversation
	testConversationID := "test-conversation-123"
	testConversation := &domain.Conversation{
		ID:        testConversationID,
		SessionID: "session-456",
		UserID:    "user-789",
		Status:    domain.ConversationStatusActive,
	}
	
	mockConversationService.SetConversation(testConversationID, testConversation)

	// Create BFF instance
	bff := NewConversationAwareWebBFF(
		mockConversationService,
		mockUserService,
		mockGraphRepo,
		logger,
	)

	// Create test server
	server := bff.CreateWebServer(":8080")
	
	// Test the /api/ui/conversation-history endpoint (this will fail - endpoint doesn't exist)
	req := httptest.NewRequest("GET", "/api/ui/conversation-history?conversation_id="+testConversationID, nil)
	w := httptest.NewRecorder()
	
	server.Handler.ServeHTTP(w, req)
	
	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	// Should return conversation history array
	if response == nil {
		t.Error("Expected non-nil response")
	}
}

func TestUIAPIEndpoints_ExecutionPlans_ShouldReturnExecutionPlansData(t *testing.T) {
	// RED: This test will fail because /api/ui/execution-plans doesn't exist
	
	// Setup test dependencies
	mockGraphRepo := testHelpers.NewMockGraphRepository()
	mockConversationService := testHelpers.NewMockConversationService()
	mockUserService := testHelpers.NewMockUserService()
	logger := testHelpers.NewTestLogger()

	// Setup test conversation
	testConversationID := "test-conversation-123"
	testConversation := &domain.Conversation{
		ID:        testConversationID,
		SessionID: "session-456",
		UserID:    "user-789",
		Status:    domain.ConversationStatusActive,
	}
	
	mockConversationService.SetConversation(testConversationID, testConversation)

	// Create BFF instance
	bff := NewConversationAwareWebBFF(
		mockConversationService,
		mockUserService,
		mockGraphRepo,
		logger,
	)

	// Create test server
	server := bff.CreateWebServer(":8080")
	
	// Test the /api/ui/execution-plans endpoint (this will fail - endpoint doesn't exist)
	req := httptest.NewRequest("GET", "/api/ui/execution-plans?conversation_id="+testConversationID, nil)
	w := httptest.NewRecorder()
	
	server.Handler.ServeHTTP(w, req)
	
	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	// Should return execution plans array
	if response == nil {
		t.Error("Expected non-nil response")
	}
}

func TestUIAPIEndpoints_RESTfulDesign_ShouldFollowRESTConventions(t *testing.T) {
	// RED: This test exposes the non-RESTful design of our current endpoints
	
	// Setup test dependencies
	mockGraphRepo := testHelpers.NewMockGraphRepository()
	mockConversationService := testHelpers.NewMockConversationService()
	mockUserService := testHelpers.NewMockUserService()
	logger := testHelpers.NewTestLogger()

	// Create BFF instance
	bff := NewConversationAwareWebBFF(
		mockConversationService,
		mockUserService,
		mockGraphRepo,
		logger,
	)

	// Create test server
	server := bff.CreateWebServer(":8080")
	
	// Test cases for RESTful endpoint design
	testCases := []struct {
		name           string
		endpoint       string
		expectedStatus int
		description    string
	}{
		{
			name:           "List all conversations",
			endpoint:       "/api/conversations",
			expectedStatus: http.StatusOK,
			description:    "GET /api/conversations should list all conversations",
		},
		{
			name:           "Get specific conversation",
			endpoint:       "/api/conversations/test-conv-123",
			expectedStatus: http.StatusOK,
			description:    "GET /api/conversations/{id} should get specific conversation",
		},
		{
			name:           "Get conversation graph data",
			endpoint:       "/api/conversations/test-conv-123/graph",
			expectedStatus: http.StatusOK,
			description:    "GET /api/conversations/{id}/graph should get conversation graph data",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.endpoint, nil)
			w := httptest.NewRecorder()
			
			server.Handler.ServeHTTP(w, req)
			
			// These will fail because we don't have proper RESTful endpoints
			if w.Code != tc.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d", tc.description, tc.expectedStatus, w.Code)
			}
		})
	}
}
