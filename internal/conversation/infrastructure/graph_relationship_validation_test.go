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

// TestConversationUserRelationshipCreation validates User-Conversation graph relationships
func TestConversationUserRelationshipCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx := context.Background()
	logger := logging.NewNoOpLogger()

	// Setup graph connection
	config := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}
	g, err := graph.NewNeo4jGraph(ctx, config, logger)
	require.NoError(t, err, "Failed to connect to Neo4j")
	defer g.Close(ctx)

	repo := NewGraphConversationRepository(g)

	// Clean up any existing test data
	err = g.ClearTestData(ctx)
	require.NoError(t, err, "Failed to clean up test data")

	// Ensure schema exists first
	err = repo.EnsureConversationSchema(ctx)
	require.NoError(t, err, "Failed to ensure conversation schema")

	// Test data
	conversationID := "conv-123"
	userID := "user-456"
	sessionID := "session-789"
	projectID := "project-abc"

	// CRITICAL: Create User and Session nodes first (they don't exist yet!)
	userProps := map[string]interface{}{
		"id":         userID,
		"created_at": "2025-07-30T10:00:00Z",
	}
	err = g.AddNode(ctx, "User", userID, userProps)
	require.NoError(t, err, "Should create User node")

	sessionProps := map[string]interface{}{
		"id":         sessionID,
		"created_at": "2025-07-30T10:00:00Z",
	}
	err = g.AddNode(ctx, "Session", sessionID, sessionProps)
	require.NoError(t, err, "Should create Session node")

	projectProps := map[string]interface{}{
		"id":         projectID,
		"created_at": "2025-07-30T10:00:00Z",
		"name":       "Test Project",
		"status":     "active",
	}
	err = g.AddNode(ctx, "Project", projectID, projectProps)
	require.NoError(t, err, "Should create Project node")

	// Create conversation
	conversation, err := domain.NewConversation(conversationID)
	require.NoError(t, err)

	err = repo.CreateConversation(ctx, conversation)
	require.NoError(t, err)

	// Create the relationships
	err = repo.LinkConversationToUser(ctx, conversationID, userID)
	require.NoError(t, err, "LinkConversationToUser should succeed")

	err = repo.LinkConversationToSession(ctx, conversationID, sessionID)
	require.NoError(t, err, "LinkConversationToSession should succeed")

	// Verify relationships exist in graph using GetEdgesWithTargets method
	t.Run("UserConversationRelationship", func(t *testing.T) {
		// Check if User node has outgoing PARTICIPANT_IN relationship to conversation
		edges, err := g.GetEdgesWithTargets(ctx, "User", userID)
		require.NoError(t, err, "Should be able to get user edges")

		t.Logf("User edges found: %+v", edges)

		// Look for PARTICIPANT_IN relationship to our conversation
		found := false
		for _, edge := range edges {
			t.Logf("Checking edge: %+v", edge)
			if edgeType, ok := edge["type"].(string); ok && edgeType == "PARTICIPANT_IN" {
				if targetID, ok := edge["target_id"].(string); ok && targetID == conversationID {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "User should have PARTICIPANT_IN relationship to conversation")
	})

	t.Run("SessionConversationRelationship", func(t *testing.T) {
		// Check if Session node has outgoing IN_SESSION relationship to conversation
		edges, err := g.GetEdgesWithTargets(ctx, "Session", sessionID)
		require.NoError(t, err, "Should be able to get session edges")

		t.Logf("Session edges found: %+v", edges)

		// Look for IN_SESSION relationship to our conversation
		found := false
		for _, edge := range edges {
			t.Logf("Checking edge: %+v", edge)
			if edgeType, ok := edge["type"].(string); ok && edgeType == "IN_SESSION" {
				if targetID, ok := edge["target_id"].(string); ok && targetID == conversationID {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Session should have IN_SESSION relationship to conversation")
	})
}
