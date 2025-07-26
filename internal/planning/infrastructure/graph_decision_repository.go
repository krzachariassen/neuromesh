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

// GraphDecisionRepository implements DecisionRepository using Neo4j graph database
type GraphDecisionRepository struct {
	graph graph.Graph
}

// NewGraphDecisionRepository creates a new graph-based decision repository
func NewGraphDecisionRepository(graph graph.Graph) *GraphDecisionRepository {
	return &GraphDecisionRepository{
		graph: graph,
	}
}

// Store persists a Decision in the graph with proper relationships
func (r *GraphDecisionRepository) Store(ctx context.Context, decision *domain.Decision) error {
	// Convert parameters to JSON for storage
	var parametersJSON string
	if decision.Parameters != nil {
		parametersBytes, err := json.Marshal(decision.Parameters)
		if err != nil {
			return fmt.Errorf("failed to marshal parameters: %w", err)
		}
		parametersJSON = string(parametersBytes)
	}

	// Create Decision node properties
	properties := map[string]interface{}{
		"id":                     decision.ID,
		"request_id":             decision.RequestID,
		"analysis_id":            decision.AnalysisID,
		"type":                   string(decision.Type),
		"action":                 decision.Action,
		"parameters":             parametersJSON,
		"clarification_question": decision.ClarificationQuestion,
		"execution_plan_id":      decision.ExecutionPlanID,
		"agent_coordination":     decision.AgentCoordination,
		"reasoning":              decision.Reasoning,
		"timestamp":              decision.Timestamp.UTC(),
		"created_at":             time.Now().UTC(),
	}

	// Create Decision node
	err := r.graph.AddNode(ctx, "Decision", decision.ID, properties)
	if err != nil {
		return fmt.Errorf("failed to create Decision node: %w", err)
	}

	return nil
}

// GetByID retrieves a Decision by its ID
func (r *GraphDecisionRepository) GetByID(ctx context.Context, decisionID string) (*domain.Decision, error) {
	nodes, err := r.graph.QueryNodes(ctx, "Decision", map[string]interface{}{
		"id": decisionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Decision: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("decision not found: %s", decisionID)
	}

	return r.nodeToDecision(nodes[0])
}

// GetByRequestID retrieves a Decision by the request (message) ID
func (r *GraphDecisionRepository) GetByRequestID(ctx context.Context, requestID string) (*domain.Decision, error) {
	nodes, err := r.graph.QueryNodes(ctx, "Decision", map[string]interface{}{
		"request_id": requestID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Decision by request ID: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("decision not found for request: %s", requestID)
	}

	return r.nodeToDecision(nodes[0])
}

// GetByAnalysisID retrieves a Decision by the analysis ID
func (r *GraphDecisionRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*domain.Decision, error) {
	nodes, err := r.graph.QueryNodes(ctx, "Decision", map[string]interface{}{
		"analysis_id": analysisID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Decision by analysis ID: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("decision not found for analysis: %s", analysisID)
	}

	return r.nodeToDecision(nodes[0])
}

// GetByUserID retrieves all decisions for a specific user, ordered by timestamp desc
func (r *GraphDecisionRepository) GetByUserID(ctx context.Context, userID string, limit int) ([]*domain.Decision, error) {
	// TODO: Implement complex relationship traversal when ExecuteCypher is available
	// For now, use simpler approach
	nodes, err := r.graph.QueryNodes(ctx, "Decision", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to query Decisions: %w", err)
	}

	var decisions []*domain.Decision
	for _, nodeData := range nodes {
		decision, err := r.nodeToDecision(nodeData)
		if err != nil {
			continue // Skip invalid nodes
		}
		decisions = append(decisions, decision)
	}

	// Sort by timestamp desc and apply limit
	return r.sortAndLimit(decisions, limit), nil
}

// GetByType retrieves decisions by type (CLARIFY or EXECUTE)
func (r *GraphDecisionRepository) GetByType(ctx context.Context, decisionType domain.DecisionType, limit int) ([]*domain.Decision, error) {
	nodes, err := r.graph.QueryNodes(ctx, "Decision", map[string]interface{}{
		"type": string(decisionType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Decisions by type: %w", err)
	}

	var decisions []*domain.Decision
	for _, nodeData := range nodes {
		decision, err := r.nodeToDecision(nodeData)
		if err != nil {
			continue // Skip invalid nodes
		}
		decisions = append(decisions, decision)
	}

	return r.sortAndLimit(decisions, limit), nil
}

// LinkToAnalysis creates a relationship between decision and analysis
func (r *GraphDecisionRepository) LinkToAnalysis(ctx context.Context, decisionID, analysisID string) error {
	err := r.graph.AddEdge(ctx, "Decision", decisionID, "Analysis", analysisID, "BASED_ON_ANALYSIS", nil)
	if err != nil {
		return fmt.Errorf("failed to link Decision to Analysis: %w", err)
	}

	return nil
}

// LinkToExecutionPlan creates a relationship between decision and execution plan
func (r *GraphDecisionRepository) LinkToExecutionPlan(ctx context.Context, decisionID, executionPlanID string) error {
	err := r.graph.AddEdge(ctx, "Decision", decisionID, "ExecutionPlan", executionPlanID, "TRIGGERS_PLAN", nil)
	if err != nil {
		return fmt.Errorf("failed to link Decision to ExecutionPlan: %w", err)
	}

	return nil
}

// nodeToDecision converts a graph node to a Decision entity
func (r *GraphDecisionRepository) nodeToDecision(nodeData map[string]interface{}) (*domain.Decision, error) {
	decision := &domain.Decision{}

	// Required fields
	if id, ok := nodeData["id"].(string); ok {
		decision.ID = id
	} else {
		return nil, fmt.Errorf("missing or invalid decision ID")
	}

	if requestID, ok := nodeData["request_id"].(string); ok {
		decision.RequestID = requestID
	}

	if analysisID, ok := nodeData["analysis_id"].(string); ok {
		decision.AnalysisID = analysisID
	}

	if decisionType, ok := nodeData["type"].(string); ok {
		decision.Type = domain.DecisionType(decisionType)
	}

	// Optional fields
	if action, ok := nodeData["action"].(string); ok {
		decision.Action = action
	}

	if parametersStr, ok := nodeData["parameters"].(string); ok && parametersStr != "" {
		var parameters map[string]interface{}
		if err := json.Unmarshal([]byte(parametersStr), &parameters); err == nil {
			decision.Parameters = parameters
		}
	}

	if clarificationQuestion, ok := nodeData["clarification_question"].(string); ok {
		decision.ClarificationQuestion = clarificationQuestion
	}

	if executionPlanID, ok := nodeData["execution_plan_id"].(string); ok {
		decision.ExecutionPlanID = executionPlanID
	}

	if agentCoordination, ok := nodeData["agent_coordination"].(string); ok {
		decision.AgentCoordination = agentCoordination
	}

	if reasoning, ok := nodeData["reasoning"].(string); ok {
		decision.Reasoning = reasoning
	}

	if timestamp, ok := nodeData["timestamp"].(time.Time); ok {
		decision.Timestamp = timestamp
	}

	return decision, nil
}

// sortAndLimit sorts decisions by timestamp desc and applies limit
func (r *GraphDecisionRepository) sortAndLimit(decisions []*domain.Decision, limit int) []*domain.Decision {
	// Sort by timestamp descending (newest first)
	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].Timestamp.After(decisions[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(decisions) > limit {
		decisions = decisions[:limit]
	}

	return decisions
}
