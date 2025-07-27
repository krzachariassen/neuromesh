package server

import (
	"context"
	"net/http"

	"neuromesh/internal/api/bff"
	"neuromesh/internal/api/rest/v1/controllers"
	"neuromesh/internal/api/rest/v1/domain"
	conversationApp "neuromesh/internal/conversation/application"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
)

// Router configures and manages all API routes for the NeuroMesh server
type Router struct {
	mux                    *http.ServeMux
	conversationController *controllers.ConversationController
	bffService             *bff.Service
	logger                 logging.Logger
}

// NewRouter creates a new router with all API endpoints configured
func NewRouter(
	conversationService conversationApp.ConversationService,
	graphService controllers.GraphService,
	bffService *bff.Service,
	logger logging.Logger,
) *Router {
	mux := http.NewServeMux()

	// Create REST API controllers
	conversationController := controllers.NewConversationController(conversationService)
	conversationController.SetGraphService(graphService)

	router := &Router{
		mux:                    mux,
		conversationController: conversationController,
		bffService:             bffService,
		logger:                 logger,
	}

	// Register all routes
	router.registerRoutes()

	return router
}

// ServeHTTP implements http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Add CORS headers for React UI development
	r.setCORSHeaders(w, req)

	// Handle preflight requests
	if req.Method == http.MethodOptions {
		r.logger.Info("CORS preflight request", "path", req.URL.Path, "origin", req.Header.Get("Origin"))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Log API requests to track potential infinite loops
	if req.URL.Path != "/health" { // Skip health check logs to reduce noise
		r.logger.Info("API request", "method", req.Method, "path", req.URL.Path, "origin", req.Header.Get("Origin"))
	}

	r.mux.ServeHTTP(w, req)
}

// setCORSHeaders sets CORS headers to allow React UI access
func (r *Router) setCORSHeaders(w http.ResponseWriter, req *http.Request) {
	// Allow React dev server (port 3000) and any localhost origins
	origin := req.Header.Get("Origin")
	if origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	// Allow common headers used by React
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

// registerRoutes registers all API routes
func (r *Router) registerRoutes() {
	// REST API v1 routes
	r.registerRESTRoutes()

	// BFF routes (chat and websocket)
	r.registerBFFRoutes()

	// Health and utility routes
	r.registerUtilityRoutes()
}

// registerRESTRoutes registers clean REST API routes
func (r *Router) registerRESTRoutes() {
	// Conversation routes
	r.mux.HandleFunc("/api/v1/conversations/", func(w http.ResponseWriter, req *http.Request) {
		// Parse the path to route to appropriate controller method
		path := req.URL.Path

		// Handle: GET /api/v1/conversations/{id}/graph
		if req.Method == http.MethodGet && len(path) > 23 {
			// Check if path ends with "/graph"
			if len(path) > 6 && path[len(path)-6:] == "/graph" {
				r.conversationController.GetConversationGraph(w, req)
				return
			}

			// Handle: GET /api/v1/conversations/{id}
			r.conversationController.GetConversation(w, req)
			return
		}

		// TODO: Add more REST endpoints as needed
		// - POST /api/v1/conversations (create conversation)
		// - PUT /api/v1/conversations/{id} (update conversation)
		// - DELETE /api/v1/conversations/{id} (delete conversation)
		// - GET /api/v1/conversations/{id}/messages (get messages)
		// - POST /api/v1/conversations/{id}/messages (add message)

		http.NotFound(w, req)
	})

	r.logger.Info("REST API v1 routes registered", "prefix", "/api/v1/conversations")
}

// registerBFFRoutes registers Backend for Frontend routes
func (r *Router) registerBFFRoutes() {
	// Chat API endpoint
	r.mux.Handle("/api/chat", r.bffService.ChatHandler())

	// WebSocket endpoint
	r.mux.Handle("/ws", r.bffService.WebSocketHandler())

	r.logger.Info("BFF routes registered", "endpoints", []string{"/api/chat", "/ws"})
}

// registerUtilityRoutes registers health check and utility routes
func (r *Router) registerUtilityRoutes() {
	// Health check endpoint
	r.mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"neuromesh-server","apis":{"rest":"/api/v1","bff":"/api/chat","websocket":"/ws"}}`))
	})

	// API discovery endpoint
	r.mux.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"api_version": "v1",
			"endpoints": {
				"rest": {
					"base_url": "/api/v1",
					"conversations": "/api/v1/conversations/{id}",
					"conversation_graph": "/api/v1/conversations/{id}/graph"
				},
				"bff": {
					"chat": "/api/chat",
					"websocket": "/ws"
				},
				"utility": {
					"health": "/health",
					"api_info": "/api"
				}
			}
		}`))
	})

	r.logger.Info("Utility routes registered", "endpoints", []string{"/health", "/api"})
}

// GraphServiceAdapter adapts the graph.Graph to the controllers.GraphService interface
type GraphServiceAdapter struct {
	graph graph.Graph
}

// NewGraphServiceAdapter creates a new adapter
func NewGraphServiceAdapter(g graph.Graph) *GraphServiceAdapter {
	return &GraphServiceAdapter{graph: g}
}

// GetConversationGraph implements controllers.GraphService interface
func (a *GraphServiceAdapter) GetConversationGraph(ctx context.Context, conversationID string) (*domain.GraphData, error) {
	// TODO: Implement actual graph data retrieval
	// For now, return richer mock data to test the visualization
	return &domain.GraphData{
		Nodes: []domain.Node{
			// User node
			{
				ID:   "user-1",
				Type: "user",
				Data: map[string]interface{}{
					"name":   "Alice Johnson",
					"status": "active",
				},
				Position: &domain.NodePosition{X: 50, Y: 150},
			},
			// Conversation node
			{
				ID:   conversationID,
				Type: "conversation",
				Data: map[string]interface{}{
					"title":  "AI Task Planning Session",
					"status": "active",
				},
				Position: &domain.NodePosition{X: 300, Y: 150},
			},
			// Agent nodes
			{
				ID:   "agent-text-processor",
				Type: "agent",
				Data: map[string]interface{}{
					"name":   "Text Processor",
					"status": "busy",
				},
				Position: &domain.NodePosition{X: 550, Y: 50},
			},
			{
				ID:   "agent-orchestrator",
				Type: "agent",
				Data: map[string]interface{}{
					"name":   "AI Orchestrator",
					"status": "active",
				},
				Position: &domain.NodePosition{X: 550, Y: 150},
			},
			{
				ID:   "agent-code-generator",
				Type: "agent",
				Data: map[string]interface{}{
					"name":   "Code Generator",
					"status": "idle",
				},
				Position: &domain.NodePosition{X: 550, Y: 250},
			},
			// Execution plan nodes
			{
				ID:   "plan-1",
				Type: "execution_plan",
				Data: map[string]interface{}{
					"title":  "Document Analysis Plan",
					"status": "executing",
				},
				Position: &domain.NodePosition{X: 800, Y: 100},
			},
			{
				ID:   "plan-2",
				Type: "execution_plan",
				Data: map[string]interface{}{
					"title":  "Code Generation Plan",
					"status": "pending",
				},
				Position: &domain.NodePosition{X: 800, Y: 200},
			},
			// Execution step nodes
			{
				ID:   "step-1",
				Type: "execution_step",
				Data: map[string]interface{}{
					"title":  "Parse Document",
					"status": "completed",
				},
				Position: &domain.NodePosition{X: 1050, Y: 50},
			},
			{
				ID:   "step-2",
				Type: "execution_step",
				Data: map[string]interface{}{
					"title":  "Extract Entities",
					"status": "executing",
				},
				Position: &domain.NodePosition{X: 1050, Y: 150},
			},
			// Result node
			{
				ID:   "result-1",
				Type: "result",
				Data: map[string]interface{}{
					"title":  "Analysis Results",
					"status": "ready",
				},
				Position: &domain.NodePosition{X: 1300, Y: 100},
			},
		},
		Edges: []domain.Edge{
			// User to conversation
			{
				ID:     "edge-1",
				Source: "user-1",
				Target: conversationID,
				Type:   "participates_in",
			},
			// Conversation to agents
			{
				ID:     "edge-2",
				Source: conversationID,
				Target: "agent-orchestrator",
				Type:   "assigned_to",
			},
			{
				ID:     "edge-3",
				Source: "agent-orchestrator",
				Target: "agent-text-processor",
				Type:   "delegates_to",
			},
			{
				ID:     "edge-4",
				Source: "agent-orchestrator",
				Target: "agent-code-generator",
				Type:   "delegates_to",
			},
			// Agents to execution plans
			{
				ID:     "edge-5",
				Source: "agent-text-processor",
				Target: "plan-1",
				Type:   "creates",
			},
			{
				ID:     "edge-6",
				Source: "agent-code-generator",
				Target: "plan-2",
				Type:   "creates",
			},
			// Plans to steps
			{
				ID:     "edge-7",
				Source: "plan-1",
				Target: "step-1",
				Type:   "contains",
			},
			{
				ID:     "edge-8",
				Source: "plan-1",
				Target: "step-2",
				Type:   "contains",
			},
			// Steps to results
			{
				ID:     "edge-9",
				Source: "step-1",
				Target: "result-1",
				Type:   "produces",
			},
			{
				ID:     "edge-10",
				Source: "step-2",
				Target: "result-1",
				Type:   "contributes_to",
			},
		},
	}, nil
}
