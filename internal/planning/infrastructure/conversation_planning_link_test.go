package infrastructure

import (
	"context"
	"testing"

	"neuromesh/internal/planning/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlanningResultConversationLinking tests that planning results are properly linked to conversations
// This test exposes the current design gap where planning results are only linked to messages, not conversations
func TestPlanningResultConversationLinking(t *testing.T) {
	// Setup test graph
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphPlanningRepository(graph)
	require.NoError(t, repo.EnsureSchema(ctx))

	// Create test data
	userID := "test-user-123"
	sessionID := "test-session-456"
	conversationID := "test-conversation-789"
	messageID := "test-message-012"

	// Create the graph structure: User -> Session -> Conversation -> Message
	createTestUser(t, graph, userID, sessionID)
	createTestConversation(t, graph, userID, sessionID, conversationID)
	createTestMessage(t, graph, conversationID, messageID, "user", "What is the weather?")

	// Create a planning result linked to a message
	planningResult := domain.NewRespondDirectlyPlanningResult(
		messageID, // requestID is messageID
		"test user request",
		"guidance",
		95,
		[]string{},
		[]string{},
		"Direct response reasoning",
		"This is a direct response",
	)

	// Store the planning result
	err := repo.Store(ctx, planningResult)
	require.NoError(t, err)

	// Link to request (message)
	err = repo.LinkToRequest(ctx, planningResult.ID, messageID)
	require.NoError(t, err)

	// This should exist: Link to conversation
	// Currently this method doesn't exist and will fail (RED phase)
	err = repo.LinkToConversation(ctx, planningResult.ID, conversationID)
	assert.NoError(t, err, "Planning results should be linkable to conversations")

	// Verify the conversation link exists in the graph by querying edges with targets
	// Use GetEdgesWithTargets to get the target_id information
	edges, err := graph.GetEdgesWithTargets(ctx, "Conversation", conversationID)
	require.NoError(t, err)

	// Look for HAS_PLANNING relationship to our planning result
	found := false
	for _, edge := range edges {
		if edge["type"] == "HAS_PLANNING" && edge["target_id"] == planningResult.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find HAS_PLANNING relationship from conversation to planning result")

	// Cleanup
	_ = repo.Delete(ctx, planningResult.ID)
}
