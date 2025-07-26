package web

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"

	testHelpers "neuromesh/testHelpers"
	userDomain "neuromesh/internal/user/domain"
	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/internal/logging"
	"neuromesh/internal/orchestrator/application"
)

// setupEnhancedWebSocketTest creates a properly configured BFF with mocks for enhanced WebSocket testing
func setupEnhancedWebSocketTest(t *testing.T) (*ConversationAwareWebBFF, func()) {
	logger := logging.NewStructuredLogger(logging.LevelDebug)

	// Create mock orchestrator
	orchestrator := &MockAIOrchestrator{
		responses: make(map[string]*application.OrchestratorResult),
	}

	// Create mock services
	conversationService := testHelpers.NewMockConversationService()
	userService := testHelpers.NewMockUserService()
	testGraph := testHelpers.NewCleanMockGraph()

	bff := NewConversationAwareWebBFF(
		orchestrator,
		conversationService,
		userService,
		testGraph,
		logger,
	)

	cleanup := func() {
		// Only assert expectations if they were set
	}

	return bff, cleanup
}

// setupEnhancedWebSocketTestWithChatExpectations sets up mocks for tests that will send chat messages
func setupEnhancedWebSocketTestWithChatExpectations(t *testing.T) (*ConversationAwareWebBFF, func()) {
	logger := logging.NewStructuredLogger(logging.LevelDebug)

	// Create mock orchestrator
	orchestrator := &MockAIOrchestrator{
		responses: make(map[string]*application.OrchestratorResult),
	}

	// Create mock services with proper expectations
	conversationService := testHelpers.NewMockConversationService()
	userService := testHelpers.NewMockUserService()
	testGraph := testHelpers.NewCleanMockGraph()

	// Set up user service expectations for GetUser calls
	testUser := &userDomain.User{
		ID:        "test-user",
		SessionID: "test-session",
		UserType:  userDomain.UserTypeWebSession,
		Status:    userDomain.UserStatusActive,
	}
	userService.On("GetUser", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("string")).Return(testUser, nil)
	
	// Set up session service expectations
	testSession := &userDomain.Session{
		ID:     "test-session",
		UserID: "test-user",
		Status: userDomain.SessionStatusActive,
	}
	userService.On("GetSession", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("string")).Return(testSession, nil)
	userService.On("CreateSession", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(testSession, nil)
	
	// Set up conversation service expectations
	testConversation := &conversationDomain.Conversation{
		ID:        "test-conversation",
		SessionID: "test-session",
		UserID:    "test-user",
		Messages:  []conversationDomain.ConversationMessage{},
	}
	conversationService.On("FindConversationsBySession", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("string")).Return([]*conversationDomain.Conversation{}, nil) // Return empty to trigger creation
	conversationService.On("CreateConversation", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(testConversation, nil)
	conversationService.On("AddMessage", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("domain.MessageRole"), mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	bff := NewConversationAwareWebBFF(
		orchestrator,
		conversationService,
		userService,
		testGraph,
		logger,
	)

	cleanup := func() {
		// Verify mock expectations were met
		userService.AssertExpectations(t)
		conversationService.AssertExpectations(t)
	}

	return bff, cleanup
}// TestEnhancedWebSocket_RED tests the enhanced WebSocket functionality (RED phase)
func TestEnhancedWebSocket_RED(t *testing.T) {
	t.Run("Should_Accept_WebSocket_Connections", func(t *testing.T) {
		// GIVEN: Enhanced WebSocket handler
		bff, cleanup := setupEnhancedWebSocketTest(t)
		defer cleanup()

		server := httptest.NewServer(bff.EnhancedWebSocketHandler())
		defer server.Close()

		// WHEN: Connecting via WebSocket
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)

		// THEN: Connection should be established successfully
		require.NoError(t, err)
		defer conn.Close()
		assert.NotNil(t, conn)
	})

	t.Run("Should_Handle_Ping_Pong_Messages", func(t *testing.T) {
		// GIVEN: WebSocket connection established
		bff, cleanup := setupEnhancedWebSocketTest(t)
		defer cleanup()

		server := httptest.NewServer(bff.EnhancedWebSocketHandler())
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// WHEN: Sending a ping message
		pingMessage := EnhancedWebSocketMessage{
			Type: EnhancedMessageTypePing,
			ID:   "ping-1",
		}

		err = conn.WriteJSON(pingMessage)
		require.NoError(t, err)

		// THEN: Should receive pong response
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		var response EnhancedWebSocketMessage
		err = conn.ReadJSON(&response)
		require.NoError(t, err)

		assert.Equal(t, EnhancedMessageTypePong, response.Type)
		assert.Equal(t, "ping-1", response.ID)
	})

	// NOTE: Chat message tests are complex due to BFF dependencies
	// These will be tested in integration tests where we can use real dependencies
	// For now, we focus on testing the WebSocket message structure and basic functionality
}
