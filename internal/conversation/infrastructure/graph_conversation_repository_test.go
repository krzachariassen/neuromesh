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

// TestGraphConversationRepository_ConversationSchema tests Conversation and Message schema creation
func TestGraphConversationRepository_ConversationSchema(t *testing.T) {
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

	t.Run("GREEN: should create and store Conversation nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123", "session-456", "user-789")
		require.NoError(t, err, "Failed to create conversation")

		// Now this should succeed
		err = repo.CreateConversation(ctx, conversation)
		assert.NoError(t, err, "CreateConversation should succeed")

		// Verify the conversation was created by retrieving it
		retrievedConversation, err := repo.GetConversation(ctx, "conv-123")
		assert.NoError(t, err, "Should be able to retrieve created conversation")
		assert.NotNil(t, retrievedConversation, "Retrieved conversation should not be nil")
		assert.Equal(t, conversation.ID, retrievedConversation.ID, "Conversation ID should match")
		assert.Equal(t, conversation.SessionID, retrievedConversation.SessionID, "Session ID should match")
		assert.Equal(t, conversation.UserID, retrievedConversation.UserID, "User ID should match")
		assert.Equal(t, conversation.Status, retrievedConversation.Status, "Conversation status should match")
	})

	t.Run("GREEN: should create and store Message nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schemas exist first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")
		err = repo.EnsureMessageSchema(ctx)
		require.NoError(t, err, "Failed to ensure message schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123", "session-456", "user-789")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Add message to conversation
		err = conversation.AddMessage("msg-1", domain.MessageRoleUser, "Hello world", nil)
		require.NoError(t, err, "Failed to add message to conversation")

		// Get the message
		messages := conversation.GetMessagesByRole(domain.MessageRoleUser)
		require.Len(t, messages, 1, "Should have one user message")
		message := &messages[0]

		// Store message in graph
		err = repo.AddMessage(ctx, "conv-123", message)
		assert.NoError(t, err, "AddMessage should succeed")

		// Verify the message was created by retrieving it
		retrievedMessages, err := repo.GetConversationMessages(ctx, "conv-123")
		assert.NoError(t, err, "Should be able to retrieve conversation messages")
		assert.Len(t, retrievedMessages, 1, "Should have one message")
		assert.Equal(t, message.ID, retrievedMessages[0].ID, "Message ID should match")
		assert.Equal(t, message.Role, retrievedMessages[0].Role, "Message role should match")
		assert.Equal(t, message.Content, retrievedMessages[0].Content, "Message content should match")
	})

	t.Run("GREEN: should establish Conversation-Session-User relationships", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schemas exist first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123", "session-456", "user-789")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Link conversation to session and user
		err = repo.LinkConversationToSession(ctx, "conv-123", "session-456")
		assert.NoError(t, err, "LinkConversationToSession should succeed")

		err = repo.LinkConversationToUser(ctx, "conv-123", "user-789")
		assert.NoError(t, err, "LinkConversationToUser should succeed")
	})

	t.Run("GREEN: should query conversations by user and session", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversations
		conv1, err := domain.NewConversation("conv-1", "session-456", "user-789")
		require.NoError(t, err, "Failed to create conversation 1")
		err = repo.CreateConversation(ctx, conv1)
		require.NoError(t, err, "Failed to create conversation 1")

		conv2, err := domain.NewConversation("conv-2", "session-456", "user-789")
		require.NoError(t, err, "Failed to create conversation 2")
		err = repo.CreateConversation(ctx, conv2)
		require.NoError(t, err, "Failed to create conversation 2")

		// Query conversations by user
		userConversations, err := repo.FindConversationsByUser(ctx, "user-789")
		assert.NoError(t, err, "FindConversationsByUser should succeed")
		assert.Len(t, userConversations, 2, "Should find 2 conversations for user")

		// Query conversations by session
		sessionConversations, err := repo.FindConversationsBySession(ctx, "session-456")
		assert.NoError(t, err, "FindConversationsBySession should succeed")
		assert.Len(t, sessionConversations, 2, "Should find 2 conversations for session")

		// Query active conversations
		activeConversations, err := repo.FindActiveConversations(ctx)
		assert.NoError(t, err, "FindActiveConversations should succeed")
		assert.Len(t, activeConversations, 2, "Should find 2 active conversations")
	})

	t.Run("GREEN: should handle message filtering by role", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schemas exist first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")
		err = repo.EnsureMessageSchema(ctx)
		require.NoError(t, err, "Failed to ensure message schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123", "session-456", "user-789")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Add different types of messages
		err = conversation.AddMessage("msg-1", domain.MessageRoleUser, "User message 1", nil)
		require.NoError(t, err, "Failed to add user message 1")
		err = conversation.AddMessage("msg-2", domain.MessageRoleAssistant, "Assistant response", nil)
		require.NoError(t, err, "Failed to add assistant message")
		err = conversation.AddMessage("msg-3", domain.MessageRoleUser, "User message 2", nil)
		require.NoError(t, err, "Failed to add user message 2")

		// Store messages in graph
		for _, message := range conversation.Messages {
			msg := message // Capture loop variable
			err = repo.AddMessage(ctx, "conv-123", &msg)
			require.NoError(t, err, "Failed to add message to graph")
		}

		// Query messages by role
		userMessages, err := repo.GetMessagesByRole(ctx, "conv-123", domain.MessageRoleUser)
		assert.NoError(t, err, "GetMessagesByRole should succeed for user")
		assert.Len(t, userMessages, 2, "Should find 2 user messages")

		assistantMessages, err := repo.GetMessagesByRole(ctx, "conv-123", domain.MessageRoleAssistant)
		assert.NoError(t, err, "GetMessagesByRole should succeed for assistant")
		assert.Len(t, assistantMessages, 1, "Should find 1 assistant message")
	})
}
