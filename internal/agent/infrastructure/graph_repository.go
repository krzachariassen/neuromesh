package infrastructure

import (
	"context"
	"fmt"
	"time"

	"neuromesh/internal/agent/domain"
	"neuromesh/internal/graph"
)

// GraphAgentRepository implements the AgentRepository interface using the graph backend
type GraphAgentRepository struct {
	graph graph.Graph
}

// NewGraphAgentRepository creates a new graph-based agent repository
func NewGraphAgentRepository(g graph.Graph) *GraphAgentRepository {
	return &GraphAgentRepository{
		graph: g,
	}
}

// EnsureSchema ensures that the required schema for Agent domain is in place
func (r *GraphAgentRepository) EnsureSchema(ctx context.Context) error {
	// Define Agent domain schema requirements

	// Agent node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "agent", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for agent.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "agent", "name"); err != nil {
		return fmt.Errorf("failed to create index for agent.name: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "agent", "status"); err != nil {
		return fmt.Errorf("failed to create index for agent.status: %w", err)
	}

	// Capability node constraints and indexes
	// Check if constraint already exists before trying to create it
	hasConstraint, err := r.graph.HasUniqueConstraint(ctx, "capability", "name")
	if err != nil {
		return fmt.Errorf("failed to check existing capability.name constraint: %w", err)
	}

	if !hasConstraint {
		// Check if there's an existing index that needs to be dropped first
		hasIndex, err := r.graph.HasIndex(ctx, "capability", "name")
		if err != nil {
			return fmt.Errorf("failed to check existing capability.name index: %w", err)
		}

		if hasIndex {
			// Drop the existing index so we can create the constraint
			if err := r.dropIndex(ctx, "capability", "name"); err != nil {
				return fmt.Errorf("failed to drop existing capability.name index: %w", err)
			}
		}

		// Now create the unique constraint
		if err := r.graph.CreateUniqueConstraint(ctx, "capability", "name"); err != nil {
			return fmt.Errorf("failed to create unique constraint for capability.name: %w", err)
		}
	}

	// Ensure HAS_CAPABILITY relationship type exists in the schema
	// Check if the relationship type already exists
	hasRelType, err := r.graph.HasRelationshipType(ctx, "HAS_CAPABILITY")
	if err != nil {
		return fmt.Errorf("failed to check HAS_CAPABILITY relationship type: %w", err)
	}

	if !hasRelType {
		// Create schema nodes to define the relationship type permanently
		schemaAgentID := "schema_agent"
		schemaCapabilityID := "schema_capability"

		// Create schema nodes if they don't exist
		if err := r.graph.AddNode(ctx, "agent", schemaAgentID, map[string]interface{}{
			"id":     schemaAgentID,
			"name":   "Schema Definition Agent",
			"status": "schema",
		}); err != nil {
			// Node might already exist, check if it's actually an error
			existingNode, getErr := r.graph.GetNode(ctx, "agent", schemaAgentID)
			if getErr != nil || existingNode == nil {
				return fmt.Errorf("failed to create schema agent node: %w", err)
			}
		}

		if err := r.graph.AddNode(ctx, "capability", schemaCapabilityID, map[string]interface{}{
			"name": schemaCapabilityID,
		}); err != nil {
			// Node might already exist, check if it's actually an error
			existingNode, getErr := r.graph.GetNode(ctx, "capability", schemaCapabilityID)
			if getErr != nil || existingNode == nil {
				return fmt.Errorf("failed to create schema capability node: %w", err)
			}
		}

		// Create the schema relationship to register the type permanently
		if err := r.graph.AddEdge(ctx, "agent", schemaAgentID, "capability", schemaCapabilityID, "HAS_CAPABILITY", map[string]interface{}{
			"schema":      true,
			"description": "Schema definition relationship",
		}); err != nil {
			return fmt.Errorf("failed to create schema HAS_CAPABILITY relationship: %w", err)
		}
	}

	return nil
}

// Create persists a new agent to the graph
func (r *GraphAgentRepository) Create(ctx context.Context, agent *domain.Agent) error {
	if err := agent.Validate(); err != nil {
		return fmt.Errorf("invalid agent: %w", err)
	}

	// Convert domain model to graph data
	data := agent.ToMap()

	// Store in graph with proper node type
	nodeID := fmt.Sprintf("agent:%s", agent.ID)

	// Create the agent node
	if err := r.graph.AddNode(ctx, "agent", nodeID, data); err != nil {
		return fmt.Errorf("failed to create agent node: %w", err)
	}

	// Add capability relationships
	for _, capability := range agent.Capabilities {
		capabilityNodeID := fmt.Sprintf("capability:%s:%s", agent.ID, capability.Name)
		capabilityData := map[string]interface{}{
			"name":        capability.Name,
			"description": capability.Description,
			"parameters":  capability.Parameters,
		}

		// Create capability node
		if err := r.graph.AddNode(ctx, "capability", capabilityNodeID, capabilityData); err != nil {
			return fmt.Errorf("failed to create capability node: %w", err)
		}

		// Create relationship
		if err := r.graph.AddEdge(ctx, "agent", nodeID, "capability", capabilityNodeID, "HAS_CAPABILITY", nil); err != nil {
			return fmt.Errorf("failed to create capability relationship: %w", err)
		}
	}

	return nil
}

// GetByID retrieves an agent by its ID from the graph
func (r *GraphAgentRepository) GetByID(ctx context.Context, id string) (*domain.Agent, error) {
	if id == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	nodeID := fmt.Sprintf("agent:%s", id)

	// Get agent node data
	node, err := r.graph.GetNode(ctx, "agent", nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent node: %w", err)
	}

	if node == nil {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	// Get capabilities using graph traversal
	capabilities, err := r.getAgentCapabilities(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent capabilities: %w", err)
	}

	// Create a map with node properties and capabilities
	nodeData := make(map[string]interface{})
	for k, v := range node {
		nodeData[k] = v
	}
	nodeData["capabilities"] = capabilities

	// Convert from graph data to domain model
	agent, err := domain.AgentFromMap(nodeData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize agent: %w", err)
	}

	return agent, nil
}

// GetAll retrieves all agents from the graph
func (r *GraphAgentRepository) GetAll(ctx context.Context) ([]*domain.Agent, error) {
	// Get all agent nodes
	nodes, err := r.graph.QueryNodes(ctx, "agent", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent nodes: %w", err)
	}

	agents := make([]*domain.Agent, 0, len(nodes))
	for _, node := range nodes {
		// Extract agent ID from node data
		agentID, ok := node["id"].(string)
		if !ok {
			continue // Skip nodes without valid ID
		}

		// Get capabilities for this agent
		nodeID := fmt.Sprintf("agent:%s", agentID)
		capabilities, err := r.getAgentCapabilities(ctx, nodeID)
		if err != nil {
			continue // Skip agents with capability errors
		}

		// Create map with node properties and capabilities
		nodeData := make(map[string]interface{})
		for k, v := range node {
			nodeData[k] = v
		}
		nodeData["capabilities"] = capabilities

		// Convert to domain model
		agent, err := domain.AgentFromMap(nodeData)
		if err != nil {
			continue // Skip invalid agents
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// GetByStatus retrieves agents by their status
func (r *GraphAgentRepository) GetByStatus(ctx context.Context, status domain.AgentStatus) ([]*domain.Agent, error) {
	// Get all agents and filter by status
	allAgents, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var filteredAgents []*domain.Agent
	for _, agent := range allAgents {
		if agent.Status == status {
			filteredAgents = append(filteredAgents, agent)
		}
	}

	return filteredAgents, nil
}

// GetByCapability retrieves agents that have a specific capability
func (r *GraphAgentRepository) GetByCapability(ctx context.Context, capabilityName string) ([]*domain.Agent, error) {
	// Get all agents and filter by capability
	allAgents, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var filteredAgents []*domain.Agent
	for _, agent := range allAgents {
		for _, capability := range agent.Capabilities {
			if capability.Name == capabilityName {
				filteredAgents = append(filteredAgents, agent)
				break
			}
		}
	}

	return filteredAgents, nil
}

// Update updates an existing agent in the graph
func (r *GraphAgentRepository) Update(ctx context.Context, agent *domain.Agent) error {
	if err := agent.Validate(); err != nil {
		return fmt.Errorf("invalid agent: %w", err)
	}

	nodeID := fmt.Sprintf("agent:%s", agent.ID)

	// Check if agent exists
	existing, err := r.graph.GetNode(ctx, "agent", nodeID)
	if err != nil {
		return fmt.Errorf("failed to check existing agent: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	// Update agent node
	data := agent.ToMap()
	if err := r.graph.UpdateNode(ctx, "agent", nodeID, data); err != nil {
		return fmt.Errorf("failed to update agent node: %w", err)
	}

	// TODO: Handle capability updates (remove old, add new)
	// For now, we'll just update the main agent properties

	return nil
}

// Delete removes an agent from the graph
func (r *GraphAgentRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	nodeID := fmt.Sprintf("agent:%s", id)

	// Get and remove capability nodes and edges
	edges, err := r.graph.GetEdges(ctx, "agent", nodeID)
	if err == nil { // Don't fail if no edges found
		for _, edge := range edges {
			// Check if this is a HAS_CAPABILITY edge
			if edgeType, ok := edge["type"].(string); ok && edgeType == "HAS_CAPABILITY" {
				// Extract target capability node ID
				if targetID, ok := edge["target_id"].(string); ok {
					// Delete capability node
					if err := r.graph.DeleteNode(ctx, "capability", targetID); err != nil {
						// Log but don't fail
					}
				}
			}
		}
	}

	// Delete the agent node (this will also remove associated edges)
	if err := r.graph.DeleteNode(ctx, "agent", nodeID); err != nil {
		return fmt.Errorf("failed to delete agent node: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of an agent
func (r *GraphAgentRepository) UpdateStatus(ctx context.Context, id string, status domain.AgentStatus) error {
	if id == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s", status)
	}

	nodeID := fmt.Sprintf("agent:%s", id)

	// Update just the status property
	properties := map[string]interface{}{
		"status": string(status),
	}

	if err := r.graph.UpdateNode(ctx, "agent", nodeID, properties); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	return nil
}

// UpdateLastSeen updates the last seen timestamp of an agent
func (r *GraphAgentRepository) UpdateLastSeen(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	nodeID := fmt.Sprintf("agent:%s", id)

	// Update just the last_seen property
	properties := map[string]interface{}{
		"last_seen": time.Now().Unix(),
	}

	if err := r.graph.UpdateNode(ctx, "agent", nodeID, properties); err != nil {
		return fmt.Errorf("failed to update agent last seen: %w", err)
	}

	return nil
}

// getAgentCapabilities retrieves capabilities for an agent
func (r *GraphAgentRepository) getAgentCapabilities(ctx context.Context, agentNodeID string) ([]interface{}, error) {
	// Get edges from the agent node
	edges, err := r.graph.GetEdges(ctx, "agent", agentNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get capability edges: %w", err)
	}

	var capabilities []interface{}
	for _, edge := range edges {
		// Check if this is a HAS_CAPABILITY edge
		edgeType, ok := edge["type"].(string)
		if !ok || edgeType != "HAS_CAPABILITY" {
			continue
		}

		// Get target capability node ID
		targetID, ok := edge["target_id"].(string)
		if !ok {
			continue
		}

		// Get capability node data
		capabilityNode, err := r.graph.GetNode(ctx, "capability", targetID)
		if err != nil {
			continue // Skip if capability node is not found
		}

		// Convert to capability data
		capabilityData := map[string]interface{}{
			"name":        capabilityNode["name"],
			"description": capabilityNode["description"],
			"parameters":  capabilityNode["parameters"],
		}
		capabilities = append(capabilities, capabilityData)
	}

	return capabilities, nil
}

// Helper methods for testing schema verification
// These methods are used by tests to verify that schema elements exist in the database

// dropIndex removes an index on the specified node label and property
func (r *GraphAgentRepository) dropIndex(ctx context.Context, nodeLabel, property string) error {
	return r.graph.DropIndex(ctx, nodeLabel, property)
}

// hasUniqueConstraint checks if a unique constraint exists on the specified node label and property
func (r *GraphAgentRepository) hasUniqueConstraint(ctx context.Context, nodeLabel, property string) (bool, error) {
	return r.graph.HasUniqueConstraint(ctx, nodeLabel, property)
}

// hasIndex checks if an index exists on the specified node label and property
func (r *GraphAgentRepository) hasIndex(ctx context.Context, nodeLabel, property string) (bool, error) {
	return r.graph.HasIndex(ctx, nodeLabel, property)
}

// hasRelationshipType checks if a relationship type is defined in the graph
func (r *GraphAgentRepository) hasRelationshipType(ctx context.Context, relationshipType string) (bool, error) {
	return r.graph.HasRelationshipType(ctx, relationshipType)
}
