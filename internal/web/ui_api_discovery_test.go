package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/logging"
	"neuromesh/internal/orchestrator/application"
	"neuromesh/testHelpers"
)

// TestUIAPIEndpoints_Discovery tests the API endpoints needed for the React UI
// This test follows our TDD protocol - it will fail initially and drive our implementation
func TestUIAPIEndpoints_Discovery(t *testing.T) {
	// This test defines the API contract needed for our React UI
	// Following TDD RED phase - these endpoints don't exist yet

	t.Run("Should_Expose_Graph_Data_Endpoint", func(t *testing.T) {
		// GIVEN: We have a conversation BFF with graph data
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		// WHEN: React UI requests graph data for a conversation
		req := httptest.NewRequest("GET", "/api/graph/conversation/conv-123", nil)
		w := httptest.NewRecorder()

		// This will fail initially - we need to implement this endpoint
		handler := bff.GraphDataHandler()
		handler.ServeHTTP(w, req)

		// THEN: Should return structured graph data for React Flow
		assert.Equal(t, http.StatusOK, w.Code)

		var graphData GraphDataResponse
		err := json.Unmarshal(w.Body.Bytes(), &graphData)
		require.NoError(t, err)

		// Should contain nodes and edges for React Flow
		assert.NotEmpty(t, graphData.Nodes)
		assert.NotEmpty(t, graphData.Edges)
		assert.Equal(t, "conv-123", graphData.ConversationID)
	})

	t.Run("Should_Expose_Execution_Plan_Endpoint", func(t *testing.T) {
		// GIVEN: We have execution plans in the graph
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		// WHEN: React UI requests execution plan details
		req := httptest.NewRequest("GET", "/api/execution-plan/plan-456", nil)
		w := httptest.NewRecorder()

		// This will fail initially - we need to implement this endpoint
		handler := bff.ExecutionPlanHandler()
		handler.ServeHTTP(w, req)

		// THEN: Should return structured execution plan data
		assert.Equal(t, http.StatusOK, w.Code)

		var planData ExecutionPlanResponse
		err := json.Unmarshal(w.Body.Bytes(), &planData)
		require.NoError(t, err)

		assert.Equal(t, "plan-456", planData.ID)
		assert.NotEmpty(t, planData.Steps)
	})

	t.Run("Should_Expose_Conversation_History_Endpoint", func(t *testing.T) {
		// GIVEN: We have conversation history
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		// WHEN: React UI requests conversation history
		req := httptest.NewRequest("GET", "/api/conversations/session-789", nil)
		w := httptest.NewRecorder()

		// This will fail initially - we need to implement this endpoint
		handler := bff.ConversationHistoryHandler()
		handler.ServeHTTP(w, req)

		// THEN: Should return conversation messages
		assert.Equal(t, http.StatusOK, w.Code)

		var history ConversationHistoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &history)
		require.NoError(t, err)

		assert.Equal(t, "session-789", history.SessionID)
		assert.NotNil(t, history.Messages)
	})

	t.Run("Should_Expose_Real_Time_WebSocket_With_Typed_Messages", func(t *testing.T) {
		// GIVEN: We have enhanced WebSocket for typed real-time updates
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		// WHEN: React UI connects to enhanced WebSocket
		server := httptest.NewServer(bff.EnhancedWebSocketHandler())
		defer server.Close()

		// This will fail initially - we need to implement enhanced WebSocket
		// For now, just verify the handler exists
		handler := bff.EnhancedWebSocketHandler()
		assert.NotNil(t, handler)

		// TODO: Add actual WebSocket client test when implementation exists
	})

	t.Run("Should_Expose_Agent_Status_Endpoint", func(t *testing.T) {
		// GIVEN: We have agents in the system
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		// WHEN: React UI requests agent status
		req := httptest.NewRequest("GET", "/api/agents/status", nil)
		w := httptest.NewRecorder()

		// This will fail initially - we need to implement this endpoint
		handler := bff.AgentStatusHandler()
		handler.ServeHTTP(w, req)

		// THEN: Should return agent status information
		assert.Equal(t, http.StatusOK, w.Code)

		var agentStatus AgentStatusResponse
		err := json.Unmarshal(w.Body.Bytes(), &agentStatus)
		require.NoError(t, err)

		assert.NotNil(t, agentStatus.Agents)
	})
}

// TestUIDataModels_TypeScriptInterfaces tests the data structures needed for TypeScript
func TestUIDataModels_TypeScriptInterfaces(t *testing.T) {
	// This test defines the Go structs that will map to TypeScript interfaces
	// Following TDD RED phase - these structs don't exist yet

	t.Run("Should_Define_GraphDataResponse_Structure", func(t *testing.T) {
		// Test will fail until we define these structs
		graphData := GraphDataResponse{
			ConversationID: "conv-123",
			Nodes: []GraphNode{
				{
					ID:       "user-1",
					Type:     "user",
					Data:     map[string]interface{}{"name": "Test User"},
					Position: &NodePosition{X: 100, Y: 200},
				},
			},
			Edges: []GraphEdge{
				{
					ID:     "edge-1",
					Source: "user-1",
					Target: "conv-123",
					Type:   "created",
				},
			},
		}

		assert.Equal(t, "conv-123", graphData.ConversationID)
		assert.Len(t, graphData.Nodes, 1)
		assert.Len(t, graphData.Edges, 1)
	})

	t.Run("Should_Define_ExecutionPlanResponse_Structure", func(t *testing.T) {
		// Test will fail until we define these structs
		planData := ExecutionPlanResponse{
			ID:          "plan-456",
			Name:        "Test Plan",
			Description: "Test execution plan",
			Status:      "PENDING",
			Steps: []ExecutionStepData{
				{
					StepNumber:  1,
					Name:        "First Step",
					Description: "Execute first action",
					AgentName:   "text-processor",
					Status:      "PENDING",
					CompletedAt: nil,
				},
			},
		}

		assert.Equal(t, "plan-456", planData.ID)
		assert.Len(t, planData.Steps, 1)
	})
}

// Helper functions for test setup

func setupTestConversationBFF(t *testing.T) (*ConversationAwareWebBFF, func()) {
	// Create test dependencies
	logger := logging.NewStructuredLogger(logging.LevelDebug)

	// Create mock orchestrator using local web package mock
	orchestrator := &MockAIOrchestrator{
		responses: make(map[string]*application.OrchestratorResult),
	}

	// For now, create simple implementations that satisfy interface
	conversationService := testHelpers.NewMockConversationService()
	userService := testHelpers.NewMockUserService()

	// Add graph dependency for real data integration
	testGraph := testHelpers.NewCleanMockGraph() // Use clean graph for controlled testing

	// Create conversation-aware WebBFF
	bff := NewConversationAwareWebBFF(
		orchestrator,
		conversationService,
		userService,
		testGraph,
		logger,
	)

	cleanup := func() {
		// Cleanup test resources
	}

	return bff, cleanup
}
