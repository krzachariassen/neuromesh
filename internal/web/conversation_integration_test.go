package web

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
	conversationInfra "neuromesh/internal/conversation/infrastructure"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	orchestratorDomain "neuromesh/internal/orchestrator/domain"
	planningDomain "neuromesh/internal/planning/domain"
	userApp "neuromesh/internal/user/application"
	userDomain "neuromesh/internal/user/domain"
	userInfra "neuromesh/internal/user/infrastructure"
)

// TestConversationAwareWebBFFWithGraph tests ConversationAwareWebBFF with actual graph backend
func TestConversationAwareWebBFFWithGraph(t *testing.T) {
	logger := logging.NewStructuredLogger(logging.LevelDebug)
	ctx := context.Background()

	// Use Neo4j for testing
	graphConfig := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}
	testGraph, err := graph.NewNeo4jGraph(ctx, graphConfig, logger)
	if err != nil {
		t.Skip("Neo4j not available, skipping integration test")
		return
	}
	defer testGraph.Close(ctx)

	// Create repositories and services
	userRepo := userInfra.NewGraphUserRepository(testGraph)
	conversationRepo := conversationInfra.NewGraphConversationRepository(testGraph)
	userService := userApp.NewUserService(userRepo)
	conversationService := conversationApp.NewConversationService(conversationRepo)

	// Create mock orchestrator with realistic responses
	mockOrchestrator := &MockOrchestratorImpl{
		responses: map[string]*orchestratorApp.OrchestratorResult{
			"Hello": {
				Message: "Hi there! How can I help you today?",
				Success: true,
				Analysis: &planningDomain.Analysis{
					Intent:     "greeting",
					Confidence: 95,
					Category:   "social",
				},
				Decision: &orchestratorDomain.Decision{
					Type:      orchestratorDomain.DecisionTypeClarify,
					Reasoning: "Simple greeting from user",
				},
			},
			"What can you do?": {
				Message: "I can help you with various tasks. Let me know what you need!",
				Success: true,
				Analysis: &planningDomain.Analysis{
					Intent:     "capability_inquiry",
					Confidence: 85,
					Category:   "information",
				},
				Decision: &orchestratorDomain.Decision{
					Type:      orchestratorDomain.DecisionTypeClarify,
					Reasoning: "User asking about capabilities",
				},
			},
		},
	}

	// Create ConversationAwareWebBFF
	webBFF := NewConversationAwareWebBFF(mockOrchestrator, conversationService, userService, logger)

	// Initialize schemas
	err = webBFF.InitializeSchema(ctx)
	require.NoError(t, err)

	t.Run("should persist conversation across multiple messages", func(t *testing.T) {
		sessionID := "test-session-conversation-flow"

		// First message
		response1, err := webBFF.ProcessWebMessageWithConversation(ctx, sessionID, "Hello")
		require.NoError(t, err)
		require.NotNil(t, response1)
		assert.Equal(t, "Hi there! How can I help you today?", response1.Content)
		assert.Equal(t, sessionID, response1.SessionID)
		assert.Equal(t, "greeting", response1.Intent)

		// Verify user and session were created
		user, err := userService.GetUser(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, sessionID, user.ID)
		assert.Equal(t, userDomain.UserTypeWebSession, user.UserType)

		session, err := userService.GetSession(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, sessionID, session.ID)

		// Verify conversation was created and get it with messages
		conversations, err := conversationService.FindConversationsBySession(ctx, sessionID)
		require.NoError(t, err)
		require.Len(t, conversations, 1)

		conversation, err := conversationService.GetConversationWithMessages(ctx, conversations[0].ID)
		require.NoError(t, err)
		assert.Equal(t, sessionID, conversation.SessionID)
		assert.Equal(t, sessionID, conversation.UserID)
		assert.Equal(t, conversationDomain.ConversationStatusActive, conversation.Status)
		assert.Len(t, conversation.Messages, 2) // user + assistant

		// Check first message pair
		userMsg1 := conversation.Messages[0]
		assert.Equal(t, conversationDomain.MessageRoleUser, userMsg1.Role)
		assert.Equal(t, "Hello", userMsg1.Content)

		assistantMsg1 := conversation.Messages[1]
		assert.Equal(t, conversationDomain.MessageRoleAssistant, assistantMsg1.Role)
		assert.Equal(t, "Hi there! How can I help you today?", assistantMsg1.Content)
		assert.Equal(t, "greeting", assistantMsg1.Metadata["analysis_intent"])

		// Second message in same session
		response2, err := webBFF.ProcessWebMessageWithConversation(ctx, sessionID, "What can you do?")
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, "I can help you with various tasks. Let me know what you need!", response2.Content)
		assert.Equal(t, sessionID, response2.SessionID)

		// Verify the same conversation was extended
		conversations, err = conversationService.FindConversationsBySession(ctx, sessionID)
		require.NoError(t, err)
		require.Len(t, conversations, 1) // Still only one conversation

		conversation, err = conversationService.GetConversationWithMessages(ctx, conversations[0].ID)
		require.NoError(t, err)
		assert.Len(t, conversation.Messages, 4) // user1 + assistant1 + user2 + assistant2

		// Check second message pair
		userMsg2 := conversation.Messages[2]
		assert.Equal(t, conversationDomain.MessageRoleUser, userMsg2.Role)
		assert.Equal(t, "What can you do?", userMsg2.Content)

		assistantMsg2 := conversation.Messages[3]
		assert.Equal(t, conversationDomain.MessageRoleAssistant, assistantMsg2.Role)
		assert.Equal(t, "I can help you with various tasks. Let me know what you need!", assistantMsg2.Content)
		assert.Equal(t, "capability_inquiry", assistantMsg2.Metadata["analysis_intent"])

		// Verify conversation continuity
		assert.Equal(t, conversation.Messages[0].ID, userMsg1.ID)
		assert.Equal(t, conversation.Messages[1].ID, assistantMsg1.ID)
		assert.True(t, conversation.Messages[2].Timestamp.After(conversation.Messages[1].Timestamp))
		assert.True(t, conversation.Messages[3].Timestamp.After(conversation.Messages[2].Timestamp))
	})

	t.Run("should create separate conversations for different sessions", func(t *testing.T) {
		sessionID1 := "test-session-1"
		sessionID2 := "test-session-2"

		// Send message to first session
		_, err := webBFF.ProcessWebMessageWithConversation(ctx, sessionID1, "Hello")
		require.NoError(t, err)

		// Send message to second session
		_, err = webBFF.ProcessWebMessageWithConversation(ctx, sessionID2, "Hello")
		require.NoError(t, err)

		// Verify separate conversations
		conversations1, err := conversationService.FindConversationsBySession(ctx, sessionID1)
		require.NoError(t, err)
		require.Len(t, conversations1, 1)

		conversations2, err := conversationService.FindConversationsBySession(ctx, sessionID2)
		require.NoError(t, err)
		require.Len(t, conversations2, 1)

		assert.NotEqual(t, conversations1[0].ID, conversations2[0].ID)
		assert.Equal(t, sessionID1, conversations1[0].SessionID)
		assert.Equal(t, sessionID2, conversations2[0].SessionID)
	})
}

// MockOrchestratorImpl implements AIOrchestrator for testing
type MockOrchestratorImpl struct {
	responses map[string]*orchestratorApp.OrchestratorResult
}

func (m *MockOrchestratorImpl) ProcessRequest(ctx context.Context, userInput string, userID string) (*orchestratorApp.OrchestratorResult, error) {
	if response, exists := m.responses[userInput]; exists {
		return response, nil
	}

	// Default response
	return &orchestratorApp.OrchestratorResult{
		Message: "I understand your request",
		Success: true,
		Analysis: &planningDomain.Analysis{
			Intent:     "general",
			Confidence: 70,
			Category:   "general",
		},
		Decision: &orchestratorDomain.Decision{
			Type:      orchestratorDomain.DecisionTypeClarify,
			Reasoning: "General response",
		},
	}, nil
}
