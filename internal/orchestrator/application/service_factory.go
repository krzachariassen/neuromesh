package application

import (
	aiDomain "neuromesh/internal/ai/domain"
	aiInfrastructure "neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
)

// ServiceFactory creates properly wired orchestrator service instances
type ServiceFactory struct {
	logger             logging.Logger
	graph              graph.Graph
	messageBus         messaging.MessageBus
	aiMessageBus       messaging.AIMessageBus
	aiProvider         aiDomain.AIProvider
	correlationTracker *infrastructure.CorrelationTracker
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(
	logger logging.Logger,
	graph graph.Graph,
	messageBus messaging.MessageBus,
	aiProvider aiDomain.AIProvider,
) *ServiceFactory {
	// Create AIMessageBus from the base MessageBus
	aiMessageBus := messaging.NewAIMessageBus(messageBus, graph, logger)

	return &ServiceFactory{
		logger:       logger,
		graph:        graph,
		messageBus:   messageBus,
		aiMessageBus: aiMessageBus,
		aiProvider:   aiProvider,
	}
}

// CreateOrchestratorService creates a fully wired orchestrator service
func (sf *ServiceFactory) CreateOrchestratorService() *OrchestratorService {
	// Create infrastructure services
	agentService := infrastructure.NewGraphAgentService(sf.graph)
	conversationService := infrastructure.NewGraphConversationService(sf.graph)

	// Create all application services with proper dependencies
	aiDecisionEngine := NewAIDecisionEngine(sf.aiProvider)
	graphExplorer := NewGraphExplorer(agentService)
	aiConversationEngine := NewAIConversationEngine(sf.aiProvider, sf.aiMessageBus, sf.correlationTracker)
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

// CreateAIProvider creates an AI provider with the given configuration
func CreateAIProvider(config *aiInfrastructure.OpenAIConfig, logger logging.Logger) aiDomain.AIProvider {
	return aiInfrastructure.NewOpenAIProvider(config, logger)
}
