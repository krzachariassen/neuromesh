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

// TestConversationGraphRepository_GetConversationGraph tests the clean architecture implementation
func TestConversationGraphRepository_GetConversationGraph(t *testing.T) {
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

	// Create the clean architecture service
	graphService := NewConversationGraphRepository(neo4jGraph, logger)

	t.Run("should implement domain interface", func(t *testing.T) {
		// This test ensures our repository implements the domain interface
		var _ domain.ConversationGraphService = graphService
	})

	t.Run("should return empty graph for non-existent conversation", func(t *testing.T) {
		// Test that we get a valid response even when no data exists
		graphData, err := graphService.GetConversationGraph(ctx, "non-existent-conversation")

		assert.NoError(t, err)
		assert.NotNil(t, graphData)
		assert.Empty(t, graphData.Nodes)
		assert.Empty(t, graphData.Edges)
	})

	t.Run("should handle subgraph queries", func(t *testing.T) {
		// Test subgraph functionality
		graphData, err := graphService.GetConversationSubgraph(ctx, "test-conversation", []string{"user", "agent"})

		assert.NoError(t, err)
		assert.NotNil(t, graphData)
		// Should return empty for non-existent conversation but not error
		assert.Empty(t, graphData.Nodes)
		assert.Empty(t, graphData.Edges)
	})

	t.Run("should return graph stats", func(t *testing.T) {
		// Test stats functionality
		stats, err := graphService.GetGraphStats(ctx, "test-conversation")

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		// Should contain basic stats even if empty
		assert.Contains(t, stats, "total_nodes")
		assert.Contains(t, stats, "total_edges")
	})
}
