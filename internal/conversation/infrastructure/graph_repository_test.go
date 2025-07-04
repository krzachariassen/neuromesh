package infrastructure

import (
	"context"
	"testing"

	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	planningdomain "neuromesh/internal/planning/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("GREEN: should create Conversation schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// This should succeed
		err = repo.EnsureConversationSchema(ctx)
		assert.NoError(t, err, "EnsureConversationSchema should succeed")
	})

	t.Run("GREEN: should create Message schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// This should succeed
		err = repo.EnsureMessageSchema(ctx)
		assert.NoError(t, err, "EnsureMessageSchema should succeed")
	})
}

// TestGraphConversationRepository_UserRequestSchema tests UserRequest and AIDecision schema creation (RED Phase)
func TestGraphConversationRepository_UserRequestSchema(t *testing.T) {
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
		userRequest, err := domain.NewUserRequest("req-123", "user-456", "session-789", "Hello, world!")
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
		aiDecision, err := planningdomain.NewAIDecision("decision-123", "req-456", planningdomain.DecisionTypeExecute, "use text processor", 0.95)
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
		userRequest, err := domain.NewUserRequest("req-123", "user-456", "session-789", "Hello, world!")
		require.NoError(t, err, "Failed to create user request")
		err = repo.CreateUserRequest(ctx, userRequest)
		require.NoError(t, err, "Failed to create user request node")

		aiDecision, err := planningdomain.NewAIDecision("decision-456", "req-123", planningdomain.DecisionTypeExecute, "use text processor", 0.95)
		require.NoError(t, err, "Failed to create AI decision")
		err = repo.CreateAIDecision(ctx, aiDecision)
		require.NoError(t, err, "Failed to create AI decision node")

		// This should succeed now that LinkUserRequestToAIDecision is implemented
		err = repo.LinkUserRequestToAIDecision(ctx, "req-123", "decision-456")
		assert.NoError(t, err, "LinkUserRequestToAIDecision should succeed")
	})

	t.Run("GREEN: should query UserRequest with decisions", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema first
		err = repo.EnsureUserRequestSchema(ctx)
		require.NoError(t, err, "Failed to ensure user request schema")

		// Create test user request
		userRequest, err := domain.NewUserRequest("req-123", "user-456", "session-789", "Hello, world!")
		require.NoError(t, err, "Failed to create user request")
		err = repo.CreateUserRequest(ctx, userRequest)
		require.NoError(t, err, "Failed to create user request node")

		// This should succeed now that GetUserRequestWithDecisions is implemented
		retrievedUserRequest, err := repo.GetUserRequestWithDecisions(ctx, "req-123")
		assert.NoError(t, err, "GetUserRequestWithDecisions should succeed")
		assert.NotNil(t, retrievedUserRequest, "Retrieved UserRequest should not be nil")
		assert.Equal(t, "req-123", retrievedUserRequest.ID, "UserRequest ID should match")
	})
}
