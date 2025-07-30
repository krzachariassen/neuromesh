package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"neuromesh/internal/agent/registry"
	aiInfrastructure "neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/api/bff"
	pb "neuromesh/internal/api/grpc/api"
	apiServer "neuromesh/internal/api/server"
	conversationInfrastructure "neuromesh/internal/conversation/infrastructure"
	"neuromesh/internal/graph"
	grpcServer "neuromesh/internal/grpc/server"
	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/application"
)

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	// Create context for the entire application
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create RabbitMQ message bus for production-grade messaging
	rabbitmqURL := getEnvOrDefault("RABBITMQ_URL", "amqp://orchestrator:orchestrator123@localhost:5672/")
	messageBusConfig := messaging.RabbitMQConfig{
		URL:            rabbitmqURL,
		ReconnectDelay: 5 * time.Second,
		MaxReconnects:  5,
		Heartbeat:      10 * time.Second,
	}

	messageBus := messaging.NewRabbitMQMessageBus(messageBusConfig, logger)

	// Connect to RabbitMQ
	if err := messageBus.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// Ensure RabbitMQ is closed on shutdown
	defer func() {
		if err := messageBus.Close(); err != nil {
			logger.Error("Failed to close RabbitMQ connection", err)
		}
	}()

	logger.Info("✅ Connected to RabbitMQ for agent messaging")

	// Create production Neo4j graph
	graphConfig := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      getEnvOrDefault("NEO4J_URL", "bolt://localhost:7687"),
		Neo4jUser:     getEnvOrDefault("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnvOrDefault("NEO4J_PASSWORD", "orchestrator123"),
	}

	productionGraph, err := graph.NewNeo4jGraph(ctx, graphConfig, logger)
	if err != nil {
		log.Fatalf("Failed to initialize Neo4j graph: %v", err)
	}

	// Ensure graph is closed on shutdown
	defer func() {
		if err := productionGraph.Close(ctx); err != nil {
			logger.Error("Failed to close graph connection", err)
		}
	}()

	// Create AI message bus (graph is used for message storage and context)
	aiMessageBus := messaging.NewAIMessageBus(messageBus, productionGraph, logger)

	// Create AI provider (production OpenAI with new clean architecture)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logger.Warn("OPENAI_API_KEY not set, using placeholder - AI functionality will not work")
		apiKey = "placeholder"
	}

	aiConfig := aiInfrastructure.DefaultOpenAIConfig()
	aiConfig.APIKey = apiKey
	aiProvider := aiInfrastructure.NewOpenAIProvider(aiConfig, logger)

	// Create the orchestrator service using the service factory for proper wiring
	serviceFactory := application.NewServiceFactory(logger, productionGraph, messageBus, aiProvider)
	orchestratorService := serviceFactory.CreateOrchestratorService()

	// Get conversation and user services from service factory for conversation persistence
	conversationService := serviceFactory.GetConversationService()
	userService := serviceFactory.GetUserService()
	projectService := serviceFactory.GetProjectService()

	// Ensure service factory is properly shut down
	defer func() {
		if err := serviceFactory.Shutdown(); err != nil {
			logger.Error("Failed to shutdown service factory", err)
		}
	}()

	// Start all background services (including GlobalMessageConsumer)
	err = serviceFactory.StartServices(ctx)
	if err != nil {
		log.Fatalf("Failed to start background services: %v", err)
	}

	logger.Info("🧠 Clean Architecture AI Orchestrator initialized and ready!")

	// Create registry service for agent management
	registryService := registry.NewService(productionGraph, logger)

	// Create adapter for orchestrator service to work with BFF
	orchestratorAdapter := bff.NewOrchestratorAdapter(orchestratorService)

	// Create conversation graph service using clean architecture principles
	// This follows dependency injection - the API layer doesn't know it's Neo4j
	conversationGraphService := conversationInfrastructure.NewConversationGraphRepository(productionGraph, logger)

	// Create unified API server with clean architecture and proper dependency injection
	unifiedServer := apiServer.NewServer(
		":8080",
		orchestratorAdapter,
		conversationService,
		userService,
		projectService,
		productionGraph,
		conversationGraphService, // Inject the graph service abstraction
		logger,
	)

	logger.Info("🌐 Unified API server initialized with clean architecture")

	// Create gRPC server (thin proxy layer)
	grpcSrv := grpcServer.NewOrchestrationServer(aiMessageBus, registryService, logger)

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterOrchestrationServiceServer(s, grpcSrv)
	reflection.Register(s)

	logger.Info("🚀 gRPC server initialized", "port", ":50051")

	// Start gRPC server in background
	go func() {
		logger.Info("🎧 gRPC server starting")
		if err := s.Serve(lis); err != nil {
			logger.Error("gRPC server failed", err)
		}
	}()

	// Start unified API server in background
	go func() {
		logger.Info("🌐 Starting unified API server")
		if err := unifiedServer.Start(ctx); err != nil && err != http.ErrServerClosed {
			logger.Error("Unified API server failed", err)
		}
	}()

	// Start agent health monitoring background process
	go func() {
		logger.Info("Starting agent health monitoring", "interval", "30s")
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := registryService.MonitorAgentHealth(ctx); err != nil {
					logger.Error("Agent health monitoring failed", err)
				}
			case <-ctx.Done():
				logger.Info("Agent health monitoring stopped")
				return
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		s.GracefulStop()
		done <- true
	}()

	select {
	case <-done:
		logger.Info("gRPC Server gracefully stopped")
	case <-ctx.Done():
		logger.Info("gRPC Server shutdown timed out, forcing stop")
		s.Stop()
	}

	// Shutdown unified API server
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := unifiedServer.Stop(ctx); err != nil {
			logger.Error("Unified API server Stop:", err)
		}
	}()

	select {
	case <-done:
		logger.Info("Unified API Server gracefully stopped")
	case <-ctx.Done():
		logger.Info("Unified API Server shutdown timed out, forcing stop")
	}
}
