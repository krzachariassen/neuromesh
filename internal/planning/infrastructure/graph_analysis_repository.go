package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"neuromesh/internal/graph"
	"neuromesh/internal/planning/domain"
)

// GraphAnalysisRepository implements AnalysisRepository using Neo4j graph database
type GraphAnalysisRepository struct {
	graph graph.Graph
}

// NewGraphAnalysisRepository creates a new graph-based analysis repository
func NewGraphAnalysisRepository(graph graph.Graph) *GraphAnalysisRepository {
	return &GraphAnalysisRepository{
		graph: graph,
	}
}

// Store persists an Analysis in the graph with proper relationships to User/Conversation/Message
func (r *GraphAnalysisRepository) Store(ctx context.Context, analysis *domain.Analysis) error {
	// Convert required agents to JSON for storage
	requiredAgentsJSON, err := json.Marshal(analysis.RequiredAgents)
	if err != nil {
		return fmt.Errorf("failed to marshal required agents: %w", err)
	}

	// Create Analysis node properties
	properties := map[string]interface{}{
		"id":              analysis.ID,
		"request_id":      analysis.RequestID,
		"intent":          analysis.Intent,
		"category":        analysis.Category,
		"confidence":      analysis.Confidence,
		"required_agents": string(requiredAgentsJSON),
		"reasoning":       analysis.Reasoning,
		"timestamp":       analysis.Timestamp.UTC(),
		"created_at":      time.Now().UTC(),
	}

	// Create Analysis node
	err = r.graph.AddNode(ctx, "Analysis", analysis.ID, properties)
	if err != nil {
		return fmt.Errorf("failed to create Analysis node: %w", err)
	}

	// Create relationship from Message to Analysis (Message TRIGGERS_ANALYSIS Analysis)
	err = r.graph.AddEdge(ctx, "Message", analysis.RequestID, "Analysis", analysis.ID, "TRIGGERS_ANALYSIS", nil)
	if err != nil {
		return fmt.Errorf("failed to create Message->Analysis relationship: %w", err)
	}

	return nil
}

// GetByID retrieves an Analysis by its ID
func (r *GraphAnalysisRepository) GetByID(ctx context.Context, analysisID string) (*domain.Analysis, error) {
	nodes, err := r.graph.QueryNodes(ctx, "Analysis", map[string]interface{}{
		"id": analysisID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Analysis by ID: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("analysis not found with ID: %s", analysisID)
	}

	return r.nodeToAnalysis(nodes[0])
}

// GetByRequestID retrieves an Analysis by the request (message) ID
func (r *GraphAnalysisRepository) GetByRequestID(ctx context.Context, requestID string) (*domain.Analysis, error) {
	nodes, err := r.graph.QueryNodes(ctx, "Analysis", map[string]interface{}{
		"request_id": requestID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Analysis by request ID: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("analysis not found with request ID: %s", requestID)
	}

	return r.nodeToAnalysis(nodes[0])
}

// GetByUserID retrieves all analyses for a specific user, ordered by timestamp desc
func (r *GraphAnalysisRepository) GetByUserID(ctx context.Context, userID string, limit int) ([]*domain.Analysis, error) {
	// TODO: Implement complex relationship traversal when ExecuteCypher is available
	// For now, we'll use a simpler approach with QueryNodes
	// This should eventually use a Cypher query like:
	// MATCH (u:User {id: $userID})-[:HAS_SESSION]->(s:Session)-[:HAS_CONVERSATION]->(c:Conversation)-[:CONTAINS_MESSAGE]->(m:Message)-[:TRIGGERS_ANALYSIS]->(a:Analysis)
	// RETURN a ORDER BY a.timestamp DESC LIMIT $limit

	// Get all analyses and filter by user relationship later
	// This is not optimal but works with current interface
	nodes, err := r.graph.QueryNodes(ctx, "Analysis", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses: %w", err)
	}

	var analyses []*domain.Analysis
	for _, nodeData := range nodes {
		analysis, err := r.nodeToAnalysis(nodeData)
		if err != nil {
			// Log error and continue instead of breaking the entire query
			continue
		}
		analyses = append(analyses, analysis)
	}

	// TODO: Filter by user relationship
	// For now, return all analyses sorted and limited
	// This will be improved when ExecuteCypher is available for complex relationship traversal
	return r.sortAndLimit(analyses, limit), nil
}

// GetByConfidenceRange retrieves analyses within a confidence range
func (r *GraphAnalysisRepository) GetByConfidenceRange(ctx context.Context, minConfidence, maxConfidence int, limit int) ([]*domain.Analysis, error) {
	// Get all analyses and filter by confidence
	nodes, err := r.graph.QueryNodes(ctx, "Analysis", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses: %w", err)
	}

	var analyses []*domain.Analysis
	for _, nodeData := range nodes {
		analysis, err := r.nodeToAnalysis(nodeData)
		if err != nil {
			// Log error and continue instead of breaking the entire query
			continue
		}

		// Filter by confidence range
		if analysis.Confidence >= minConfidence && analysis.Confidence <= maxConfidence {
			analyses = append(analyses, analysis)
		}
	}

	// Sort by timestamp desc and apply limit
	return r.sortAndLimit(analyses, limit), nil
}

// GetByCategory retrieves analyses by category
func (r *GraphAnalysisRepository) GetByCategory(ctx context.Context, category string, limit int) ([]*domain.Analysis, error) {
	nodes, err := r.graph.QueryNodes(ctx, "Analysis", map[string]interface{}{
		"category": category,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses by category: %w", err)
	}

	var analyses []*domain.Analysis
	for _, nodeData := range nodes {
		analysis, err := r.nodeToAnalysis(nodeData)
		if err != nil {
			// Log error and continue instead of breaking the entire query
			continue
		}
		analyses = append(analyses, analysis)
	}

	// Sort by timestamp desc and apply limit
	return r.sortAndLimit(analyses, limit), nil
}

// sortAndLimit sorts analyses by timestamp desc and applies limit
func (r *GraphAnalysisRepository) sortAndLimit(analyses []*domain.Analysis, limit int) []*domain.Analysis {
	// Sort by timestamp descending (newest first)
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].Timestamp.After(analyses[j].Timestamp)
	})

	// Apply limit if specified and valid
	if limit > 0 && len(analyses) > limit {
		return analyses[:limit]
	}

	return analyses
}

// nodeToAnalysis converts a Neo4j node to Analysis domain object with improved error handling
func (r *GraphAnalysisRepository) nodeToAnalysis(nodeData map[string]interface{}) (*domain.Analysis, error) {
	// Validate required fields
	id, ok := nodeData["id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("invalid or missing analysis ID in node data")
	}

	requestID, ok := nodeData["request_id"].(string)
	if !ok || requestID == "" {
		return nil, fmt.Errorf("invalid or missing request_id in node data for analysis %s", id)
	}

	intent, _ := nodeData["intent"].(string)
	category, _ := nodeData["category"].(string)
	reasoning, _ := nodeData["reasoning"].(string)

	// Handle confidence with better type conversion
	var confidence int
	switch v := nodeData["confidence"].(type) {
	case int:
		confidence = v
	case int64:
		confidence = int(v)
	case float64:
		confidence = int(v)
	case string:
		// Try to parse string confidence (shouldn't happen in normal cases)
		if parsed, err := strconv.Atoi(v); err == nil {
			confidence = parsed
		} else {
			return nil, fmt.Errorf("invalid confidence value '%s' for analysis %s", v, id)
		}
	default:
		return nil, fmt.Errorf("unsupported confidence type %T for analysis %s", v, id)
	}

	// Validate confidence range
	if confidence < 0 || confidence > 100 {
		return nil, fmt.Errorf("confidence %d out of valid range (0-100) for analysis %s", confidence, id)
	}

	// Parse required agents JSON with better error handling
	var requiredAgents []string
	if requiredAgentsStr, ok := nodeData["required_agents"].(string); ok && requiredAgentsStr != "" {
		if err := json.Unmarshal([]byte(requiredAgentsStr), &requiredAgents); err != nil {
			return nil, fmt.Errorf("failed to parse required_agents JSON for analysis %s: %w", id, err)
		}
	}

	// Parse timestamp with better error handling
	var timestamp time.Time
	if timestampValue := nodeData["timestamp"]; timestampValue != nil {
		switch v := timestampValue.(type) {
		case string:
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				timestamp = parsed
			} else {
				return nil, fmt.Errorf("failed to parse timestamp '%s' for analysis %s: %w", v, id, err)
			}
		case time.Time:
			timestamp = v
		default:
			return nil, fmt.Errorf("unsupported timestamp type %T for analysis %s", v, id)
		}
	} else {
		return nil, fmt.Errorf("missing timestamp for analysis %s", id)
	}

	// Create Analysis domain object
	analysis := &domain.Analysis{
		ID:             id,
		RequestID:      requestID,
		Intent:         intent,
		Category:       category,
		Confidence:     confidence,
		RequiredAgents: requiredAgents,
		Reasoning:      reasoning,
		Timestamp:      timestamp,
	}

	return analysis, nil
}
