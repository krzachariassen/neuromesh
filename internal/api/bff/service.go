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

// ChatRequest represents a chat request from the web UI with hierarchical context
type ChatRequest struct {
	Message        string `json:"message"`
	SessionID      string `json:"session_id,omitempty"`      // Auto-generated if not provided
	TenantID       string `json:"tenant_id,omitempty"`       // Future: Organization/Company level
	ProjectID      string `json:"project_id,omitempty"`      // Future: Project/Department level
	ConversationID string `json:"conversation_id,omitempty"` // Auto-generated per conversation
	UserID         string `json:"user_id,omitempty"`         // Auto-generated from session
}

// WebResponse represents a response from the BFF to the web client
type WebResponse struct {
	Content        string `json:"content"`
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	ProjectID      string `json:"project_id,omitempty"` // Future: Include project context
	TenantID       string `json:"tenant_id,omitempty"`  // Future: Include tenant context
	Intent         string `json:"intent,omitempty"`
	Error          string `json:"error,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
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

// ProcessMessage processes a message through the BFF with hierarchical ID management
func (s *Service) ProcessMessage(ctx context.Context, sessionID, message string) (*WebResponse, error) {
	// Validate inputs (allowing sessionID to be empty for auto-generation)
	if message == "" {
		return nil, errors.New("message cannot be empty")
	}

	// Check context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create request structure for hierarchical processing
	request := &ChatRequest{
		Message:   message,
		SessionID: sessionID, // May be empty - will be auto-generated
	}

	s.logger.Debug("Processing web message with conversation persistence",
		"providedSessionID", sessionID, "message", message)

	// 1. Get or create conversation with hierarchical context (this handles all ID generation)
	conversation, actualSessionID, _, err := s.getOrCreateConversation(ctx, request)
	if err != nil {
		s.logger.Error("Failed to get or create conversation", err, "sessionID", sessionID)
		return s.handleError("Failed to initialize conversation", actualSessionID), nil
	}

	// 2. Ensure user and session exist (using the actual session ID)
	user, _, err := s.ensureUserAndSession(ctx, actualSessionID)
	if err != nil {
		s.logger.Error("Failed to ensure user and session", err, "sessionID", actualSessionID)
		return s.handleError("Failed to initialize session", actualSessionID), nil
	}

	// 3. Add user message to conversation

	// 3. Add user message to conversation
	userMessageID := generateMessageID()
	err = s.conversationService.AddMessage(ctx, conversation.ID, userMessageID,
		conversationDomain.MessageRoleUser, message, nil)
	if err != nil {
		s.logger.Error("Failed to add user message to conversation", err,
			"conversationID", conversation.ID, "messageID", userMessageID)
		// Continue processing even if message storage fails
	}

	// 4. Process through orchestrator using the new hierarchical interface
	orchestratorRequest := &orchestratorApp.OrchestratorRequest{
		UserInput:      message,
		UserID:         user.ID,
		SessionID:      actualSessionID, // Use the actual session ID (auto-generated if needed)
		MessageID:      userMessageID,
		ConversationID: conversation.ID, // Pass conversation ID for cross-domain relationships
	}

	aiResponse, err := s.orchestrator.ProcessUserRequest(ctx, orchestratorRequest)
	if err != nil {
		s.logger.Error("Failed to process orchestrator request", err, "sessionID", actualSessionID)
		return s.handleError("Failed to process request", actualSessionID), nil
	}

	// Check if orchestrator processing was successful
	if !aiResponse.Success {
		s.logger.Error("Orchestrator processing failed", errors.New(aiResponse.Error), "sessionID", actualSessionID)
		return s.handleError("Failed to process request", actualSessionID), nil
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

	// Note: Relationship management (linking execution plans, decisions, etc.)
	// should be handled by the orchestrator as part of domain logic,
	// not by the BFF layer. The BFF should only handle presentation concerns.

	// 6. Build and return web response with hierarchical IDs
	response := s.buildWebResponse(aiResponse, actualSessionID, conversation.ID, conversation.ProjectID)
	s.logger.Debug("Web message processed successfully", "sessionID", actualSessionID, "response", response.Content)

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

// getOrCreateConversation gets or creates a conversation with hierarchical context support
func (s *Service) getOrCreateConversation(ctx context.Context, request *ChatRequest) (*conversationDomain.Conversation, string, string, error) {
	// Ensure sessionID is set (auto-generate if not provided)
	sessionID := s.ensureSessionID(request.SessionID)

	// Generate userID from sessionID for consistency
	userID := generateUserID(sessionID)

	// Determine projectID (use provided or default)
	projectID := request.ProjectID
	if projectID == "" {
		projectID = "default-project" // Fallback for backward compatibility
	}

	var conversation *conversationDomain.Conversation
	var err error

	if request.ConversationID != "" {
		// Try to get existing conversation
		conversation, err = s.conversationService.GetConversation(ctx, request.ConversationID)
		if err != nil {
			// Conversation doesn't exist, create new one with the provided ID
			conversation, err = s.conversationService.CreateConversation(ctx, request.ConversationID, sessionID, userID, projectID)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to create conversation with ID %s: %w", request.ConversationID, err)
			}
		}
	} else {
		// Try to find existing conversation for this session
		conversations, err := s.conversationService.FindConversationsBySession(ctx, sessionID)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to find conversations for session: %w", err)
		}

		// Find an active conversation
		for _, conv := range conversations {
			if conv.Status == conversationDomain.ConversationStatusActive {
				conversation = conv
				break
			}
		}

		// No active conversation found, create a new one
		if conversation == nil {
			conversationID := generateConversationID()
			conversation, err = s.conversationService.CreateConversation(ctx, conversationID, sessionID, userID, projectID)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to create conversation: %w", err)
			}
		}
	}

	return conversation, sessionID, userID, nil
}

// ensureSessionID returns the provided sessionID or generates a new one if empty
func (s *Service) ensureSessionID(providedSessionID string) string {
	if providedSessionID != "" {
		return providedSessionID
	}
	return fmt.Sprintf("session-%s", uuid.New().String())
}

// buildAssistantMetadata builds metadata for assistant messages based on orchestrator result
func (s *Service) buildAssistantMetadata(aiResponse *orchestratorApp.OrchestratorResult) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Add execution plan ID if available
	if aiResponse.ExecutionPlanID != "" {
		metadata["execution_plan_id"] = aiResponse.ExecutionPlanID
	}

	// Add decision information if available
	if aiResponse.PlanningResult != nil {
		metadata["planning_type"] = string(aiResponse.PlanningResult.Type)
		metadata["planning_id"] = aiResponse.PlanningResult.ID
		metadata["planning_intent"] = aiResponse.PlanningResult.Intent
		metadata["planning_confidence"] = aiResponse.PlanningResult.Confidence
		metadata["planning_reasoning"] = aiResponse.PlanningResult.Reasoning
		if len(aiResponse.PlanningResult.RequiredAgents) > 0 {
			metadata["required_agents"] = aiResponse.PlanningResult.RequiredAgents
		}
		if len(aiResponse.PlanningResult.AgentGap) > 0 {
			metadata["agent_gap"] = aiResponse.PlanningResult.AgentGap
		}
	}

	return metadata
}

// buildWebResponse builds a WebResponse from an orchestrator result with hierarchical context
func (s *Service) buildWebResponse(aiResponse *orchestratorApp.OrchestratorResult, sessionID, conversationID, projectID string) *WebResponse {
	response := &WebResponse{
		Content:        aiResponse.Message,
		SessionID:      sessionID,
		ConversationID: conversationID,
		ProjectID:      projectID,
	}

	// Add execution plan ID if available
	if aiResponse.ExecutionPlanID != "" {
		response.CorrelationID = aiResponse.ExecutionPlanID
	}

	return response
}

// handleError creates an error response with hierarchical context
func (s *Service) handleError(message, sessionID string) *WebResponse {
	return &WebResponse{
		Content:        "I'm sorry, I encountered an error processing your request.",
		SessionID:      sessionID,
		ConversationID: "", // Error responses may not have conversation context
		Error:          message,
	}
}

// Utility functions for ID generation
func generateMessageID() string {
	return uuid.New().String()
}

// GetConversationContext retrieves complete context from graph using only conversation ID
// This implements the graph-native approach instead of passing redundant IDs
func (s *Service) GetConversationContext(ctx context.Context, conversationID string) (*conversationApp.ConversationContext, error) {
	// GREEN phase: Minimal implementation to make tests pass
	// Delegate to conversation service which will query the graph
	return s.conversationService.GetConversationContext(ctx, conversationID)
}

// ProcessMessageGraphNative processes a message using only conversation ID and graph traversal
// This eliminates the anti-pattern of passing multiple redundant IDs
func (s *Service) ProcessMessageGraphNative(ctx context.Context, conversationID, message string) (*WebResponse, error) {
	// GREEN phase: Minimal implementation to make tests pass

	// Get full context from graph using conversation ID
	context, err := s.GetConversationContext(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation context: %w", err)
	}

	// Create orchestrator request with context derived from graph
	request := &orchestratorApp.OrchestratorRequest{
		UserInput:      message,
		UserID:         context.UserID,
		ConversationID: conversationID,
		// Note: No need to pass session/project IDs - orchestrator can query graph if needed
	}

	// Process through orchestrator
	result, err := s.orchestrator.ProcessUserRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("orchestrator failed: %w", err)
	}

	// Build response with context from graph
	return &WebResponse{
		Content:        result.Message,
		ConversationID: conversationID,
		SessionID:      context.SessionID,
		ProjectID:      context.ProjectID,
		// Future: Add TenantID when tenant layer is implemented
	}, nil
}

func generateUserID(sessionID string) string {
	// Generate a user ID based on session ID for consistency
	return fmt.Sprintf("user-%s", sessionID)
}

func generateConversationID() string {
	return uuid.New().String()
}
