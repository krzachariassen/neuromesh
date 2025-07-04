package infrastructure

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	"neuromesh/internal/user/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraphUserRepository_UserSchema tests User and Session schema creation
func TestGraphUserRepository_UserSchema(t *testing.T) {
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
	repo := NewGraphUserRepository(g)

	t.Run("GREEN: should create User schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Now this should succeed
		err = repo.EnsureUserSchema(ctx)
		assert.NoError(t, err, "EnsureUserSchema should succeed")
	})

	t.Run("GREEN: should create Session schema constraints and indexes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Now this should succeed
		err = repo.EnsureSessionSchema(ctx)
		assert.NoError(t, err, "EnsureSessionSchema should succeed")
	})

	t.Run("GREEN: should create and store User nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureUserSchema(ctx)
		require.NoError(t, err, "Failed to ensure user schema")

		// Create test user
		user, err := domain.NewUser("user-123", "session-456", domain.UserTypeWebSession)
		require.NoError(t, err, "Failed to create user")

		// Now this should succeed
		err = repo.CreateUser(ctx, user)
		assert.NoError(t, err, "CreateUser should succeed")

		// Verify the user was created by retrieving it
		retrievedUser, err := repo.GetUser(ctx, "user-123")
		assert.NoError(t, err, "Should be able to retrieve created user")
		assert.NotNil(t, retrievedUser, "Retrieved user should not be nil")
		assert.Equal(t, user.ID, retrievedUser.ID, "User ID should match")
		assert.Equal(t, user.SessionID, retrievedUser.SessionID, "Session ID should match")
		assert.Equal(t, user.UserType, retrievedUser.UserType, "User type should match")
		assert.Equal(t, user.Status, retrievedUser.Status, "User status should match")
	})

	t.Run("GREEN: should create and store Session nodes", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schema exists first
		err = repo.EnsureSessionSchema(ctx)
		require.NoError(t, err, "Failed to ensure session schema")

		// Create test session
		session, err := domain.NewSession("session-456", "user-123", 24*time.Hour)
		require.NoError(t, err, "Failed to create session")

		// Now this should succeed
		err = repo.CreateSession(ctx, session)
		assert.NoError(t, err, "CreateSession should succeed")

		// Verify the session was created by retrieving it
		retrievedSession, err := repo.GetSession(ctx, "session-456")
		assert.NoError(t, err, "Should be able to retrieve created session")
		assert.NotNil(t, retrievedSession, "Retrieved session should not be nil")
		assert.Equal(t, session.ID, retrievedSession.ID, "Session ID should match")
		assert.Equal(t, session.UserID, retrievedSession.UserID, "User ID should match")
		assert.Equal(t, session.Status, retrievedSession.Status, "Session status should match")
	})

	t.Run("GREEN: should establish User-Session relationships", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schemas exist first
		err = repo.EnsureUserSchema(ctx)
		require.NoError(t, err, "Failed to ensure user schema")
		err = repo.EnsureSessionSchema(ctx)
		require.NoError(t, err, "Failed to ensure session schema")

		// Create test user and session
		user, err := domain.NewUser("user-123", "session-456", domain.UserTypeWebSession)
		require.NoError(t, err, "Failed to create user")
		err = repo.CreateUser(ctx, user)
		require.NoError(t, err, "Failed to create user")

		session, err := domain.NewSession("session-456", "user-123", 24*time.Hour)
		require.NoError(t, err, "Failed to create session")
		err = repo.CreateSession(ctx, session)
		require.NoError(t, err, "Failed to create session")

		// Now this should succeed
		err = repo.LinkUserToSession(ctx, "user-123", "session-456")
		assert.NoError(t, err, "LinkUserToSession should succeed")
	})

	t.Run("GREEN: should query User with relationships", func(t *testing.T) {
		// Clean up any existing test data
		err := g.ClearTestData(ctx)
		require.NoError(t, err, "Failed to clean up test data")

		// Ensure schemas exist first
		err = repo.EnsureUserSchema(ctx)
		require.NoError(t, err, "Failed to ensure user schema")

		// Create test user
		user, err := domain.NewUser("user-123", "session-456", domain.UserTypeWebSession)
		require.NoError(t, err, "Failed to create user")
		err = repo.CreateUser(ctx, user)
		require.NoError(t, err, "Failed to create user")

		// Now this should succeed
		retrievedUser, err := repo.GetUserWithSessions(ctx, "user-123")
		assert.NoError(t, err, "GetUserWithSessions should succeed")
		assert.NotNil(t, retrievedUser, "User should not be nil")
		assert.Equal(t, "user-123", retrievedUser.ID, "User ID should match")
		assert.Equal(t, "session-456", retrievedUser.SessionID, "Session ID should match")
	})
}
