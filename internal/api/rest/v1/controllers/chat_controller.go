package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"neuromesh/internal/api/rest/v1/domain"
	"neuromesh/internal/api/rest/v1/responses"
	conversationApp "neuromesh/internal/conversation/application"
	projectApp "neuromesh/internal/project/application"
	userApp "neuromesh/internal/user/application"

	"github.com/google/uuid"
)

// ChatController handles HTTP requests for chat API endpoints
type ChatController struct {
	conversationService conversationApp.ConversationService
	projectService      projectApp.ProjectService
	userService         userApp.UserService
}

// NewChatController creates a new chat controller
func NewChatController(
	conversationService conversationApp.ConversationService,
	projectService projectApp.ProjectService,
	userService userApp.UserService,
) *ChatController {
	return &ChatController{
		conversationService: conversationService,
		projectService:      projectService,
		userService:         userService,
	}
}

// StartNewConversation handles POST /api/v1/chat
func (c *ChatController) StartNewConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responses.MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.BadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.Message == "" {
		responses.BadRequest(w, "message is required")
		return
	}

	if req.ProjectID == "" {
		responses.BadRequest(w, "project_id is required")
		return
	}

	if req.UserID == "" {
		responses.BadRequest(w, "user_id is required")
		return
	}

	// Verify project exists
	_, err := c.projectService.GetProject(r.Context(), req.ProjectID)
	if err != nil {
		responses.BadRequest(w, "project not found")
		return
	}

	// Verify user exists
	_, err = c.userService.GetUser(r.Context(), req.UserID)
	if err != nil {
		responses.BadRequest(w, "user not found")
		return
	}

	// Generate IDs for the new conversation
	conversationID := uuid.New().String()
	sessionID := uuid.New().String()

	// Create conversation
	conversation, err := c.conversationService.CreateConversation(r.Context(), conversationID, sessionID, req.UserID, req.ProjectID)
	if err != nil {
		responses.InternalError(w, "Failed to create conversation")
		return
	}

	// TODO: Process the message with AI orchestrator (for now, return a simple response)
	response := domain.ChatResponse{
		ConversationID: conversationID,
		SessionID:      sessionID,
		Response:       "I understand your message: " + req.Message,
		ProjectID:      conversation.ProjectID,
		UserID:         conversation.UserID,
	}

	responses.Success(w, response)
}

// ContinueConversation handles POST /api/v1/chat/{conversation_id}
func (c *ChatController) ContinueConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responses.MethodNotAllowed(w, "Method not allowed")
		return
	}

	// Extract conversation ID from URL path
	conversationID := c.extractConversationID(r.URL.Path)
	if conversationID == "" {
		responses.BadRequest(w, "conversation_id is required")
		return
	}

	var req domain.ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.BadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.Message == "" {
		responses.BadRequest(w, "message is required")
		return
	}

	// Verify conversation exists
	conversation, err := c.conversationService.GetConversation(r.Context(), conversationID)
	if err != nil {
		responses.BadRequest(w, "conversation not found")
		return
	}

	// TODO: Process the message with AI orchestrator (for now, return a simple response)
	response := domain.ChatResponse{
		ConversationID: conversationID,
		SessionID:      conversation.SessionID,
		Response:       "I understand your follow-up message: " + req.Message,
		ProjectID:      conversation.ProjectID,
		UserID:         conversation.UserID,
	}

	responses.Success(w, response)
}

// extractConversationID extracts conversation ID from /api/v1/chat/{id}
func (c *ChatController) extractConversationID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 5 && parts[4] != "" {
		return parts[4]
	}
	return ""
}
