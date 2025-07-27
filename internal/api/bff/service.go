package bff

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	userApp "neuromesh/internal/user/application"
	userDomain "neuromesh/internal/user/domain"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ChatRequest represents a chat request from the web UI
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// WebResponse represents a response from the BFF to the web client
type WebResponse struct {
	Content       string `json:"content"`
	SessionID     string `json:"session_id"`
	Intent        string `json:"intent,omitempty"`
	Error         string `json:"error,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// AIOrchestrator defines the interface for AI orchestration operations
type AIOrchestrator interface {
	ProcessRequest(ctx context.Context, userInput, userID string) (*orchestratorApp.OrchestratorResult, error)
	ProcessUserRequest(ctx context.Context, request *orchestratorApp.OrchestratorRequest) (*orchestratorApp.OrchestratorResult, error)
}

// WebSession represents a web user session
type WebSession struct {
	SessionID string
	UserID    string
	CreatedAt int64
}

// Service provides Backend for Frontend functionality with conversation persistence
// This unified service replaces both WebBFF and ConversationAwareWebBFF
type Service struct {
	orchestrator        AIOrchestrator
	conversationService conversationApp.ConversationService
	userService         userApp.UserService
	graph               graph.Graph
	logger              logging.Logger
	sessions            map[string]*WebSession
	sessionMutex        sync.RWMutex
	upgrader            websocket.Upgrader
}

// NewService creates a new unified BFF service
func NewService(
	orchestrator AIOrchestrator,
	conversationService conversationApp.ConversationService,
	userService userApp.UserService,
	graph graph.Graph,
	logger logging.Logger,
) *Service {
	return &Service{
		orchestrator:        orchestrator,
		conversationService: conversationService,
		userService:         userService,
		graph:               graph,
		logger:              logger,
		sessions:            make(map[string]*WebSession),
		sessionMutex:        sync.RWMutex{},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin for development
				// TODO: Implement proper origin checking for production
				return true
			},
		},
	}
}

// ProcessMessage processes a message from a web session with full conversation persistence
func (s *Service) ProcessMessage(ctx context.Context, sessionID, message string) (*WebResponse, error) {
	// Validate input
	if sessionID == "" {
		return nil, errors.New("session ID cannot be empty")
	}
	if message == "" {
		return nil, errors.New("message cannot be empty")
	}

	// Check context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	s.logger.Debug("Processing web message with conversation persistence",
		"sessionID", sessionID, "message", message)

	// 1. Ensure user and session exist
	user, _, err := s.ensureUserAndSession(ctx, sessionID)
	if err != nil {
		s.logger.Error("Failed to ensure user and session", err, "sessionID", sessionID)
		return s.handleError("Failed to initialize session", sessionID), nil
	}

	// 2. Get or create conversation for this session
	conversation, err := s.getOrCreateConversation(ctx, sessionID, user.ID)
	if err != nil {
		s.logger.Error("Failed to get or create conversation", err, "sessionID", sessionID)
		return s.handleError("Failed to initialize conversation", sessionID), nil
	}

	// 3. Add user message to conversation
	userMessageID := generateMessageID()
	err = s.conversationService.AddMessage(ctx, conversation.ID, userMessageID,
		conversationDomain.MessageRoleUser, message, nil)
	if err != nil {
		s.logger.Error("Failed to add user message to conversation", err,
			"conversationID", conversation.ID, "messageID", userMessageID)
		// Continue processing even if message storage fails
	}

	// 4. Process through orchestrator using the new interface
	orchestratorRequest := &orchestratorApp.OrchestratorRequest{
		UserInput: message,
		UserID:    user.ID,
		SessionID: sessionID,
		MessageID: userMessageID,
	}

	aiResponse, err := s.orchestrator.ProcessUserRequest(ctx, orchestratorRequest)
	if err != nil {
		s.logger.Error("Failed to process orchestrator request", err, "sessionID", sessionID)
		return s.handleError("Failed to process request", sessionID), nil
	}

	// Check if orchestrator processing was successful
	if !aiResponse.Success {
		s.logger.Error("Orchestrator processing failed", errors.New(aiResponse.Error), "sessionID", sessionID)
		return s.handleError("Failed to process request", sessionID), nil
	}

	// 5. Add AI response to conversation
	assistantMessageID := generateMessageID()
	assistantMetadata := s.buildAssistantMetadata(aiResponse)

	err = s.conversationService.AddMessage(ctx, conversation.ID, assistantMessageID,
		conversationDomain.MessageRoleAssistant, aiResponse.Message, assistantMetadata)
	if err != nil {
		s.logger.Error("Failed to add assistant message to conversation", err,
			"conversationID", conversation.ID, "messageID", assistantMessageID)
		// Continue with response even if storage fails
	}

	// 6. Build and return web response
	response := s.buildWebResponse(aiResponse, sessionID)
	s.logger.Debug("Web message processed successfully", "sessionID", sessionID, "response", response.Content)

	return response, nil
}

// InitializeSchema ensures all required schemas are in place
func (s *Service) InitializeSchema(ctx context.Context) error {
	// Initialize conversation schema
	if err := s.conversationService.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("failed to initialize conversation schema: %w", err)
	}

	// Initialize user schema
	if err := s.userService.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("failed to initialize user schema: %w", err)
	}

	return nil
}

// ensureUserAndSession ensures that a user and session exist for the given session ID
func (s *Service) ensureUserAndSession(ctx context.Context, sessionID string) (*userDomain.User, *userDomain.Session, error) {
	// Get or create session
	session := s.getOrCreateSession(sessionID)

	// Check if user exists
	user, err := s.userService.GetUser(ctx, session.UserID)
	if err != nil {
		// User doesn't exist, create one with appropriate user type
		userID := session.UserID
		user, err = s.userService.CreateUser(ctx, userID, sessionID, userDomain.UserTypeWebSession)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Ensure session exists in user service with reasonable duration
	userSession, err := s.userService.GetSession(ctx, sessionID)
	if err != nil {
		// Session doesn't exist, create one (24 hours duration)
		userSession, err = s.userService.CreateSession(ctx, sessionID, user.ID, 24*time.Hour)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	return user, userSession, nil
}

// getOrCreateSession gets an existing session or creates a new one
func (s *Service) getOrCreateSession(sessionID string) *WebSession {
	s.sessionMutex.RLock()
	session, exists := s.sessions[sessionID]
	s.sessionMutex.RUnlock()

	if exists {
		return session
	}

	// Create new session
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	// Double-check after acquiring write lock
	if session, exists := s.sessions[sessionID]; exists {
		return session
	}

	session = &WebSession{
		SessionID: sessionID,
		UserID:    generateUserID(sessionID),
		CreatedAt: time.Now().Unix(),
	}

	s.sessions[sessionID] = session
	return session
}

// getOrCreateConversation gets or creates a conversation for the session
func (s *Service) getOrCreateConversation(ctx context.Context, sessionID, userID string) (*conversationDomain.Conversation, error) {
	// Try to find existing conversation for this session
	conversations, err := s.conversationService.FindConversationsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversations for session: %w", err)
	}

	// Find an active conversation
	for _, conv := range conversations {
		if conv.Status == conversationDomain.ConversationStatusActive {
			return conv, nil
		}
	}

	// No active conversation found, create a new one
	conversationID := generateConversationID()
	conversation, err := s.conversationService.CreateConversation(ctx, conversationID, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conversation, nil
}

// buildAssistantMetadata builds metadata for assistant messages based on orchestrator result
func (s *Service) buildAssistantMetadata(aiResponse *orchestratorApp.OrchestratorResult) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Add execution plan ID if available
	if aiResponse.ExecutionPlanID != "" {
		metadata["execution_plan_id"] = aiResponse.ExecutionPlanID
	}

	// Add decision information if available
	if aiResponse.Decision != nil {
		metadata["decision_type"] = string(aiResponse.Decision.Type)
		metadata["decision_id"] = aiResponse.Decision.ID
		if aiResponse.Decision.Action != "" {
			metadata["decision_action"] = aiResponse.Decision.Action
		}
	}

	// Add analysis information if available
	if aiResponse.Analysis != nil {
		metadata["analysis_confidence"] = aiResponse.Analysis.Confidence
		metadata["analysis_reasoning"] = aiResponse.Analysis.Reasoning
	}

	return metadata
}

// buildWebResponse builds a WebResponse from an orchestrator result
func (s *Service) buildWebResponse(aiResponse *orchestratorApp.OrchestratorResult, sessionID string) *WebResponse {
	response := &WebResponse{
		Content:   aiResponse.Message,
		SessionID: sessionID,
	}

	// Add execution plan ID if available
	if aiResponse.ExecutionPlanID != "" {
		response.CorrelationID = aiResponse.ExecutionPlanID
	}

	return response
}

// handleError creates an error response
func (s *Service) handleError(message, sessionID string) *WebResponse {
	return &WebResponse{
		Content:   "I'm sorry, I encountered an error processing your request.",
		SessionID: sessionID,
		Error:     message,
	}
}

// Utility functions for ID generation
func generateMessageID() string {
	return uuid.New().String()
}

func generateUserID(sessionID string) string {
	// Generate a user ID based on session ID for consistency
	return fmt.Sprintf("user-%s", sessionID)
}

func generateConversationID() string {
	return uuid.New().String()
}
