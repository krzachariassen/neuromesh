package bff

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conversationApp "neuromesh/internal/conversation/application"
	conversationInfra "neuromesh/internal/conversation/infrastructure"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	projectApp "neuromesh/internal/project/application"
	projectInfra "neuromesh/internal/project/infrastructure"
	userApp "neuromesh/internal/user/application"
	userDomain "neuromesh/internal/user/domain"
	userInfra "neuromesh/internal/user/infrastructure"
)

// TestBFFServiceConversationRelationships tests the actual BFF service conversation creation
func TestBFFServiceConversationRelationships(t *testing.T) {
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

	// Create repositories
	conversationRepo := conversationInfra.NewGraphConversationRepository(g)
	userRepo := userInfra.NewGraphUserRepository(g)
	projectRepo := projectInfra.NewGraphProjectRepository(g)

	// Create services
	conversationService := conversationApp.NewConversationService(conversationRepo)
	userService := userApp.NewUserService(userRepo)
	projectService := projectApp.NewProjectService(projectRepo)

	// Create BFF service (like the real system)
	bffService := NewService(
		nil, // orchestrator not needed for this test
		conversationService,
		userService,
		projectService,
		g, // graph needed for BFF service
		logger,
	)

	// Clean up any existing test data
	err = g.ClearTestData(ctx)
	require.NoError(t, err, "Failed to clean up test data")

	// Ensure schemas exist
	err = conversationRepo.EnsureConversationSchema(ctx)
	require.NoError(t, err, "Failed to ensure conversation schema")
	err = userRepo.EnsureUserSchema(ctx)
	require.NoError(t, err, "Failed to ensure user schema")
	err = projectRepo.EnsureProjectSchema(ctx)
	require.NoError(t, err, "Failed to ensure project schema")

	// Create a test project first
	testProject, err := projectService.CreateProject(ctx, "test-project-123", "BFF Test Project", "test@neuromesh.ai")
	require.NoError(t, err, "Failed to create test project")

	// Test conversation creation through BFF service (like real API)
	// Create test request with session and user IDs
	testSessionID := "test-session-123"
	testUserID := "test-user-123"

	// First, ensure User and Session nodes exist (this is what ProcessMessage does)
	err = userRepo.EnsureUserSchema(ctx)
	require.NoError(t, err, "Should create user schema")

	// Create user domain object and save
	user, err := userDomain.NewUser(testUserID, testSessionID, userDomain.UserTypeWebSession)
	require.NoError(t, err, "Should create user domain object")
	err = userRepo.CreateUser(ctx, user)
	require.NoError(t, err, "Should create user")

	// Create session domain object and save
	session, err := userDomain.NewSession(testSessionID, testUserID, time.Hour*24)
	require.NoError(t, err, "Should create session domain object")
	err = userRepo.CreateSession(ctx, session)
	require.NoError(t, err, "Should create session")

	request := &ChatRequest{
		Message:   "Hello, world!",
		ProjectID: testProject.ID,
		SessionID: testSessionID,
		UserID:    testUserID,
		// Don't set ConversationID to test auto-generation
	}

	conversation, err := bffService.ensureConversation(ctx, request)
	require.NoError(t, err, "BFF service should create conversation successfully")
	require.NotNil(t, conversation, "Conversation should not be nil")

	// Get conversation context to verify relationships were created
	conversationContext, err := bffService.GetConversationContext(ctx, conversation.ID)
	require.NoError(t, err, "Should be able to get conversation context")
	require.NotNil(t, conversationContext, "Conversation context should not be nil")

	// Should have the expected IDs from our request
	require.Equal(t, testSessionID, conversationContext.SessionID, "Session ID should match")
	require.Equal(t, testUserID, conversationContext.UserID, "User ID should match")

	t.Logf("Created conversation: %s, session: %s, user: %s, project: %s",
		conversation.ID, testSessionID, testUserID, testProject.ID)

	// Now verify relationships exist in Neo4j using direct queries
	t.Run("UserConversationRelationship", func(t *testing.T) {
		edges, err := g.GetEdgesWithTargets(ctx, "User", testUserID)
		require.NoError(t, err, "Should be able to get user edges")

		t.Logf("User edges: %+v", edges)

		found := false
		for _, edge := range edges {
			if edgeType, ok := edge["type"].(string); ok && edgeType == "PARTICIPANT_IN" {
				if targetID, ok := edge["target_id"].(string); ok && targetID == conversation.ID {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "User should have PARTICIPANT_IN relationship to conversation")
	})

	t.Run("SessionConversationRelationship", func(t *testing.T) {
		edges, err := g.GetEdgesWithTargets(ctx, "Session", testSessionID)
		require.NoError(t, err, "Should be able to get session edges")

		t.Logf("Session edges: %+v", edges)

		found := false
		for _, edge := range edges {
			if edgeType, ok := edge["type"].(string); ok && edgeType == "IN_SESSION" {
				if targetID, ok := edge["target_id"].(string); ok && targetID == conversation.ID {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Session should have IN_SESSION relationship to conversation")
	})

	t.Run("ConversationProjectRelationship", func(t *testing.T) {
		edges, err := g.GetEdgesWithTargets(ctx, "Conversation", conversation.ID)
		require.NoError(t, err, "Should be able to get conversation edges")

		t.Logf("Conversation edges: %+v", edges)

		found := false
		for _, edge := range edges {
			if edgeType, ok := edge["type"].(string); ok && edgeType == "BELONGS_TO" {
				if targetID, ok := edge["target_id"].(string); ok && targetID == testProject.ID {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Conversation should have BELONGS_TO relationship to correct project")
	})
}
