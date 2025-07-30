package bff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	projectApp "neuromesh/internal/project/application"
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
	projectService      projectApp.ProjectService
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
	projectService projectApp.ProjectService,
	graph graph.Graph,
	logger logging.Logger,
) *Service {
	return &Service{
		orchestrator:        orchestrator,
		conversationService: conversationService,
		userService:         userService,
		projectService:      projectService,
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
func (s *Service) ProcessMessage(ctx context.Context, sessionID, message, projectID string) (*WebResponse, error) {
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
		ProjectID: projectID, // Include the project ID from the request
	}

	s.logger.Debug("Processing web message with conversation persistence",
		"providedSessionID", sessionID, "message", message, "projectID", projectID)

	// 1. Ensure sessionID is set (auto-generate if not provided)
	actualSessionID := s.ensureSessionID(request.SessionID)

	// 2. Ensure user and session exist FIRST (create User/Session nodes before conversation)
	user, _, err := s.ensureUserAndSession(ctx, actualSessionID)
	if err != nil {
		s.logger.Error("Failed to ensure user and session", err, "sessionID", actualSessionID)
		return s.handleError("Failed to initialize session", actualSessionID), nil
	}

	// 3. Update request with actual User/Session IDs for conversation creation
	request.SessionID = actualSessionID
	request.UserID = user.ID

	// 4. Get or create conversation with hierarchical context (User/Session nodes now exist for relationships)
	conversation, err := s.ensureConversation(ctx, request)
	if err != nil {
		s.logger.Error("Failed to ensure conversation", err, "sessionID", actualSessionID)
		return s.handleError("Failed to initialize conversation", actualSessionID), nil
	}

	// 5. Add user message to conversation
	userMessageID := generateMessageID()
	err = s.conversationService.AddMessage(ctx, conversation.ID, userMessageID,
		conversationDomain.MessageRoleUser, message, nil)
	if err != nil {
		s.logger.Error("Failed to add user message to conversation", err,
			"conversationID", conversation.ID, "messageID", userMessageID)
		// Continue processing even if message storage fails
	}

	// 6. Process through orchestrator using the new hierarchical interface
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

	// 7. Add AI response to conversation
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

	// 8. Build and return web response with hierarchical IDs
	response := s.buildWebResponse(aiResponse, actualSessionID, conversation.ID, projectID)
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

	// Initialize project schema
	if err := s.projectService.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("failed to initialize project schema: %w", err)
	}

	// Ensure default project exists
	if err := s.ensureDefaultProject(ctx); err != nil {
		return fmt.Errorf("failed to ensure default project: %w", err)
	}

	return nil
}

// ensureDefaultProject ensures that the default project exists in the system
func (s *Service) ensureDefaultProject(ctx context.Context) error {
	// Check if default project already exists
	_, err := s.projectService.GetProject(ctx, "default-project")
	if err == nil {
		// Default project already exists
		return nil
	}

	// Create default project
	s.logger.Info("Creating default project for system initialization")
	_, err = s.projectService.CreateProject(ctx, "default-project", "Default Project", "system@neuromesh.ai")
	if err != nil {
		return fmt.Errorf("failed to create default project: %w", err)
	}

	s.logger.Info("Default project created successfully", "projectID", "default-project")
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

// ensureConversation ensures a conversation exists for the request, following clean architecture principles
func (s *Service) ensureConversation(ctx context.Context, request *ChatRequest) (*conversationDomain.Conversation, error) {
	// Validate project exists first
	if err := s.validateProjectExists(ctx, request.ProjectID); err != nil {
		return nil, err
	}

	// Try to get existing conversation if ID is provided
	if request.ConversationID != "" {
		conversation, err := s.getConversation(ctx, request.ConversationID)
		if err == nil {
			return conversation, nil
		}
		// If conversation doesn't exist, create it with the provided ID
		return s.createConversation(ctx, request.ConversationID, request.SessionID, request.UserID, request.ProjectID)
	}

	// Try to find existing active conversation for session
	conversation, err := s.findActiveConversationBySession(ctx, request.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversations by session: %w", err)
	}

	// If found, return it
	if conversation != nil {
		return conversation, nil
	}

	// No active conversation found, create a new one
	conversationID := generateConversationID()
	return s.createConversation(ctx, conversationID, request.SessionID, request.UserID, request.ProjectID)
}

// validateProjectExists validates that a project exists, with fallback to default
func (s *Service) validateProjectExists(ctx context.Context, projectID string) error {
	// Use default project if none specified
	if projectID == "" {
		projectID = "default-project"
	}

	_, err := s.projectService.GetProject(ctx, projectID)
	if err != nil {
		if projectID == "default-project" {
			return fmt.Errorf("default project not found - system misconfiguration: %w", err)
		}
		return fmt.Errorf("project '%s' not found: %w", projectID, err)
	}
	return nil
}

// getConversation retrieves a conversation by ID
func (s *Service) getConversation(ctx context.Context, conversationID string) (*conversationDomain.Conversation, error) {
	return s.conversationService.GetConversation(ctx, conversationID)
}

// createConversation creates a new conversation with the specified parameters
func (s *Service) createConversation(ctx context.Context, conversationID, sessionID, userID, projectID string) (*conversationDomain.Conversation, error) {
	// Use default project if none specified
	if projectID == "" {
		projectID = "default-project"
	}

	return s.conversationService.CreateConversation(ctx, conversationID, sessionID, userID, projectID)
}

// findActiveConversationBySession finds an active conversation for the given session
func (s *Service) findActiveConversationBySession(ctx context.Context, sessionID string) (*conversationDomain.Conversation, error) {
	conversations, err := s.conversationService.FindConversationsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Find an active conversation
	for _, conv := range conversations {
		if conv.Status == conversationDomain.ConversationStatusActive {
			return conv, nil
		}
	}

	return nil, nil // No active conversation found
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
	if aiResponse.ExecutionPlan != nil {
		metadata["planning_type"] = string(aiResponse.ExecutionPlan.Type)
		metadata["planning_id"] = aiResponse.ExecutionPlan.ID
		metadata["planning_intent"] = aiResponse.ExecutionPlan.Intent
		metadata["planning_confidence"] = aiResponse.ExecutionPlan.Confidence
		metadata["planning_reasoning"] = aiResponse.ExecutionPlan.Reasoning
		if len(aiResponse.ExecutionPlan.RequiredAgents) > 0 {
			metadata["required_agents"] = aiResponse.ExecutionPlan.RequiredAgents
		}
		if len(aiResponse.ExecutionPlan.AgentGap) > 0 {
			metadata["agent_gap"] = aiResponse.ExecutionPlan.AgentGap
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

// Project Management Endpoints

// CreateProjectRequest represents a request to create a new project
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OwnerEmail  string `json:"owner_email"`
}

// CreateProjectResponse represents the response after creating a project
type CreateProjectResponse struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// CreateProject creates a new project with the given parameters
func (s *Service) CreateProject(ctx context.Context, request *CreateProjectRequest) (*CreateProjectResponse, error) {
	// Validate request
	if request.Name == "" {
		return nil, errors.New("project name is required")
	}
	if request.OwnerEmail == "" {
		return nil, errors.New("owner email is required")
	}

	// Generate project ID
	projectID := fmt.Sprintf("proj_%s", uuid.New().String())

	// Create project through project service
	project, err := s.projectService.CreateProject(ctx, projectID, request.Name, request.OwnerEmail)
	if err != nil {
		s.logger.Error("Failed to create project", err, "projectID", projectID, "name", request.Name)
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Update description if provided
	if request.Description != "" {
		err = s.projectService.UpdateProjectDescription(ctx, projectID, request.Description)
		if err != nil {
			s.logger.Warn("Failed to update project description after creation",
				"projectID", projectID, "description", request.Description, "error", err)
			// Continue - project was created successfully
		}
	}

	s.logger.Info("Project created successfully", "projectID", projectID, "name", request.Name, "owner", request.OwnerEmail)

	return &CreateProjectResponse{
		ProjectID:   project.ID,
		Name:        project.Name,
		Description: request.Description,
		Status:      string(project.Status),
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
	}, nil
}

// GetProject retrieves project information by ID
func (s *Service) GetProject(ctx context.Context, projectID string) (*CreateProjectResponse, error) {
	project, err := s.projectService.GetProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	return &CreateProjectResponse{
		ProjectID:   project.ID,
		Name:        project.Name,
		Description: project.Description,
		Status:      string(project.Status),
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
	}, nil
}

// HTTP Handlers for Project Management

// CreateProjectHandler returns an HTTP handler for creating projects
func (s *Service) CreateProjectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		response, err := s.CreateProject(r.Context(), &request)
		if err != nil {
			s.logger.Error("Failed to create project", err)
			http.Error(w, "Failed to create project", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetProjectHandler returns an HTTP handler for getting projects
func (s *Service) GetProjectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract project ID from URL path
		projectID := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		if projectID == "" {
			http.Error(w, "Project ID required", http.StatusBadRequest)
			return
		}

		response, err := s.GetProject(r.Context(), projectID)
		if err != nil {
			s.logger.Error("Failed to get project", err, "projectID", projectID)
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
