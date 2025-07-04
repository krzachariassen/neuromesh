package application

import (
	"context"
	"fmt"

	aiDomain "neuromesh/internal/ai/domain"
	aiInfrastructure "neuromesh/internal/ai/infrastructure"
	conversationApp "neuromesh/internal/conversation/application"
	conversationInfra "neuromesh/internal/conversation/infrastructure"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
	planningApp "neuromesh/internal/planning/application"
)

// ServiceFactory creates properly wired orchestrator service instances
type ServiceFactory struct {
	logger                logging.Logger
	graph                 graph.Graph
	messageBus            messaging.MessageBus
	aiMessageBus          messaging.AIMessageBus
	aiProvider            aiDomain.AIProvider
	correlationTracker    *infrastructure.CorrelationTracker
	globalMessageConsumer *infrastructure.GlobalMessageConsumer
	shutdownContext       context.Context
	shutdownCancel        context.CancelFunc
	started               bool // Track startup state to prevent double-start
}

// NewServiceFactory creates a new service factory with proper dependency wiring
func NewServiceFactory(
	logger logging.Logger,
	graph graph.Graph,
	messageBus messaging.MessageBus,
	aiProvider aiDomain.AIProvider,
) *ServiceFactory {
	// Create shutdown context for graceful cleanup
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	// Create correlation tracker
	correlationTracker := infrastructure.NewCorrelationTracker()

	// Create AIMessageBus from the base MessageBus (only if dependencies are available)
	var aiMessageBus messaging.AIMessageBus
	var globalMessageConsumer *infrastructure.GlobalMessageConsumer

	if messageBus != nil && graph != nil {
		aiMessageBus = messaging.NewAIMessageBus(messageBus, graph, logger)
		globalMessageConsumer = infrastructure.NewGlobalMessageConsumer(aiMessageBus, correlationTracker)
	}

	return &ServiceFactory{
		logger:                logger,
		graph:                 graph,
		messageBus:            messageBus,
		aiMessageBus:          aiMessageBus,
		aiProvider:            aiProvider,
		correlationTracker:    correlationTracker,
		globalMessageConsumer: globalMessageConsumer,
		shutdownContext:       shutdownCtx,
		shutdownCancel:        shutdownCancel,
	}
}

// CreateOrchestratorService creates a fully wired orchestrator service
func (sf *ServiceFactory) CreateOrchestratorService() *OrchestratorService {
	// Create infrastructure services
	agentService := infrastructure.NewGraphAgentService(sf.graph)
	conversationService := conversationInfra.NewGraphConversationService(sf.graph)

	// Create all application services with proper dependencies
	aiDecisionEngine := planningApp.NewAIDecisionEngine(sf.aiProvider)
	graphExplorer := NewGraphExplorer(agentService)
	aiConversationEngine := conversationApp.NewAIConversationEngine(sf.aiProvider, sf.aiMessageBus, sf.correlationTracker)
	learningService := NewLearningService(conversationService)

	// Wire everything together
	return NewOrchestratorService(
		aiDecisionEngine,
		graphExplorer,
		aiConversationEngine,
		learningService,
		sf.logger,
	)
}

// StartServices starts all background services in proper order
func (sf *ServiceFactory) StartServices(ctx context.Context) error {
	sf.logger.Info("ServiceFactory: Starting background services...")

	// Check if already started
	if sf.started {
		sf.logger.Warn("ServiceFactory: Services already started, skipping startup")
		return nil
	}

	// Check if required dependencies are available
	if sf.globalMessageConsumer == nil {
		return fmt.Errorf("global message consumer not initialized - ensure both messageBus and graph are provided")
	}

	if sf.aiMessageBus == nil {
		return fmt.Errorf("AI message bus not initialized - ensure both messageBus and graph are provided")
	}

	// Start global message consumer for correlation-based routing
	err := sf.globalMessageConsumer.StartConsumption(sf.shutdownContext, "ai-orchestrator")
	if err != nil {
		return fmt.Errorf("failed to start global message consumer: %w", err)
	}

	// Mark as started
	sf.started = true
	sf.logger.Info("ServiceFactory: All services started successfully")
	return nil
}

// Shutdown performs graceful shutdown of all services
func (sf *ServiceFactory) Shutdown() error {
	sf.logger.Info("ServiceFactory: Starting graceful shutdown...")

	// Cancel shutdown context to stop all background services
	sf.shutdownCancel()

	// Cleanup pending correlation requests
	if sf.correlationTracker != nil {
		sf.correlationTracker.CleanupAll()
	}

	// Reset started state
	sf.started = false

	sf.logger.Info("ServiceFactory: Graceful shutdown completed")
	return nil
}

// CreateAIProvider creates an AI provider with the given configuration
func CreateAIProvider(config *aiInfrastructure.OpenAIConfig, logger logging.Logger) aiDomain.AIProvider {
	return aiInfrastructure.NewOpenAIProvider(config, logger)
}
