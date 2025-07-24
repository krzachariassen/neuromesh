package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/planning/domain"
)

// RED: Write failing tests for Analysis graph persistence
// These tests will fail until we implement the GraphAnalysisRepository

func TestGraphAnalysisRepository_Store(t *testing.T) {
	t.Run("RED: should store analysis in Neo4j with proper relationships", func(t *testing.T) {
		// Setup real Neo4j for integration testing
		graph, cleanup := setupTestNeo4j(t)
		defer cleanup()

		repo := NewGraphAnalysisRepository(graph)

		// Create test analysis
		requestID := "test-message-123"
		analysis := domain.NewAnalysis(
			requestID,
			"deploy_application",
			"deployment",
			85,
			[]string{"deploy-agent", "test-agent"},
			"User wants to deploy their application to production",
		)

		// Store analysis - THIS WILL FAIL until we implement the repository
		err := repo.Store(context.Background(), analysis)

		// Assertions - these define what we expect from the implementation
		require.NoError(t, err, "Should store analysis without error")

		// Verify analysis was stored in Neo4j
		storedAnalysis, err := repo.GetByID(context.Background(), analysis.ID)
		require.NoError(t, err)
		assert.Equal(t, analysis.ID, storedAnalysis.ID)
		assert.Equal(t, analysis.RequestID, storedAnalysis.RequestID)
		assert.Equal(t, analysis.Intent, storedAnalysis.Intent)
		assert.Equal(t, analysis.Category, storedAnalysis.Category)
		assert.Equal(t, analysis.Confidence, storedAnalysis.Confidence)
		assert.Equal(t, analysis.RequiredAgents, storedAnalysis.RequiredAgents)
		assert.Equal(t, analysis.Reasoning, storedAnalysis.Reasoning)
		assert.WithinDuration(t, analysis.Timestamp, storedAnalysis.Timestamp, time.Second)
	})

	t.Run("RED: should create relationships to User and Conversation nodes", func(t *testing.T) {
		// Setup real Neo4j
		graph, cleanup := setupTestNeo4j(t)
		defer cleanup()

		repo := NewGraphAnalysisRepository(graph)

		// Create user and conversation first (following the existing pattern)
		userID := "test-user-123"
		sessionID := "test-session-123"
		conversationID := "test-conversation-123"
		messageID := "test-message-123"

		// Pre-create user, session, conversation, and message nodes
		// (This follows the existing conversation flow)
		createTestUser(t, graph, userID, sessionID)
		createTestConversation(t, graph, userID, sessionID, conversationID)
		createTestMessage(t, graph, conversationID, messageID, "user", "Deploy my app")

		// Create analysis linked to the message
		analysis := domain.NewAnalysis(
			messageID, // requestID = messageID
			"deploy_application",
			"deployment",
			90,
			[]string{"deploy-agent"},
			"Clear deployment request",
		)

		// Store analysis - THIS WILL FAIL until implemented
		err := repo.Store(context.Background(), analysis)
		require.NoError(t, err)

		// Verify relationships exist in graph
		// Should be able to traverse: User -> Session -> Conversation -> Message -> Analysis
		relationships := queryGraphRelationships(t, graph,
			"MATCH (u:User {id: $userID})-[:HAS_SESSION]->(s:Session)-[:HAS_CONVERSATION]->(c:Conversation)-[:CONTAINS_MESSAGE]->(m:Message)-[:TRIGGERS_ANALYSIS]->(a:Analysis {id: $analysisID}) RETURN count(*) as count",
			map[string]interface{}{
				"userID":     userID,
				"analysisID": analysis.ID,
			})

		assert.Equal(t, 1, relationships["count"], "Should have complete relationship chain from User to Analysis")
	})
}

func TestGraphAnalysisRepository_GetByRequestID(t *testing.T) {
	t.Run("RED: should retrieve analysis by request ID", func(t *testing.T) {
		graph, cleanup := setupTestNeo4j(t)
		defer cleanup()

		repo := NewGraphAnalysisRepository(graph)

		// Store analysis first
		requestID := "test-message-456"
		analysis := domain.NewAnalysis(
			requestID,
			"security_scan",
			"security",
			75,
			[]string{"security-agent"},
			"User wants security analysis",
		)

		err := repo.Store(context.Background(), analysis)
		require.NoError(t, err)

		// Retrieve by request ID - THIS WILL FAIL until implemented
		retrieved, err := repo.GetByRequestID(context.Background(), requestID)
		require.NoError(t, err)
		assert.Equal(t, analysis.ID, retrieved.ID)
		assert.Equal(t, requestID, retrieved.RequestID)
	})
}

func TestGraphAnalysisRepository_GetByUserID(t *testing.T) {
	t.Run("RED: should retrieve all analyses for a user ordered by timestamp", func(t *testing.T) {
		graph, cleanup := setupTestNeo4j(t)
		defer cleanup()

		repo := NewGraphAnalysisRepository(graph)

		// Create test user and multiple analyses
		userID := "test-user-789"
		sessionID := "test-session-789"
		conversationID := "test-conversation-789"

		createTestUser(t, graph, userID, sessionID)
		createTestConversation(t, graph, userID, sessionID, conversationID)

		// Create multiple analyses for the same user (different messages)
		analyses := []*domain.Analysis{
			domain.NewAnalysis("msg-1", "deploy", "deployment", 80, []string{"deploy-agent"}, "First request"),
			domain.NewAnalysis("msg-2", "monitor", "monitoring", 75, []string{"monitor-agent"}, "Second request"),
			domain.NewAnalysis("msg-3", "backup", "backup", 90, []string{"backup-agent"}, "Third request"),
		}

		// Store all analyses
		for _, analysis := range analyses {
			createTestMessage(t, graph, conversationID, analysis.RequestID, "user", "Test message")
			err := repo.Store(context.Background(), analysis)
			require.NoError(t, err)
		}

		// Retrieve analyses by user ID - THIS WILL FAIL until implemented
		userAnalyses, err := repo.GetByUserID(context.Background(), userID, 10)
		require.NoError(t, err)
		assert.Len(t, userAnalyses, 3)

		// Should be ordered by timestamp (newest first)
		for i := 1; i < len(userAnalyses); i++ {
			assert.True(t, userAnalyses[i-1].Timestamp.After(userAnalyses[i].Timestamp) ||
				userAnalyses[i-1].Timestamp.Equal(userAnalyses[i].Timestamp))
		}
	})
}

func TestGraphAnalysisRepository_GetByConfidenceRange(t *testing.T) {
	t.Run("RED: should retrieve analyses within confidence range", func(t *testing.T) {
		graph, cleanup := setupTestNeo4j(t)
		defer cleanup()

		repo := NewGraphAnalysisRepository(graph)

		// Create analyses with different confidence levels
		analyses := []*domain.Analysis{
			domain.NewAnalysis("msg-low", "unclear", "general", 30, []string{}, "Low confidence"),
			domain.NewAnalysis("msg-medium", "maybe_deploy", "deployment", 75, []string{"deploy-agent"}, "Medium confidence"),
			domain.NewAnalysis("msg-high", "deploy", "deployment", 95, []string{"deploy-agent"}, "High confidence"),
		}

		for _, analysis := range analyses {
			err := repo.Store(context.Background(), analysis)
			require.NoError(t, err)
		}

		// Query for high confidence analyses (80-100) - THIS WILL FAIL until implemented
		highConfidenceAnalyses, err := repo.GetByConfidenceRange(context.Background(), 80, 100, 10)
		require.NoError(t, err)
		assert.Len(t, highConfidenceAnalyses, 1)
		assert.Equal(t, "deploy", highConfidenceAnalyses[0].Intent)
		assert.Equal(t, 95, highConfidenceAnalyses[0].Confidence)
	})
}

func TestGraphAnalysisRepository_GetByCategory(t *testing.T) {
	t.Run("RED: should retrieve analyses by category", func(t *testing.T) {
		graph, cleanup := setupTestNeo4j(t)
		defer cleanup()

		repo := NewGraphAnalysisRepository(graph)

		// Create analyses in different categories
		analyses := []*domain.Analysis{
			domain.NewAnalysis("msg-deploy1", "deploy_app", "deployment", 85, []string{"deploy-agent"}, "Deploy request 1"),
			domain.NewAnalysis("msg-deploy2", "deploy_service", "deployment", 90, []string{"deploy-agent"}, "Deploy request 2"),
			domain.NewAnalysis("msg-monitor", "check_health", "monitoring", 80, []string{"monitor-agent"}, "Monitor request"),
		}

		for _, analysis := range analyses {
			err := repo.Store(context.Background(), analysis)
			require.NoError(t, err)
		}

		// Query for deployment category - THIS WILL FAIL until implemented
		deploymentAnalyses, err := repo.GetByCategory(context.Background(), "deployment", 10)
		require.NoError(t, err)
		assert.Len(t, deploymentAnalyses, 2)

		for _, analysis := range deploymentAnalyses {
			assert.Equal(t, "deployment", analysis.Category)
		}
	})
}
