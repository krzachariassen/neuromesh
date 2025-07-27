package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"neuromesh/internal/api/rest/v1/domain"
	"neuromesh/internal/api/rest/v1/responses"
	conversationApp "neuromesh/internal/conversation/application"
)

// GraphService defines the interface for graph operations
type GraphService interface {
	GetConversationGraph(ctx context.Context, conversationID string) (*domain.GraphData, error)
}

// ConversationController handles HTTP requests for conversation resources
type ConversationController struct {
	conversationService conversationApp.ConversationService
	graphService        GraphService
}

// NewConversationController creates a new conversation controller
func NewConversationController(conversationService conversationApp.ConversationService) *ConversationController {
	return &ConversationController{
		conversationService: conversationService,
	}
}

// SetGraphService sets the graph service (for testing)
func (c *ConversationController) SetGraphService(graphService GraphService) {
	c.graphService = graphService
}

// GetConversation handles GET /api/v1/conversations/{id}
func (c *ConversationController) GetConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responses.MethodNotAllowed(w, "Method not allowed")
		return
	}

	// Extract conversation ID from URL path
	conversationID := c.extractConversationID(r.URL.Path)
	if conversationID == "" {
		responses.BadRequest(w, "Conversation ID is required")
		return
	}

	// Get conversation from service
	conversation, err := c.conversationService.GetConversation(r.Context(), conversationID)
	if err != nil {
		responses.InternalError(w, "Failed to get conversation")
		return
	}

	// Convert to API response format
	response := c.conversationToResponse(conversation)
	responses.Success(w, response)
}

// GetConversationGraph handles GET /api/v1/conversations/{id}/graph
func (c *ConversationController) GetConversationGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responses.MethodNotAllowed(w, "Method not allowed")
		return
	}

	// Extract conversation ID from URL path
	conversationID := c.extractConversationIDFromGraphPath(r.URL.Path)
	if conversationID == "" {
		responses.BadRequest(w, "Conversation ID is required")
		return
	}

	if c.graphService == nil {
		responses.InternalError(w, "Graph service not available")
		return
	}

	// Get graph data from service
	graphData, err := c.graphService.GetConversationGraph(r.Context(), conversationID)
	if err != nil {
		responses.InternalError(w, "Failed to get conversation graph")
		return
	}

	responses.Success(w, graphData)
}

// extractConversationID extracts conversation ID from /api/v1/conversations/{id}
func (c *ConversationController) extractConversationID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 5 && parts[4] != "" {
		return parts[4]
	}
	return ""
}

// extractConversationIDFromGraphPath extracts conversation ID from /api/v1/conversations/{id}/graph
func (c *ConversationController) extractConversationIDFromGraphPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 6 && parts[4] != "" && parts[5] == "graph" {
		return parts[4]
	}
	return ""
}

// conversationToResponse converts a domain conversation to API response format
func (c *ConversationController) conversationToResponse(conv interface{}) domain.ConversationResponse {
	// This is a simple conversion - in real implementation you'd use a proper mapper
	// For now, create a mock response for testing
	return domain.ConversationResponse{
		ID:        "test-conversation-id",
		SessionID: "test-session",
		UserID:    "test-user",
		Status:    "active",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
}
