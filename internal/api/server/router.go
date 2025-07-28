package server

import (
	"net/http"

	"neuromesh/internal/api/bff"
	"neuromesh/internal/api/rest/v1/controllers"
	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
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
	graphService conversationDomain.ConversationGraphService,
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
