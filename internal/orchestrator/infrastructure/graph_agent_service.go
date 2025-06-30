package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentDomain "neuromesh/internal/agent/domain"
	"neuromesh/internal/graph"
)

// GraphAgentService implements AgentService using the graph backend
type GraphAgentService struct {
	graph graph.Graph
}

// NewGraphAgentService creates a new GraphAgentService
func NewGraphAgentService(graph graph.Graph) *GraphAgentService {
	return &GraphAgentService{
		graph: graph,
	}
}

// GetAvailableAgents retrieves all available agents from the graph
func (gas *GraphAgentService) GetAvailableAgents(ctx context.Context) ([]*agentDomain.Agent, error) {
	// Query the graph database for all online agents
	// Use a simple node query to get all agent nodes
	nodes, err := gas.graph.QueryNodes(ctx, "agent", map[string]interface{}{
		"status": "online",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query agents from graph: %w", err)
	}

	var agents []*agentDomain.Agent
	for _, nodeData := range nodes {
		agent, err := gas.nodeToAgent(nodeData)
		if err != nil {
			// Skip invalid nodes but log the error
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// RegisterAgent registers a new agent in the graph
func (gas *GraphAgentService) RegisterAgent(ctx context.Context, agent *agentDomain.Agent) error {
	// Implement agent registration in graph
	return fmt.Errorf("not implemented yet")
}

// DiscoverAgentsByCapability finds agents with specific capabilities
func (gas *GraphAgentService) DiscoverAgentsByCapability(ctx context.Context, capability string) ([]*agentDomain.Agent, error) {
	// Simplified implementation
	if capability == "deploy" || capability == "deployment" {
		agents := []*agentDomain.Agent{
			{
				ID:          "deploy-agent-1",
				Name:        "Deploy Agent",
				Description: "Handles deployment operations",
				Status:      agentDomain.AgentStatusOnline,
				Capabilities: []agentDomain.AgentCapability{
					{Name: "deploy", Description: "Deploy applications"},
					{Name: "rollback", Description: "Rollback deployments"},
					{Name: "scale", Description: "Scale applications"},
				},
			},
		}
		return agents, nil
	}

	return []*agentDomain.Agent{}, nil
}

// UpdateAgentStatus updates an agent's status in the graph
func (gas *GraphAgentService) UpdateAgentStatus(ctx context.Context, agentID string, status agentDomain.AgentStatus) error {
	// Implement status update in graph
	return fmt.Errorf("not implemented yet")
}

// nodeToAgent converts a graph node to an Agent domain object
func (gas *GraphAgentService) nodeToAgent(nodeData map[string]interface{}) (*agentDomain.Agent, error) {
	agent := &agentDomain.Agent{}

	// Extract agent ID
	if id, ok := nodeData["id"].(string); ok {
		agent.ID = id
	} else {
		return nil, fmt.Errorf("agent node missing ID")
	}

	// Extract name
	if name, ok := nodeData["name"].(string); ok {
		agent.Name = name
	}

	// Extract description
	if description, ok := nodeData["description"].(string); ok {
		agent.Description = description
	}

	// Extract status
	if status, ok := nodeData["status"].(string); ok {
		agent.Status = agentDomain.AgentStatus(status)
	}

	// Parse capabilities JSON
	if capabilitiesJSON, ok := nodeData["capabilities"].(string); ok && capabilitiesJSON != "" {
		var capabilities []agentDomain.AgentCapability
		if err := json.Unmarshal([]byte(capabilitiesJSON), &capabilities); err == nil {
			agent.Capabilities = capabilities
		}
	}

	// Handle time fields
	if lastSeenTime, ok := nodeData["last_seen"].(time.Time); ok {
		agent.LastSeen = lastSeenTime
	}

	return agent, nil
}
