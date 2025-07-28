package main

import (
	"context"
	"os"
	"testing"
	"time"

	aiInfrastructure "neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	userDomain "neuromesh/internal/user/domain"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/require"
)

// TestEndToEndCrossDomainRelationships tests the complete flow
// RED: This test should expose whether cross-domain relationships are actually created
func TestEndToEndCrossDomainRelationships(t *testing.T) {
	// Skip if running unit tests only
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Initialize logger
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	// Create context
	ctx := context.Background()

	// Create graph config
	graphConfig := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      getEnvOrDefault("NEO4J_URL", "bolt://localhost:7687"),
		Neo4jUser:     getEnvOrDefault("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnvOrDefault("NEO4J_PASSWORD", "testpass"),
	}

	// Create graph instance
	graphInstance, err := graph.NewNeo4jGraph(ctx, graphConfig, logger)
	require.NoError(t, err, "Failed to connect to Neo4j")
	defer graphInstance.Close(ctx)

	// Clean up any existing test data
	session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `MATCH (n) WHERE n.id STARTS WITH 'e2e-test-' DETACH DELETE n`, nil)
		return nil, err
	})
	session.Close(ctx)
	require.NoError(t, err, "Failed to clean test data")

	// Create RabbitMQ message bus for testing (use memory bus)
	messageBusConfig := messaging.RabbitMQConfig{
		URL:            getEnvOrDefault("RABBITMQ_URL", "amqp://orchestrator:orchestrator123@localhost:5672/"),
		ReconnectDelay: 1 * time.Second,
		MaxReconnects:  2,
		Heartbeat:      5 * time.Second,
	}
	rabbitMQBus := messaging.NewRabbitMQMessageBus(messageBusConfig, logger)

	// Try to connect to RabbitMQ, fall back to memory bus if it fails
	var messageBus messaging.MessageBus
	if err := rabbitMQBus.Connect(ctx); err != nil {
		t.Logf("RabbitMQ not available, using memory bus: %v", err)
		// For testing, create a memory bus instead
		messageBus = messaging.NewMemoryMessageBus(logger)
	} else {
		messageBus = rabbitMQBus
		defer rabbitMQBus.Close()
	}

	// Create AI provider
	aiConfig := aiInfrastructure.DefaultOpenAIConfig()
	aiConfig.APIKey = getEnvOrDefault("OPENAI_API_KEY", "test-key")
	aiProvider := aiInfrastructure.NewOpenAIProvider(aiConfig, logger)

	// Create service factory
	serviceFactory := orchestratorApp.NewServiceFactory(logger, graphInstance, messageBus, aiProvider)
	defer serviceFactory.Shutdown()

	// Start services
	err = serviceFactory.StartServices(ctx)
	require.NoError(t, err, "Failed to start services")

	// Get services from factory
	orchestratorService := serviceFactory.CreateOrchestratorService()
	conversationService := serviceFactory.GetConversationService()
	userService := serviceFactory.GetUserService()

	// Test unique session ID
	sessionID := "e2e-test-session-" + time.Now().Format("20060102-150405")

	t.Run("should create conversation and execution plan with relationship", func(t *testing.T) {
		// Create a test user (using the same logic as BFF)
		userID := "e2e-test-user-" + sessionID
		user, err := userService.CreateUser(ctx, userID, sessionID, userDomain.UserTypeWebSession)
		require.NoError(t, err, "Failed to create user")

		// Create conversation
		conversationID := "e2e-test-conv-" + sessionID
		_, err = conversationService.CreateConversation(ctx, conversationID, sessionID, user.ID)
		require.NoError(t, err, "Failed to create conversation")

		// Process through orchestrator directly using the new interface
		orchestratorRequest := &orchestratorApp.OrchestratorRequest{
			UserInput:      "I need to create a comprehensive sentiment analysis test plan using Python, NLTK library, with focus on social media data processing. The plan should include data collection, preprocessing, model training, and evaluation metrics like accuracy and F1-score.",
			UserID:         user.ID,
			SessionID:      sessionID,
			MessageID:      "e2e-test-msg-" + sessionID,
			ConversationID: conversationID, // Pass conversation ID for cross-domain linking
		}

		response, err := orchestratorService.ProcessUserRequest(ctx, orchestratorRequest)
		require.NoError(t, err, "ProcessUserRequest should succeed")
		require.NotNil(t, response, "Response should not be nil")
		require.True(t, response.Success, "Response should be successful")
		require.NotEmpty(t, response.Message, "Response should have content")

		t.Logf("Orchestrator Response: %s", response.Message)

		// Give the system a moment to persist relationships
		time.Sleep(200 * time.Millisecond)

		// Query Neo4j to verify the data was created
		session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
		defer session.Close(ctx)

		// Check for conversation
		conversationResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			query := `MATCH (c:Conversation) WHERE c.session_id = $sessionID RETURN count(c) as count`
			records, err := tx.Run(ctx, query, map[string]interface{}{"sessionID": sessionID})
			if err != nil {
				return 0, err
			}

			if records.Next(ctx) {
				return records.Record().Values[0].(int64), nil
			}
			return int64(0), nil
		})
		require.NoError(t, err, "Failed to query conversations")

		conversationCount := conversationResult.(int64)
		t.Logf("Found %d conversations for session %s", conversationCount, sessionID)
		require.Greater(t, conversationCount, int64(0), "Should have created at least one conversation")

		// Check for execution plan
		planResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			query := `MATCH (p:execution_plan) RETURN count(p) as count`
			records, err := tx.Run(ctx, query, nil)
			if err != nil {
				return 0, err
			}

			if records.Next(ctx) {
				return records.Record().Values[0].(int64), nil
			}
			return int64(0), nil
		})
		require.NoError(t, err, "Failed to query execution plans")

		planCount := planResult.(int64)
		t.Logf("Found %d execution plans in database", planCount)
		require.Greater(t, planCount, int64(0), "Should have created at least one execution plan")

		// Check for cross-domain relationship (the critical test!)
		relationshipResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			query := `
				MATCH (c:Conversation)-[r:LINKED_TO_PLAN]->(p:execution_plan)
				WHERE c.session_id = $sessionID
				RETURN count(r) as count
			`
			records, err := tx.Run(ctx, query, map[string]interface{}{"sessionID": sessionID})
			if err != nil {
				return 0, err
			}

			if records.Next(ctx) {
				return records.Record().Values[0].(int64), nil
			}
			return int64(0), nil
		})
		require.NoError(t, err, "Failed to query relationships")

		relationshipCount := relationshipResult.(int64)
		t.Logf("Found %d LINKED_TO_PLAN relationships", relationshipCount)

		// This is the moment of truth - does our orchestrator actually create the relationship?
		require.Greater(t, relationshipCount, int64(0),
			"Should have LINKED_TO_PLAN relationship between conversation and execution plan")

		t.Logf("✅ End-to-end test PASSED: Cross-domain relationships are working!")
	})
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
