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

// TestGraphConversationRepository_GraphNative tests conversation repository with graph-native entities
func TestGraphConversationRepository_GraphNative(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

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

	// Create repository
	repo := NewGraphConversationRepository(g)

	t.Run("should create graph-native conversation without embedded foreign keys", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversation with graph-native structure
		conversation, err := domain.NewConversation("conv-123")
		require.NoError(t, err, "Failed to create conversation")

		// Store conversation
		err = repo.CreateConversation(ctx, conversation)
		assert.NoError(t, err, "CreateConversation should succeed")

		// Verify the conversation was created by retrieving it
		retrievedConversation, err := repo.GetConversation(ctx, "conv-123")
		assert.NoError(t, err, "Should be able to retrieve created conversation")
		assert.NotNil(t, retrievedConversation, "Retrieved conversation should not be nil")
		assert.Equal(t, conversation.ID, retrievedConversation.ID, "Conversation ID should match")
		assert.Equal(t, conversation.Status, retrievedConversation.Status, "Conversation status should match")

		// Verify no embedded foreign keys exist
		assert.Empty(t, retrievedConversation.Messages, "Messages should be empty initially")
	})

	t.Run("should create and store messages in graph-native conversation", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")
		err = repo.EnsureMessageSchema(ctx)
		require.NoError(t, err, "Failed to ensure message schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Add message to conversation
		messageID := "msg-456"
		err = conversation.AddMessage(messageID, domain.MessageRoleUser, "Test message content", nil)
		require.NoError(t, err, "Failed to add message")

		// Store updated conversation
		err = repo.UpdateConversation(ctx, conversation)
		assert.NoError(t, err, "UpdateConversation should succeed")

		// Verify message was stored
		retrievedConversation, err := repo.GetConversation(ctx, "conv-123")
		assert.NoError(t, err, "Should be able to retrieve conversation")
		assert.Len(t, retrievedConversation.Messages, 1, "Should have one message")
		assert.Equal(t, messageID, retrievedConversation.Messages[0].ID, "Message ID should match")
		assert.Equal(t, "Test message content", retrievedConversation.Messages[0].Content, "Message content should match")
	})

	t.Run("should establish graph relationships without embedded foreign keys", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Create conversation with graph-native structure
		conversation, err := domain.NewConversation("conv-123")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Test relationship methods (these should work with graph edges, not embedded properties)
		sessionID := "session-456"
		userID := "user-789"
		projectID := "project-abc"

		// These methods should create graph relationships, not set properties
		err = repo.LinkConversationToSession(ctx, "conv-123", sessionID)
		assert.NoError(t, err, "Should be able to link conversation to session")

		err = repo.LinkConversationToUser(ctx, "conv-123", userID)
		assert.NoError(t, err, "Should be able to link conversation to user")

		err = repo.LinkConversationToProject(ctx, "conv-123", projectID)
		assert.NoError(t, err, "Should be able to link conversation to project")

		// Verify relationships exist in graph (this would require graph traversal queries)
		// For now, just verify no errors occurred
	})

	t.Run("should handle message filtering by role in graph-native way", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")
		err = repo.EnsureMessageSchema(ctx)
		require.NoError(t, err, "Failed to ensure message schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Add messages with different roles
		err = conversation.AddMessage("msg-1", domain.MessageRoleUser, "User message", nil)
		require.NoError(t, err, "Failed to add user message")

		err = conversation.AddMessage("msg-2", domain.MessageRoleAssistant, "Assistant response", nil)
		require.NoError(t, err, "Failed to add assistant message")

		err = conversation.AddMessage("msg-3", domain.MessageRoleSystem, "System message", nil)
		require.NoError(t, err, "Failed to add system message")

		// Update conversation with messages
		err = repo.UpdateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to update conversation")

		// Test message filtering in the domain entity (graph-native approach)
		retrievedConversation, err := repo.GetConversation(ctx, "conv-123")
		require.NoError(t, err, "Failed to retrieve conversation")

		userMessages := retrievedConversation.GetMessagesByRole(domain.MessageRoleUser)
		assert.Len(t, userMessages, 1, "Should have one user message")
		assert.Equal(t, "User message", userMessages[0].Content, "User message content should match")

		assistantMessages := retrievedConversation.GetMessagesByRole(domain.MessageRoleAssistant)
		assert.Len(t, assistantMessages, 1, "Should have one assistant message")
		assert.Equal(t, "Assistant response", assistantMessages[0].Content, "Assistant message content should match")

		systemMessages := retrievedConversation.GetMessagesByRole(domain.MessageRoleSystem)
		assert.Len(t, systemMessages, 1, "Should have one system message")
		assert.Equal(t, "System message", systemMessages[0].Content, "System message content should match")
	})
}
