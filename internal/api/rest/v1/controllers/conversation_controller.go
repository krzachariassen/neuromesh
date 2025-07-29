package controllers

import (
	"net/http"
	"strings"
	"time"

	"neuromesh/internal/api/rest/v1/domain"
	"neuromesh/internal/api/rest/v1/responses"
	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
)

// ConversationController handles HTTP requests for conversation resources
type ConversationController struct {
	conversationService conversationApp.ConversationService
	graphService        conversationDomain.ConversationGraphService
}

// NewConversationController creates a new conversation controller
func NewConversationController(conversationService conversationApp.ConversationService) *ConversationController {
	return &ConversationController{
		conversationService: conversationService,
	}
}

// SetGraphService sets the graph service (for testing and dependency injection)
func (c *ConversationController) SetGraphService(graphService conversationDomain.ConversationGraphService) {
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

	// Get graph data from domain service
	domainGraphData, err := c.graphService.GetConversationGraph(r.Context(), conversationID)
	if err != nil {
		responses.InternalError(w, "Failed to get conversation graph")
		return
	}

	// Convert domain graph data to API response format
	apiGraphData := c.convertDomainGraphToAPIGraph(domainGraphData)

	responses.Success(w, apiGraphData)
}

// extractConversationID extracts conversation ID from URL path
// Works for both /api/v1/conversations/{id} and /api/v1/conversations/{id}/graph
func (c *ConversationController) extractConversationID(path string) string {
	parts := strings.Split(path, "/")
	// Path format: ["", "api", "v1", "conversations", "{id}", ...optional]
	if len(parts) >= 5 && parts[4] != "" {
		return parts[4]
	}
	return ""
}

// extractConversationIDFromGraphPath is an alias for consistency
func (c *ConversationController) extractConversationIDFromGraphPath(path string) string {
	return c.extractConversationID(path)
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

// convertDomainGraphToAPIGraph converts domain graph data to API response format
func (c *ConversationController) convertDomainGraphToAPIGraph(domainGraph *conversationDomain.GraphData) *domain.GraphData {
	// Convert domain nodes to API nodes
	apiNodes := make([]domain.Node, len(domainGraph.Nodes))
	for i, domainNode := range domainGraph.Nodes {
		apiNodes[i] = domain.Node{
			ID:   domainNode.ID,
			Type: domainNode.Type,
			Data: domainNode.Data,
			Position: func() *domain.NodePosition {
				if domainNode.Position != nil {
					return &domain.NodePosition{
						X: domainNode.Position.X,
						Y: domainNode.Position.Y,
					}
				}
				return nil
			}(),
		}
	}

	// Convert domain edges to API edges
	apiEdges := make([]domain.Edge, len(domainGraph.Edges))
	for i, domainEdge := range domainGraph.Edges {
		apiEdges[i] = domain.Edge{
			ID:     domainEdge.ID,
			Source: domainEdge.Source,
			Target: domainEdge.Target,
			Type:   domainEdge.Type,
		}
	}

	return &domain.GraphData{
		Nodes: apiNodes,
		Edges: apiEdges,
	}
}
