package application

import (
	"context"
	"encoding/json"
	"fmt"

	"neuromesh/internal/agent/domain"
)

// AgentService defines the interface for agent operations
type AgentService interface {
	GetAvailableAgents(ctx context.Context) ([]*domain.Agent, error)
	RegisterAgent(ctx context.Context, agent *domain.Agent) error
	DiscoverAgentsByCapability(ctx context.Context, capability string) ([]*domain.Agent, error)
	UpdateAgentStatus(ctx context.Context, agentID string, status domain.AgentStatus) error
}

// GraphExplorer handles agent discovery and context formatting for AI consumption
type GraphExplorer struct {
	agentService AgentService
}

// NewGraphExplorer creates a new GraphExplorer instance
func NewGraphExplorer(agentService AgentService) *GraphExplorer {
	return &GraphExplorer{
		agentService: agentService,
	}
}

// GetAgentContext retrieves all available agents and formats them for AI consumption
// Returns structured JSON format for precise AI parsing
func (g *GraphExplorer) GetAgentContext(ctx context.Context) (string, error) {
	agents, err := g.agentService.GetAvailableAgents(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get available agents: %w", err)
	}

	if len(agents) == 0 {
		return `{"available_agents": []}`, nil
	}

	// Create structured agent data for AI consumption
	type AgentContext struct {
		ID           string                   `json:"id"`
		Name         string                   `json:"name"`
		Status       string                   `json:"status"`
		Capabilities []domain.AgentCapability `json:"capabilities"`
	}

	type AgentContextResponse struct {
		AvailableAgents []AgentContext `json:"available_agents"`
	}

	var agentContexts []AgentContext
	for _, agent := range agents {
		agentContexts = append(agentContexts, AgentContext{
			ID:           agent.ID,
			Name:         agent.Name,
			Status:       string(agent.Status),
			Capabilities: agent.Capabilities,
		})
	}

	response := AgentContextResponse{
		AvailableAgents: agentContexts,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal agent context to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// FindCapableAgents finds agents with specific capabilities
func (g *GraphExplorer) FindCapableAgents(ctx context.Context, capabilities []string) ([]*domain.Agent, error) {
	var allAgents []*domain.Agent
	agentMap := make(map[string]*domain.Agent)

	// Find agents for each capability and deduplicate
	for _, capability := range capabilities {
		agents, err := g.agentService.DiscoverAgentsByCapability(ctx, capability)
		if err != nil {
			return nil, fmt.Errorf("failed to discover agents for capability %s: %w", capability, err)
		}

		for _, agent := range agents {
			if _, exists := agentMap[agent.ID]; !exists {
				agentMap[agent.ID] = agent
				allAgents = append(allAgents, agent)
			}
		}
	}

	return allAgents, nil
}
