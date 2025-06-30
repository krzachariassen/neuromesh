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
	"neuromesh/internal/graph"
	"neuromesh/internal/grpc/server"
	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/application"
	"neuromesh/internal/web"
	pb "neuromesh/internal/api/grpc/orchestration"
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

	logger.Info("🧠 Clean Architecture AI Orchestrator initialized and ready!")

	// Create registry service for agent management
	registryService := registry.NewService(productionGraph, logger)

	// Create adapter for web interface compatibility
	orchestratorAdapter := web.NewOrchestratorAdapter(orchestratorService)

	// Create WebBFF for web UI integration with the new orchestrator
	webBFF := web.NewWebBFF(orchestratorAdapter, logger)
	webServer := webBFF.CreateWebServer(":8081")

	logger.Info("🌐 WebBFF server initialized for web UI integration")

	// Create gRPC server (thin proxy layer)
	grpcServer := server.NewOrchestrationServer(aiMessageBus, registryService, logger)

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	// Register the orchestration service
	// Since our protobuf is minimal, we use a custom registration
	pb.RegisterOrchestrationServiceServer(s, grpcServer)

	logger.Info("OrchestrationService registered with gRPC server")

	// Enable reflection for development
	reflection.Register(s)

	logger.Info("Starting gRPC server", "port", 50051)

	// Start server in goroutine
	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Error("Failed to serve gRPC", err)
		}
	}()

	// Start WebBFF HTTP server
	go func() {
		logger.Info("Starting WebBFF HTTP server", "port", 8081)
		if err := webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to serve WebBFF HTTP", err)
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

	// Shutdown WebBFF HTTP server
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := webServer.Shutdown(ctx); err != nil {
			logger.Error("WebBFF HTTP server Shutdown:", err)
		}
	}()

	select {
	case <-done:
		logger.Info("WebBFF HTTP Server gracefully stopped")
	case <-ctx.Done():
		logger.Info("WebBFF HTTP Server shutdown timed out, forcing stop")
	}
}
