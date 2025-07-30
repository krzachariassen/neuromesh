package graph

import (
	"context"
	"testing"

	"neuromesh/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddEdge_Idempotency tests that AddEdge is idempotent and doesn't create duplicate relationships
func TestAddEdge_Idempotency(t *testing.T) {
	t.Run("should_not_create_duplicate_relationships_when_called_multiple_times", func(t *testing.T) {
		// Setup
		graph := setupTestGraph(t)
		ctx := context.Background()

		// Create test nodes
		nodeAID := "test-node-a"
		nodeBID := "test-node-b"

		nodeAProps := map[string]interface{}{
			"id":   nodeAID,
			"name": "Node A",
		}
		nodeBProps := map[string]interface{}{
			"id":   nodeBID,
			"name": "Node B",
		}

		err := graph.AddNode(ctx, "TestNodeA", nodeAID, nodeAProps)
		require.NoError(t, err, "Failed to create test node A")

		err = graph.AddNode(ctx, "TestNodeB", nodeBID, nodeBProps)
		require.NoError(t, err, "Failed to create test node B")

		// Create relationship properties
		relationshipProps := map[string]interface{}{
			"created_at": "2025-07-30T12:00:00Z",
			"type":       "test_relationship",
		}

		// Call AddEdge multiple times with the same parameters
		err = graph.AddEdge(ctx, "TestNodeA", nodeAID, "TestNodeB", nodeBID, "CONNECTS_TO", relationshipProps)
		require.NoError(t, err, "First AddEdge call should succeed")

		err = graph.AddEdge(ctx, "TestNodeA", nodeAID, "TestNodeB", nodeBID, "CONNECTS_TO", relationshipProps)
		require.NoError(t, err, "Second AddEdge call should succeed (idempotent)")

		err = graph.AddEdge(ctx, "TestNodeA", nodeAID, "TestNodeB", nodeBID, "CONNECTS_TO", relationshipProps)
		require.NoError(t, err, "Third AddEdge call should succeed (idempotent)")

		// Verify that only one relationship exists
		edges, err := graph.GetEdges(ctx, "TestNodeA", nodeAID)
		require.NoError(t, err, "Failed to get edges")

		// Should have exactly one relationship
		assert.Len(t, edges, 1, "Should have exactly one relationship, not duplicates")

		// Cleanup
		_ = graph.DeleteNode(ctx, "TestNodeA", nodeAID)
		_ = graph.DeleteNode(ctx, "TestNodeB", nodeBID)
	})

	t.Run("should_create_different_relationships_between_same_nodes", func(t *testing.T) {
		// Setup
		graph := setupTestGraph(t)
		ctx := context.Background()

		// Create test nodes
		nodeAID := "test-node-c"
		nodeBID := "test-node-d"

		nodeAProps := map[string]interface{}{
			"id":   nodeAID,
			"name": "Node C",
		}
		nodeBProps := map[string]interface{}{
			"id":   nodeBID,
			"name": "Node D",
		}

		err := graph.AddNode(ctx, "TestNodeC", nodeAID, nodeAProps)
		require.NoError(t, err, "Failed to create test node C")

		err = graph.AddNode(ctx, "TestNodeD", nodeBID, nodeBProps)
		require.NoError(t, err, "Failed to create test node D")

		// Create different types of relationships
		relationshipProps := map[string]interface{}{
			"created_at": "2025-07-30T12:00:00Z",
		}

		err = graph.AddEdge(ctx, "TestNodeC", nodeAID, "TestNodeD", nodeBID, "CONNECTS_TO", relationshipProps)
		require.NoError(t, err, "First relationship type should be created")

		err = graph.AddEdge(ctx, "TestNodeC", nodeAID, "TestNodeD", nodeBID, "BELONGS_TO", relationshipProps)
		require.NoError(t, err, "Second relationship type should be created")

		// Verify that both relationships exist (different types)
		edges, err := graph.GetEdges(ctx, "TestNodeC", nodeAID)
		require.NoError(t, err, "Failed to get edges")

		// Should have exactly two relationships (different types)
		assert.Len(t, edges, 2, "Should have two relationships with different types")

		// Cleanup
		_ = graph.DeleteNode(ctx, "TestNodeC", nodeAID)
		_ = graph.DeleteNode(ctx, "TestNodeD", nodeBID)
	})
}

// setupTestGraph creates a test graph instance
func setupTestGraph(t *testing.T) Graph {
	logger := logging.NewNoOpLogger()
	config := GraphConfig{
		Backend:       GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}

	ctx := context.Background()
	graph, err := NewNeo4jGraph(ctx, config, logger)
	require.NoError(t, err, "Failed to create test graph")

	// Clear any existing test data
	err = graph.ClearTestData(ctx)
	require.NoError(t, err, "Failed to clear test data")

	return graph
}
