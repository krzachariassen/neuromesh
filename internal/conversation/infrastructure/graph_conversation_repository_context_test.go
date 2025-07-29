package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
)

// TestGraphConversationRepository_GetConversationContext_E2E tests real graph traversal
func TestGraphConversationRepository_GetConversationContext_E2E(t *testing.T) {
	// Skip if no Neo4j available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("should_traverse_real_graph_relationships_for_context", func(t *testing.T) {
		// Setup: Real Neo4j graph
		logger := logging.NewNoOpLogger()
		config := graph.GraphConfig{
			Backend:       graph.GraphBackendNeo4j,
			Neo4jURL:      "bolt://localhost:7687",
			Neo4jUser:     "neo4j", 
			Neo4jPassword: "orchestrator123",
		}

		ctx := context.Background()
		neo4jGraph, err := graph.NewNeo4jGraph(ctx, config, logger)
		require.NoError(t, err)
		defer neo4jGraph.Close(ctx)

		repo := NewGraphConversationRepository(neo4jGraph)

		// Ensure schemas exist
		require.NoError(t, repo.EnsureConversationSchema(ctx))

		// Test data IDs
		conversationID := "test-conv-e2e-123"
		sessionID := "test-session-456"
		userID := "test-user-789"
		projectID := "test-project-abc"

		// Clean up any existing test data
		neo4jGraph.DeleteNode(ctx, "Conversation", conversationID)
		neo4jGraph.DeleteNode(ctx, "Session", sessionID)
		neo4jGraph.DeleteNode(ctx, "User", userID)
		neo4jGraph.DeleteNode(ctx, "Project", projectID)

		// Create test nodes
		err = neo4jGraph.AddNode(ctx, "Session", sessionID, map[string]interface{}{
			"id": sessionID,
			"created_at": "2025-07-29T10:00:00Z",
		})
		require.NoError(t, err)

		err = neo4jGraph.AddNode(ctx, "User", userID, map[string]interface{}{
			"id": userID,
			"email": "test@example.com",
		})
		require.NoError(t, err)

		err = neo4jGraph.AddNode(ctx, "Project", projectID, map[string]interface{}{
			"id": projectID,
			"name": "Real Test Project",
		})
		require.NoError(t, err)

		// Create conversation
		conversation, err := domain.NewConversation(conversationID, sessionID, userID, "project-abc")
		require.NoError(t, err)

		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err)

		// Create relationships using repository methods
		err = repo.LinkConversationToSession(ctx, conversationID, sessionID)
		require.NoError(t, err)

		err = repo.LinkConversationToUser(ctx, conversationID, userID)
		require.NoError(t, err)

		// Create project relationship (using the proper constant)
		err = neo4jGraph.AddEdge(ctx, "Conversation", conversationID, "Project", projectID, RelationshipBelongsTo, map[string]interface{}{
			"created_at": "2025-07-29T10:00:00Z",
		})
		require.NoError(t, err)

		// Act: Get conversation context using real graph traversal
		contextData, err := repo.GetConversationContext(ctx, conversationID)

		// Assert: Should get real data from graph relationships
		require.NoError(t, err)
		assert.Equal(t, conversationID, contextData.ConversationID)
		assert.Equal(t, sessionID, contextData.SessionID, "Should get real session ID from graph traversal")
		assert.Equal(t, userID, contextData.UserID, "Should get real user ID from graph traversal")
		assert.Equal(t, projectID, contextData.ProjectID, "Should get real project ID from graph traversal")
		assert.Equal(t, "Real Test Project", contextData.ProjectName, "Should get real project name from graph traversal")

		// Clean up
		neo4jGraph.DeleteNode(ctx, "Conversation", conversationID)
		neo4jGraph.DeleteNode(ctx, "Session", sessionID)
		neo4jGraph.DeleteNode(ctx, "User", userID)
		neo4jGraph.DeleteNode(ctx, "Project", projectID)
	})

	t.Run("should_handle_missing_optional_relationships", func(t *testing.T) {
		// Setup: Real Neo4j graph
		logger := logging.NewNoOpLogger()
		config := graph.GraphConfig{
			Backend:       graph.GraphBackendNeo4j,
			Neo4jURL:      "bolt://localhost:7687",
			Neo4jUser:     "neo4j", 
			Neo4jPassword: "orchestrator123",
		}

		ctx := context.Background()
		neo4jGraph, err := graph.NewNeo4jGraph(ctx, config, logger)
		require.NoError(t, err)
		defer neo4jGraph.Close(ctx)

		repo := NewGraphConversationRepository(neo4jGraph)

		// Test conversation with minimal relationships
		conversationID := "test-conv-minimal-456"
		sessionID := "test-session-minimal-789"
		userID := "test-user-minimal-123"

		// Clean up any existing test data
		neo4jGraph.DeleteNode(ctx, "Conversation", conversationID)
		neo4jGraph.DeleteNode(ctx, "Session", sessionID)
		neo4jGraph.DeleteNode(ctx, "User", userID)

		// Create only conversation, session, and user (no project)
		err = neo4jGraph.AddNode(ctx, "Session", sessionID, map[string]interface{}{
			"id": sessionID,
		})
		require.NoError(t, err)

		err = neo4jGraph.AddNode(ctx, "User", userID, map[string]interface{}{
			"id": userID,
		})
		require.NoError(t, err)

		conversation, err := domain.NewConversation(conversationID, sessionID, userID, "project-abc")
		require.NoError(t, err)

		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err)

		err = repo.LinkConversationToSession(ctx, conversationID, sessionID)
		require.NoError(t, err)

		err = repo.LinkConversationToUser(ctx, conversationID, userID)
		require.NoError(t, err)

		// Act: Get conversation context
		contextData, err := repo.GetConversationContext(ctx, conversationID)

		// Assert: Should handle missing project gracefully
		require.NoError(t, err)
		assert.Equal(t, conversationID, contextData.ConversationID)
		assert.Equal(t, sessionID, contextData.SessionID)
		assert.Equal(t, userID, contextData.UserID)
		assert.Empty(t, contextData.ProjectID, "Project ID should be empty when no project relationship exists")
		assert.Empty(t, contextData.ProjectName, "Project name should be empty when no project relationship exists")

		// Clean up
		neo4jGraph.DeleteNode(ctx, "Conversation", conversationID)
		neo4jGraph.DeleteNode(ctx, "Session", sessionID)
		neo4jGraph.DeleteNode(ctx, "User", userID)
	})
}
