package infrastructure

import (
	"context"
	"testing"

	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConversationGraphRepository_CrossDomainRelationships tests that the graph repository
// correctly retrieves cross-domain relationships between conversations and execution plans
// This is a critical test that ensures our graph visualization shows complete relationships
func TestConversationGraphRepository_CrossDomainRelationships(t *testing.T) {
	// RED: This test will fail because our current query doesn't include LINKED_TO_PLAN

	// Skip if no Neo4j connection available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	// Initialize Neo4j graph for testing
	graphConfig := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}

	neo4jGraph, err := graph.NewNeo4jGraph(ctx, graphConfig, logger)
	require.NoError(t, err, "Failed to connect to Neo4j")
	defer neo4jGraph.Close(ctx)

	// Clear any existing test data
	err = neo4jGraph.ClearTestData(ctx)
	require.NoError(t, err)

	repo := NewConversationGraphRepository(neo4jGraph, logger)

	conversationID := "conv-123"
	executionPlanID := "plan-456"

	// Create a conversation node
	conversationProps := map[string]interface{}{
		"id":         conversationID,
		"title":      "Test Conversation",
		"created_at": "2024-01-01T00:00:00Z",
	}
	err = neo4jGraph.AddNode(ctx, "Conversation", conversationID, conversationProps)
	require.NoError(t, err)

	// Create an execution plan node
	planProps := map[string]interface{}{
		"id":          executionPlanID,
		"title":       "Test Plan",
		"description": "A test execution plan",
		"created_at":  "2024-01-01T00:00:00Z",
	}
	err = neo4jGraph.AddNode(ctx, "ExecutionPlan", executionPlanID, planProps)
	require.NoError(t, err)

	// Create the cross-domain relationship (LINKED_TO_PLAN)
	linkProps := map[string]interface{}{
		"created_at": "2024-01-01T00:00:00Z",
	}
	err = neo4jGraph.AddEdge(ctx, "Conversation", conversationID, "ExecutionPlan", executionPlanID, "LINKED_TO_PLAN", linkProps)
	require.NoError(t, err)

	// Act
	graphData, err := repo.GetConversationGraph(ctx, conversationID)

	// Assert
	require.NoError(t, err, "GetConversationGraph should not return an error")

	// We should have both nodes in the graph
	assert.Len(t, graphData.Nodes, 2, "Should have conversation and execution plan nodes")

	// We should have the cross-domain relationship
	assert.Len(t, graphData.Edges, 1, "Should have the LINKED_TO_PLAN relationship")

	// Verify the conversation node
	conversationNode := findNodeByType(graphData.Nodes, "conversation")
	require.NotNil(t, conversationNode, "Should have a conversation node")
	assert.Equal(t, conversationID, conversationNode.Data["id"])

	// Verify the execution plan node
	planNode := findNodeByType(graphData.Nodes, "execution_plan")
	require.NotNil(t, planNode, "Should have an execution plan node")
	assert.Equal(t, executionPlanID, planNode.Data["id"])

	// Verify the relationship
	require.Len(t, graphData.Edges, 1, "Should have exactly one edge")
	edge := graphData.Edges[0]
	assert.Equal(t, "linked_to_plan", edge.Type, "Edge should be of type 'linked_to_plan'")

	// The edge should connect the conversation to the execution plan
	// Note: We need to check the actual node IDs, not the domain IDs
	assert.True(t,
		(edge.Source == conversationNode.ID && edge.Target == planNode.ID) ||
			(edge.Source == planNode.ID && edge.Target == conversationNode.ID),
		"Edge should connect conversation and execution plan nodes")
}

// Helper function to find a node by type
func findNodeByType(nodes []domain.GraphNode, nodeType string) *domain.GraphNode {
	for _, node := range nodes {
		if node.Type == nodeType {
			return &node
		}
	}
	return nil
}
