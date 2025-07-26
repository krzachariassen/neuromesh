package web

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/testHelpers"
)

// TDD RED Phase: Backend API Enhancement Tests
// These tests will drive the real integration with existing repositories
// Following TDD protocol: RED (failing tests) → GREEN (minimal implementation) → REFACTOR

func TestUIAPIService_GraphDataIntegration_TDD_RED(t *testing.T) {
	// TDD RED: This test will fail because we're using mock data instead of real graph queries
	t.Run("Should_Query_Real_Neo4j_Graph_Data", func(t *testing.T) {
		// GIVEN: We have a graph with test data representing real conversation and user data
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		ctx := context.Background()

		// Create a test graph with realistic data
		testGraph := testHelpers.NewMockGraph()

		// Add test conversation data to graph
		testConversationID := "test-conversation-123"
		err := testGraph.AddNode(ctx, "Conversation", testConversationID, map[string]interface{}{
			"id":         testConversationID,
			"user_id":    "real-user-456",
			"status":     "active",
			"created_at": "2025-07-26T10:30:00Z",
		})
		require.NoError(t, err)

		// Add test user data to graph
		err = testGraph.AddNode(ctx, "User", "real-user-456", map[string]interface{}{
			"id":   "real-user-456",
			"name": "Real Test User",
		})
		require.NoError(t, err)

		// WHEN: We request graph data for this conversation with a real graph
		uiService := NewUIAPIServiceWithGraph(bff.conversationService, bff.userService, testGraph)
		graphData, err := uiService.GetGraphData(ctx, testConversationID)
		require.NoError(t, err)

		// THEN: Should return REAL data from graph, not mock data
		assert.NotEqual(t, "Test User", graphData.Nodes[0].Data["name"],
			"Should return real user data from graph, not mock 'Test User'")

		// Should have real conversation node with actual conversation data
		conversationNode := findNodeByType(graphData.Nodes, "conversation")
		require.NotNil(t, conversationNode, "Should have conversation node from real data")
		assert.Equal(t, testConversationID, conversationNode.ID, "Should use real conversation ID")

		// Should have real user data
		userNode := findNodeByType(graphData.Nodes, "user")
		require.NotNil(t, userNode, "Should have user node from real data")
		assert.Equal(t, "Real Test User", userNode.Data["name"], "Should have real user name from graph")
		assert.Equal(t, "real-user-456", userNode.ID, "Should have real user ID from graph")

		// Should have real edges connecting entities
		assert.True(t, len(graphData.Edges) > 0, "Should have real edges from graph")
		assert.Contains(t, graphData.Edges[0].ID, "real-user-456", "Should have real edge IDs with user ID")
	})

	t.Run("Should_Query_Real_Execution_Plans", func(t *testing.T) {
		// TDD RED: This test will fail because we return mock execution plan data
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		ctx := context.Background()
		testPlanID := "test-plan-456"

		// Create a test graph with execution plan data
		testGraph := testHelpers.NewMockGraph()

		// Add test execution plan data to graph
		err := testGraph.AddNode(ctx, "ExecutionPlan", testPlanID, map[string]interface{}{
			"id":          testPlanID,
			"name":        "Real Medical Analysis Plan",
			"description": "Analyze medical case with real agents",
			"status":      "RUNNING",
			"created_at":  "2025-07-26T10:35:00Z",
		})
		require.NoError(t, err)

		// Add real execution step data
		err = testGraph.AddNode(ctx, "ExecutionStep", "step-1", map[string]interface{}{
			"id":           "step-1",
			"plan_id":      testPlanID,
			"name":         "Medical Data Analysis",
			"description":  "Analyze patient symptoms and history",
			"agent_name":   "medical-analyzer",
			"status":       "COMPLETED",
			"completed_at": "2025-07-26T10:36:00Z",
		})
		require.NoError(t, err)

		// WHEN: We request execution plan data
		uiService := NewUIAPIServiceWithGraph(bff.conversationService, bff.userService, testGraph)
		planData, err := uiService.GetExecutionPlan(ctx, testPlanID)
		require.NoError(t, err)

		// THEN: Should return REAL execution plan data, not mock data
		assert.NotEqual(t, "Test Plan", planData.Name,
			"Should return real plan name from graph, not mock 'Test Plan'")
		assert.Equal(t, "Real Medical Analysis Plan", planData.Name, "Should use real plan name from graph")
		assert.Equal(t, testPlanID, planData.ID, "Should use the requested plan ID")

		// Should have real execution steps
		require.True(t, len(planData.Steps) > 0, "Should have real execution steps")
		assert.NotEqual(t, "text-processor", planData.Steps[0].AgentName,
			"Should return real agent names from plan, not mock 'text-processor'")
		assert.Equal(t, "medical-analyzer", planData.Steps[0].AgentName, "Should use real agent name from graph")
	})

	t.Run("Should_Query_Real_Conversation_History", func(t *testing.T) {
		// TDD RED: This test will fail because we return mock conversation data
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		ctx := context.Background()
		testSessionID := "test-session-789"

		// Create a test graph with conversation history data
		testGraph := testHelpers.NewMockGraph()

		// Add real conversation data for this session
		conversationID1 := "real-conv-001"
		err := testGraph.AddNode(ctx, "Conversation", conversationID1, map[string]interface{}{
			"id":         conversationID1,
			"session_id": testSessionID,
			"user_id":    "real-user-123",
			"status":     "completed",
			"created_at": "2025-07-26T09:00:00Z",
		})
		require.NoError(t, err)

		conversationID2 := "real-conv-002"
		err = testGraph.AddNode(ctx, "Conversation", conversationID2, map[string]interface{}{
			"id":         conversationID2,
			"session_id": testSessionID,
			"user_id":    "real-user-123",
			"status":     "active",
			"created_at": "2025-07-26T10:00:00Z",
		})
		require.NoError(t, err)

		// Add real message data
		err = testGraph.AddNode(ctx, "ConversationMessage", "real-msg-001", map[string]interface{}{
			"id":              "real-msg-001",
			"conversation_id": conversationID1,
			"role":            "user",
			"content":         "I need help with medical diagnosis",
			"timestamp":       "2025-07-26T09:01:00Z",
		})
		require.NoError(t, err)

		// WHEN: We request conversation history for this session
		uiService := NewUIAPIServiceWithGraph(bff.conversationService, bff.userService, testGraph)
		historyData, err := uiService.GetConversationHistory(ctx, testSessionID)
		require.NoError(t, err)

		// THEN: Should return REAL conversation data, not mock data
		assert.Equal(t, testSessionID, historyData.SessionID, "Should use requested session ID")

		// Should have real conversations from graph
		assert.True(t, len(historyData.Conversations) >= 2, "Should return real conversations from graph")

		// Find our real conversations
		foundConv1 := false
		foundConv2 := false
		for _, conv := range historyData.Conversations {
			if conv.ID == conversationID1 {
				foundConv1 = true
				assert.Equal(t, "completed", conv.Status, "Should have real conversation status")
			}
			if conv.ID == conversationID2 {
				foundConv2 = true
				assert.Equal(t, "active", conv.Status, "Should have real conversation status")
			}
		}
		assert.True(t, foundConv1, "Should find first real conversation")
		assert.True(t, foundConv2, "Should find second real conversation")

		// The key success: should not return hardcoded "conv-123"
		for _, conv := range historyData.Conversations {
			assert.NotEqual(t, "conv-123", conv.ID,
				"Should return real conversation IDs from graph, not mock 'conv-123'")
		}

		// Should have real messages
		assert.True(t, len(historyData.Messages) > 0, "Should have real messages from graph")
		foundRealMessage := false
		for _, msg := range historyData.Messages {
			if msg.Content == "I need help with medical diagnosis" {
				foundRealMessage = true
				break
			}
		}
		assert.True(t, foundRealMessage, "Should find real message content from graph")
	})
}

func TestUIAPI_CORS_Authentication_TDD_RED(t *testing.T) {
	// TDD RED: These tests will fail because we don't have CORS/auth middleware yet
	t.Run("Should_Enable_CORS_For_React_App", func(t *testing.T) {
		// This test demonstrates the need for CORS middleware
		// Currently our handlers don't include CORS headers for React development

		// For now, we'll create a placeholder test that documents the requirement
		t.Skip("TODO: Implement CORS middleware for React development on localhost:3000")

		// When implemented, this should:
		// - Add CORS middleware to all API handlers
		// - Allow requests from localhost:3000 during development
		// - Support preflight OPTIONS requests
	})

	t.Run("Should_Have_Error_Handling_Middleware", func(t *testing.T) {
		// TDD RED: This test documents need for proper error handling
		bff, cleanup := setupTestConversationBFF(t)
		defer cleanup()

		// WHEN: An invalid request is made (malformed conversation ID)
		// THEN: Should return proper JSON error response (not implemented yet)
		t.Skip("TODO: Implement error handling middleware for consistent JSON error responses")

		// Real implementation should:
		// - Return JSON error responses instead of plain text
		// - Include error codes and structured error information
		// - Log errors appropriately

		// For now, verify basic error handling exists
		handler := bff.GraphDataHandler()
		assert.NotNil(t, handler, "Handler should exist")
	})
}

// Helper function to find nodes by type
func findNodeByType(nodes []GraphNode, nodeType string) *GraphNode {
	for _, node := range nodes {
		if node.Type == nodeType {
			return &node
		}
	}
	return nil
}
