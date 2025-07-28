package server

import (
	"context"
	"fmt"
	"net/http"

	"neuromesh/internal/api/bff"
	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	userApp "neuromesh/internal/user/application"
)

// Server represents the unified NeuroMesh API server
type Server struct {
	httpServer *http.Server
	router     *Router
	bffService *bff.Service
	logger     logging.Logger
}

// NewServer creates a new unified API server
func NewServer(
	addr string,
	orchestrator bff.AIOrchestrator,
	conversationService conversationApp.ConversationService,
	userService userApp.UserService,
	graph graph.Graph,
	graphService conversationDomain.ConversationGraphService, // Inject the graph service dependency
	logger logging.Logger,
) *Server {
	// Create BFF service
	bffService := bff.NewService(orchestrator, conversationService, userService, graph, logger)

	// Create router with all APIs, using the injected graph service
	router := NewRouter(conversationService, graphService, bffService, logger)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Server{
		httpServer: httpServer,
		router:     router,
		bffService: bffService,
		logger:     logger,
	}
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	// Initialize schemas
	if err := s.bffService.InitializeSchema(ctx); err != nil {
		return fmt.Errorf("failed to initialize schemas: %w", err)
	}

	s.logger.Info("🚀 NeuroMesh unified API server starting",
		"addr", s.httpServer.Addr,
		"apis", map[string]string{
			"rest":      "/api/v1",
			"bff":       "/api/chat",
			"websocket": "/ws",
			"health":    "/health",
		})

	return s.httpServer.ListenAndServe()
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("🛑 Shutting down NeuroMesh API server")
	return s.httpServer.Shutdown(ctx)
}
