package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"neuromesh/internal/graph"
	"neuromesh/internal/planning/domain"
)

// GraphPlanningRepository implements PlanningResultRepository using Neo4j graph database
// This replaces both GraphDecisionRepository and GraphAnalysisRepository with unified planning storage
type GraphPlanningRepository struct {
	graph graph.Graph
}

// NewGraphPlanningRepository creates a new graph-based planning repository
func NewGraphPlanningRepository(graph graph.Graph) *GraphPlanningRepository {
	return &GraphPlanningRepository{
		graph: graph,
	}
}

// EnsureSchema ensures that the required schema for PlanningResult domain is in place
func (r *GraphPlanningRepository) EnsureSchema(ctx context.Context) error {
	// PlanningResult node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "planning_result", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for planning_result.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "planning_result", "request_id"); err != nil {
		return fmt.Errorf("failed to create index for planning_result.request_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "planning_result", "type"); err != nil {
		return fmt.Errorf("failed to create index for planning_result.type: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "planning_result", "confidence"); err != nil {
		return fmt.Errorf("failed to create index for planning_result.confidence: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "planning_result", "timestamp"); err != nil {
		return fmt.Errorf("failed to create index for planning_result.timestamp: %w", err)
	}

	return nil
}

// Store persists a planning result in the graph with proper relationships
func (r *GraphPlanningRepository) Store(ctx context.Context, result *domain.PlanningResult) error {
	// Convert agent arrays to JSON for storage
	availableAgentsJSON, err := json.Marshal(result.AvailableAgents)
	if err != nil {
		return fmt.Errorf("failed to marshal available agents: %w", err)
	}

	requiredAgentsJSON, err := json.Marshal(result.RequiredAgents)
	if err != nil {
		return fmt.Errorf("failed to marshal required agents: %w", err)
	}

	agentGapJSON, err := json.Marshal(result.AgentGap)
	if err != nil {
		return fmt.Errorf("failed to marshal agent gap: %w", err)
	}

	// Create PlanningResult node properties
	properties := map[string]interface{}{
		"id":                     result.ID,
		"request_id":             result.RequestID,
		"type":                   string(result.Type),
		"available_agents":       string(availableAgentsJSON),
		"required_agents":        string(requiredAgentsJSON),
		"agent_gap":              string(agentGapJSON),
		"execution_plan_id":      result.ExecutionPlanID,
		"clarification_question": result.ClarificationQuestion,
		"direct_response":        result.DirectResponse,
		"intent":                 result.Intent,
		"category":               result.Category,
		"confidence":             result.Confidence,
		"reasoning":              result.Reasoning,
		"timestamp":              result.Timestamp.UTC(),
		"created_at":             time.Now().UTC(),
	}

	// Create PlanningResult node
	err = r.graph.AddNode(ctx, "planning_result", result.ID, properties)
	if err != nil {
		return fmt.Errorf("failed to create PlanningResult node: %w", err)
	}

	return nil
}

// GetByID retrieves a planning result by its ID
func (r *GraphPlanningRepository) GetByID(ctx context.Context, id string) (*domain.PlanningResult, error) {
	nodes, err := r.graph.QueryNodes(ctx, "planning_result", map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query PlanningResult: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("planning result not found: %s", id)
	}

	return r.nodeToPlanningResult(nodes[0])
}

// GetByRequestID retrieves planning results for a specific request
func (r *GraphPlanningRepository) GetByRequestID(ctx context.Context, requestID string) ([]*domain.PlanningResult, error) {
	nodes, err := r.graph.QueryNodes(ctx, "planning_result", map[string]interface{}{
		"request_id": requestID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query PlanningResults by request ID: %w", err)
	}

	var results []*domain.PlanningResult
	for _, nodeData := range nodes {
		result, err := r.nodeToPlanningResult(nodeData)
		if err != nil {
			continue // Skip invalid nodes
		}
		results = append(results, result)
	}

	// Sort by timestamp desc (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	return results, nil
}

// Update updates an existing planning result
func (r *GraphPlanningRepository) Update(ctx context.Context, result *domain.PlanningResult) error {
	// Convert agent arrays to JSON for storage
	availableAgentsJSON, err := json.Marshal(result.AvailableAgents)
	if err != nil {
		return fmt.Errorf("failed to marshal available agents: %w", err)
	}

	requiredAgentsJSON, err := json.Marshal(result.RequiredAgents)
	if err != nil {
		return fmt.Errorf("failed to marshal required agents: %w", err)
	}

	agentGapJSON, err := json.Marshal(result.AgentGap)
	if err != nil {
		return fmt.Errorf("failed to marshal agent gap: %w", err)
	}

	// Create update properties
	properties := map[string]interface{}{
		"type":                   string(result.Type),
		"available_agents":       string(availableAgentsJSON),
		"required_agents":        string(requiredAgentsJSON),
		"agent_gap":              string(agentGapJSON),
		"execution_plan_id":      result.ExecutionPlanID,
		"clarification_question": result.ClarificationQuestion,
		"direct_response":        result.DirectResponse,
		"intent":                 result.Intent,
		"category":               result.Category,
		"confidence":             result.Confidence,
		"reasoning":              result.Reasoning,
		"timestamp":              result.Timestamp.UTC(),
		"updated_at":             time.Now().UTC(),
	}

	err = r.graph.UpdateNode(ctx, "planning_result", result.ID, properties)
	if err != nil {
		return fmt.Errorf("failed to update PlanningResult: %w", err)
	}

	return nil
}

// Delete removes a planning result
func (r *GraphPlanningRepository) Delete(ctx context.Context, id string) error {
	err := r.graph.DeleteNode(ctx, "planning_result", id)
	if err != nil {
		return fmt.Errorf("failed to delete PlanningResult: %w", err)
	}

	return nil
}

// LinkToRequest links a planning result to a request
func (r *GraphPlanningRepository) LinkToRequest(ctx context.Context, planningResultID, requestID string) error {
	relationshipProps := map[string]interface{}{
		"created_at": time.Now().UTC(),
	}

	err := r.graph.AddEdge(ctx, "request", requestID, "planning_result", planningResultID, "HAS_PLANNING", relationshipProps)
	if err != nil {
		return fmt.Errorf("failed to create HAS_PLANNING relationship: %w", err)
	}

	return nil
}

// LinkToExecutionPlan links a planning result to an execution plan
func (r *GraphPlanningRepository) LinkToExecutionPlan(ctx context.Context, planningResultID, executionPlanID string) error {
	relationshipProps := map[string]interface{}{
		"created_at": time.Now().UTC(),
	}

	err := r.graph.AddEdge(ctx, "planning_result", planningResultID, "execution_plan", executionPlanID, "CREATES_PLAN", relationshipProps)
	if err != nil {
		return fmt.Errorf("failed to create CREATES_PLAN relationship: %w", err)
	}

	return nil
}

// LinkToConversation links a planning result to a conversation
func (r *GraphPlanningRepository) LinkToConversation(ctx context.Context, planningResultID, conversationID string) error {
	relationshipProps := map[string]interface{}{
		"created_at": time.Now().UTC(),
	}

	err := r.graph.AddEdge(ctx, "Conversation", conversationID, "planning_result", planningResultID, "HAS_PLANNING", relationshipProps)
	if err != nil {
		return fmt.Errorf("failed to create HAS_PLANNING relationship: %w", err)
	}

	return nil
}

// nodeToPlanningResult converts a graph node to a PlanningResult entity
func (r *GraphPlanningRepository) nodeToPlanningResult(nodeData map[string]interface{}) (*domain.PlanningResult, error) {
	result := &domain.PlanningResult{}

	// Required fields
	if id, ok := nodeData["id"].(string); ok {
		result.ID = id
	} else {
		return nil, fmt.Errorf("missing or invalid planning result ID")
	}

	if requestID, ok := nodeData["request_id"].(string); ok {
		result.RequestID = requestID
	}

	if planningType, ok := nodeData["type"].(string); ok {
		result.Type = domain.PlanningType(planningType)
	}

	// Parse agent arrays from JSON
	if availableAgentsStr, ok := nodeData["available_agents"].(string); ok && availableAgentsStr != "" {
		if err := json.Unmarshal([]byte(availableAgentsStr), &result.AvailableAgents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal available agents: %w", err)
		}
	}

	if requiredAgentsStr, ok := nodeData["required_agents"].(string); ok && requiredAgentsStr != "" {
		if err := json.Unmarshal([]byte(requiredAgentsStr), &result.RequiredAgents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal required agents: %w", err)
		}
	}

	if agentGapStr, ok := nodeData["agent_gap"].(string); ok && agentGapStr != "" {
		if err := json.Unmarshal([]byte(agentGapStr), &result.AgentGap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal agent gap: %w", err)
		}
	}

	// Optional fields
	if executionPlanID, ok := nodeData["execution_plan_id"].(string); ok {
		result.ExecutionPlanID = executionPlanID
	}

	if clarificationQuestion, ok := nodeData["clarification_question"].(string); ok {
		result.ClarificationQuestion = clarificationQuestion
	}

	if directResponse, ok := nodeData["direct_response"].(string); ok {
		result.DirectResponse = directResponse
	}

	if intent, ok := nodeData["intent"].(string); ok {
		result.Intent = intent
	}

	if category, ok := nodeData["category"].(string); ok {
		result.Category = category
	}

	if confidence, ok := nodeData["confidence"].(int); ok {
		result.Confidence = confidence
	} else if confidence, ok := nodeData["confidence"].(int64); ok {
		result.Confidence = int(confidence)
	}

	if reasoning, ok := nodeData["reasoning"].(string); ok {
		result.Reasoning = reasoning
	}

	if timestamp, ok := nodeData["timestamp"].(time.Time); ok {
		result.Timestamp = timestamp
	}

	return result, nil
}
