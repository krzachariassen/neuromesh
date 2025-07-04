package infrastructure

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	planningdomain "neuromesh/internal/planning/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraphConversationRepository_ConversationMessageSchema tests Conversation and Message schema creation
func TestGraphConversationRepository_ConversationMessageSchema(t *testing.T) {
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

	t.Run("GREEN: should create Conversation schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Now this should succeed
		err = repo.EnsureConversationSchema(ctx)
		assert.NoError(t, err, "EnsureConversationSchema should succeed")
	})

	t.Run("GREEN: should create Message schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Now this should succeed
		err = repo.EnsureMessageSchema(ctx)
		assert.NoError(t, err, "EnsureMessageSchema should succeed")
	})

	t.Run("GREEN: should create and store Conversation nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123", "user-456")
		require.NoError(t, err, "Failed to create conversation")

		// Now this should succeed
		err = repo.CreateConversation(ctx, conversation)
		assert.NoError(t, err, "CreateConversation should succeed")

		// Verify the conversation was created by retrieving it
		retrievedConversation, err := repo.GetConversationWithMessages(ctx, "conv-123")
		assert.NoError(t, err, "Should be able to retrieve created conversation")
		assert.NotNil(t, retrievedConversation, "Retrieved conversation should not be nil")
		assert.Equal(t, conversation.ID, retrievedConversation.ID, "Conversation ID should match")
		assert.Equal(t, conversation.UserID, retrievedConversation.UserID, "User ID should match")
		assert.Equal(t, conversation.Status, retrievedConversation.Status, "Status should match")
	})

	t.Run("GREEN: should create and store Message nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureMessageSchema(ctx)
		require.NoError(t, err, "Failed to ensure message schema")

		// Create test message
		message := &domain.ConversationMessage{
			ID:        "msg-123",
			Role:      domain.MessageRoleUser,
			Content:   "Hello, world!",
			Timestamp: time.Now().UTC(),
			Metadata:  make(map[string]interface{}),
		}

		// Now this should succeed
		err = repo.CreateMessage(ctx, "conv-123", message)
		assert.NoError(t, err, "CreateMessage should succeed")
	})

	t.Run("GREEN: should establish Conversation-Message relationships", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schemas exist first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")
		err = repo.EnsureMessageSchema(ctx)
		require.NoError(t, err, "Failed to ensure message schema")

		// Create test conversation and message
		conversation, err := domain.NewConversation("conv-123", "user-456")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		message := &domain.ConversationMessage{
			ID:        "msg-123",
			Role:      domain.MessageRoleUser,
			Content:   "Hello, world!",
			Timestamp: time.Now().UTC(),
			Metadata:  make(map[string]interface{}),
		}
		err = repo.CreateMessage(ctx, "conv-123", message)
		require.NoError(t, err, "Failed to create message")

		// Now this should succeed
		err = repo.LinkMessageToConversation(ctx, "msg-123", "conv-123")
		assert.NoError(t, err, "LinkMessageToConversation should succeed")
	})

	t.Run("GREEN: should establish User-Conversation relationships", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123", "user-456")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Now this should succeed
		err = repo.LinkConversationToUser(ctx, "conv-123", "user-456")
		assert.NoError(t, err, "LinkConversationToUser should succeed")
	})

	t.Run("GREEN: should query Conversation with messages", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversation
		conversation, err := domain.NewConversation("conv-123", "user-456")
		require.NoError(t, err, "Failed to create conversation")
		err = repo.CreateConversation(ctx, conversation)
		require.NoError(t, err, "Failed to create conversation")

		// Now this should succeed
		retrievedConversation, err := repo.GetConversationWithMessages(ctx, "conv-123")
		assert.NoError(t, err, "GetConversationWithMessages should succeed")
		assert.NotNil(t, retrievedConversation, "Conversation should not be nil")
		assert.Equal(t, "conv-123", retrievedConversation.ID, "Conversation ID should match")
		assert.Equal(t, "user-456", retrievedConversation.UserID, "User ID should match")
	})

	t.Run("GREEN: should query User conversations", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureConversationSchema(ctx)
		require.NoError(t, err, "Failed to ensure conversation schema")

		// Create test conversations for the same user
		conversation1, err := domain.NewConversation("conv-123", "user-456")
		require.NoError(t, err, "Failed to create conversation 1")
		err = repo.CreateConversation(ctx, conversation1)
		require.NoError(t, err, "Failed to create conversation 1")

		conversation2, err := domain.NewConversation("conv-456", "user-456")
		require.NoError(t, err, "Failed to create conversation 2")
		err = repo.CreateConversation(ctx, conversation2)
		require.NoError(t, err, "Failed to create conversation 2")

		// Now this should succeed
		conversations, err := repo.GetUserConversations(ctx, "user-456")
		assert.NoError(t, err, "GetUserConversations should succeed")
		assert.NotNil(t, conversations, "Conversations should not be nil")
		assert.Len(t, conversations, 2, "Should return 2 conversations for the user")
	})
}

// TestGraphConversationRepository_UserRequestAIDecisionSchema tests UserRequest and AIDecision schema creation (GREEN Phase)
func TestGraphConversationRepository_UserRequestAIDecisionSchema(t *testing.T) {
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

	t.Run("GREEN: should create UserRequest schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// This should succeed now that EnsureUserRequestSchema is implemented
		err = repo.EnsureUserRequestSchema(ctx)
		assert.NoError(t, err, "EnsureUserRequestSchema should succeed")
	})

	t.Run("GREEN: should create AIDecision schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// This should succeed now that EnsureAIDecisionSchema is implemented
		err = repo.EnsureAIDecisionSchema(ctx)
		assert.NoError(t, err, "EnsureAIDecisionSchema should succeed")
	})

	t.Run("GREEN: should create and store UserRequest nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema first
		err = repo.EnsureUserRequestSchema(ctx)
		require.NoError(t, err, "Failed to ensure schema")

		// Create test user request
		userRequest, err := domain.NewUserRequest("req-123", "user-456", "session-789", "Count words in this text")
		require.NoError(t, err, "Failed to create user request")

		// This should succeed now that CreateUserRequest is implemented
		err = repo.CreateUserRequest(ctx, userRequest)
		assert.NoError(t, err, "CreateUserRequest should succeed")
	})

	t.Run("GREEN: should create and store AIDecision nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema first
		err = repo.EnsureAIDecisionSchema(ctx)
		require.NoError(t, err, "Failed to ensure schema")

		// Create test AI decision
		aiDecision, err := planningdomain.NewAIDecision("decision-123", "req-456", planningdomain.DecisionTypeExecute, "Execute word count agent", 0.95)
		require.NoError(t, err, "Failed to create AI decision")

		// This should succeed now that CreateAIDecision is implemented
		err = repo.CreateAIDecision(ctx, aiDecision)
		assert.NoError(t, err, "CreateAIDecision should succeed")
	})

	t.Run("GREEN: should establish UserRequest-AIDecision relationships", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema first
		err = repo.EnsureUserRequestSchema(ctx)
		require.NoError(t, err, "Failed to ensure user request schema")
		err = repo.EnsureAIDecisionSchema(ctx)
		require.NoError(t, err, "Failed to ensure AI decision schema")

		// Create test nodes first
		userRequest, err := domain.NewUserRequest("req-456", "user-789", "session-123", "Count words in this text")
		require.NoError(t, err, "Failed to create user request")
		err = repo.CreateUserRequest(ctx, userRequest)
		require.NoError(t, err, "Failed to create user request node")

		aiDecision, err := domain.NewAIDecision("decision-123", "req-456", domain.DecisionTypeExecute, "Execute word count agent", 0.95)
		require.NoError(t, err, "Failed to create AI decision")
		err = repo.CreateAIDecision(ctx, aiDecision)
		require.NoError(t, err, "Failed to create AI decision node")

		// This should succeed now that LinkUserRequestToAIDecision is implemented
		err = repo.LinkUserRequestToAIDecision(ctx, "req-456", "decision-123")
		assert.NoError(t, err, "LinkUserRequestToAIDecision should succeed")
	})

	t.Run("GREEN: should query UserRequest with AIDecisions", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema first
		err = repo.EnsureUserRequestSchema(ctx)
		require.NoError(t, err, "Failed to ensure schema")

		// Create test user request
		userRequest, err := domain.NewUserRequest("req-123", "user-456", "session-789", "Count words in this text")
		require.NoError(t, err, "Failed to create user request")
		err = repo.CreateUserRequest(ctx, userRequest)
		require.NoError(t, err, "Failed to create user request node")

		// This should succeed now that GetUserRequestWithDecisions is implemented
		retrievedUserRequest, err := repo.GetUserRequestWithDecisions(ctx, "req-123")
		assert.NoError(t, err, "GetUserRequestWithDecisions should succeed")
		assert.NotNil(t, retrievedUserRequest, "UserRequest should not be nil")
		assert.Equal(t, "req-123", retrievedUserRequest.ID, "UserRequest ID should match")
	})
}
