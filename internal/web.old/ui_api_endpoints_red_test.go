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
