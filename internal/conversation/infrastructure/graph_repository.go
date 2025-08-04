package infrastructure

import (
	"context"
	"fmt"

	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ConversationGraphRepository implements ConversationGraphService using Neo4j
// This is an infrastructure implementation that the API layer doesn't know about
type ConversationGraphRepository struct {
	graph  graph.Graph
	logger logging.Logger
}

// NewConversationGraphRepository creates a new conversation graph repository
func NewConversationGraphRepository(g graph.Graph, logger logging.Logger) *ConversationGraphRepository {
	return &ConversationGraphRepository{
		graph:  g,
		logger: logger,
	}
}

// GetConversationGraph retrieves real graph data for a conversation from Neo4j
func (r *ConversationGraphRepository) GetConversationGraph(ctx context.Context, conversationID string) (*domain.GraphData, error) {
	r.logger.Info("Fetching conversation graph", "conversationID", conversationID)

	// Cast to Neo4jGraph to access the driver
	neo4jGraph, ok := r.graph.(*graph.Neo4jGraph)
	if !ok {
		return nil, fmt.Errorf("graph service is not a Neo4j graph")
	}

	// Get a Neo4j session
	session := neo4jGraph.Driver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Query Neo4j for all nodes and relationships related to this conversation
	// Following our established graph schema: User -> Conversation -> Decision -> ExecutionPlan
	// IMPORTANT: Include cross-domain relationships (HAS_EXECUTION_PLAN)
	query := `
		MATCH (c:Conversation {id: $conversationId})
		OPTIONAL MATCH (c)-[r1:CREATED_BY]-(u:User)
		OPTIONAL MATCH (c)-[r2:HAS_DECISION]-(d:Decision)
		OPTIONAL MATCH (d)-[r3:GENERATED]-(ep:ExecutionPlan)
		OPTIONAL MATCH (ep)-[r4:HAS_STEP]-(es:ExecutionStep)
		OPTIONAL MATCH (es)-[r5:EXECUTED_BY]-(a:Agent)
		OPTIONAL MATCH (a)-[r6:PRODUCED]-(result:Result)
		OPTIONAL MATCH (c)-[r7:CONTAINS]-(m:ConversationMessage)
		OPTIONAL MATCH (c)-[r8:HAS_EXECUTION_PLAN]-(linkedPlan:ExecutionPlan)
		
		RETURN c, u, d, ep, es, a, result, m, linkedPlan, r1, r2, r3, r4, r5, r6, r7, r8
	`

	params := map[string]interface{}{
		"conversationId": conversationID,
	}

	graphData := &domain.GraphData{
		Nodes: []domain.GraphNode{},
		Edges: []domain.GraphEdge{},
	}

	// Execute the query
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		// Track processed nodes and edges to avoid duplicates
		processedNodes := make(map[string]bool)
		processedEdges := make(map[string]bool)

		// Process each result record
		for result.Next(ctx) {
			record := result.Record()

			// Process nodes - get values by field name
			nodeFields := []string{"c", "u", "d", "ep", "es", "a", "result", "m", "linkedPlan"}
			for _, field := range nodeFields {
				if value, found := record.Get(field); found && value != nil {
					node := r.convertNeo4jNodeToGraphNode(value, conversationID)
					if node != nil && !processedNodes[node.ID] {
						graphData.Nodes = append(graphData.Nodes, *node)
						processedNodes[node.ID] = true
					}
				}
			}

			// Process relationships
			relFields := []string{"r1", "r2", "r3", "r4", "r5", "r6", "r7", "r8"}
			for _, field := range relFields {
				if value, found := record.Get(field); found && value != nil {
					edge := r.convertNeo4jRelationshipToGraphEdge(value)
					if edge != nil && !processedEdges[edge.ID] {
						graphData.Edges = append(graphData.Edges, *edge)
						processedEdges[edge.ID] = true
					}
				}
			}
		}

		return nil, result.Err()
	})

	if err != nil {
		r.logger.Error("Failed to execute Neo4j graph query", err, "conversationID", conversationID)
		return nil, fmt.Errorf("failed to query graph: %w", err)
	}

	r.logger.Info("Successfully fetched graph data",
		"conversationID", conversationID,
		"nodeCount", len(graphData.Nodes),
		"edgeCount", len(graphData.Edges))

	return graphData, nil
}

// GetConversationSubgraph retrieves a filtered subgraph for a conversation
func (r *ConversationGraphRepository) GetConversationSubgraph(ctx context.Context, conversationID string, nodeTypes []string) (*domain.GraphData, error) {
	// For now, delegate to the full graph and filter - can optimize later
	fullGraph, err := r.GetConversationGraph(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Filter nodes by type
	filteredNodes := []domain.GraphNode{}
	nodeIDs := make(map[string]bool)

	for _, node := range fullGraph.Nodes {
		for _, nodeType := range nodeTypes {
			if node.Type == nodeType {
				filteredNodes = append(filteredNodes, node)
				nodeIDs[node.ID] = true
				break
			}
		}
	}

	// Filter edges to only include those between filtered nodes
	filteredEdges := []domain.GraphEdge{}
	for _, edge := range fullGraph.Edges {
		if nodeIDs[edge.Source] && nodeIDs[edge.Target] {
			filteredEdges = append(filteredEdges, edge)
		}
	}

	return &domain.GraphData{
		Nodes: filteredNodes,
		Edges: filteredEdges,
	}, nil
}

// GetGraphStats returns statistics about the conversation graph
func (r *ConversationGraphRepository) GetGraphStats(ctx context.Context, conversationID string) (map[string]interface{}, error) {
	graphData, err := r.GetConversationGraph(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Calculate basic statistics
	nodeTypeCount := make(map[string]int)
	edgeTypeCount := make(map[string]int)

	for _, node := range graphData.Nodes {
		nodeTypeCount[node.Type]++
	}

	for _, edge := range graphData.Edges {
		edgeTypeCount[edge.Type]++
	}

	return map[string]interface{}{
		"total_nodes":     len(graphData.Nodes),
		"total_edges":     len(graphData.Edges),
		"node_types":      nodeTypeCount,
		"edge_types":      edgeTypeCount,
		"conversation_id": conversationID,
	}, nil
}

// convertNeo4jNodeToGraphNode converts a Neo4j node to domain graph node
func (r *ConversationGraphRepository) convertNeo4jNodeToGraphNode(nodeData interface{}, conversationID string) *domain.GraphNode {
	// Handle Neo4j node type
	if neo4jNode, ok := nodeData.(neo4j.Node); ok {
		// Extract properties from the Neo4j node
		properties := neo4jNode.Props

		// Get the node ID - always use Neo4j internal ID for consistency with relationships
		nodeID := fmt.Sprintf("node-%d", neo4jNode.Id)

		// Determine node type from Neo4j labels
		nodeType := r.determineNodeTypeFromLabels(neo4jNode.Labels)
		if nodeType == "" {
			r.logger.Warn("Unable to determine node type from labels", "labels", neo4jNode.Labels)
			return nil
		}

		// Calculate layout positions based on node type
		position := r.calculateNodePosition(nodeType, nodeID, conversationID)

		// Add Neo4j metadata to properties
		enrichedProperties := make(map[string]interface{})
		for k, v := range properties {
			enrichedProperties[k] = v
		}
		enrichedProperties["neo4j_id"] = neo4jNode.Id
		enrichedProperties["labels"] = neo4jNode.Labels

		return &domain.GraphNode{
			ID:       nodeID,
			Type:     nodeType,
			Data:     enrichedProperties,
			Position: position,
		}
	}

	r.logger.Warn("Invalid node data format from Neo4j", "type", fmt.Sprintf("%T", nodeData))
	return nil
}

// convertNeo4jRelationshipToGraphEdge converts a Neo4j relationship to domain graph edge
func (r *ConversationGraphRepository) convertNeo4jRelationshipToGraphEdge(relData interface{}) *domain.GraphEdge {
	// Handle Neo4j relationship type
	if neo4jRel, ok := relData.(neo4j.Relationship); ok {
		// For now, use Neo4j internal IDs for source and target
		// TODO: In a more complete implementation, we would need to map these to custom IDs
		// This requires either storing the ID mapping or querying the nodes separately
		sourceID := fmt.Sprintf("node-%d", neo4jRel.StartId)
		targetID := fmt.Sprintf("node-%d", neo4jRel.EndId)
		relType := neo4jRel.Type

		// Create a unique edge ID
		edgeID := fmt.Sprintf("edge-%d", neo4jRel.Id)

		return &domain.GraphEdge{
			ID:     edgeID,
			Source: sourceID,
			Target: targetID,
			Type:   r.convertRelationshipType(relType),
		}
	}

	r.logger.Warn("Invalid relationship data format from Neo4j", "type", fmt.Sprintf("%T", relData))
	return nil
}

// determineNodeTypeFromLabels determines the domain node type from Neo4j labels
func (r *ConversationGraphRepository) determineNodeTypeFromLabels(labels []string) string {
	if len(labels) == 0 {
		return "unknown"
	}

	// Convert Neo4j labels to domain types
	label := labels[0] // Use the first label
	switch label {
	case "User":
		return "user"
	case "Conversation":
		return "conversation"
	case "Decision":
		return "decision"
	case "ExecutionPlan":
		return "execution_plan"
	case "ExecutionStep":
		return "execution_step"
	case "Agent":
		return "agent"
	case "Result":
		return "result"
	case "ConversationMessage":
		return "message"
	default:
		return "unknown"
	}
}

// convertRelationshipType converts Neo4j relationship types to domain edge types
func (r *ConversationGraphRepository) convertRelationshipType(neo4jType string) string {
	switch neo4jType {
	case "CREATED_BY":
		return "created_by"
	case "HAS_DECISION":
		return "has_decision"
	case "GENERATED":
		return "generated"
	case "HAS_STEP":
		return "has_step"
	case "EXECUTED_BY":
		return "executed_by"
	case "PRODUCED":
		return "produced"
	case "CONTAINS":
		return "contains"
	case "HAS_EXECUTION_PLAN":
		return "has_execution_plan"
	default:
		return neo4jType
	}
}

// calculateNodePosition calculates layout positions for graph visualization
func (r *ConversationGraphRepository) calculateNodePosition(nodeType, nodeID, conversationID string) *domain.NodePosition {
	// Basic layout algorithm - improve this later with proper graph layout
	baseX := 100
	baseY := 150
	spacing := 200

	switch nodeType {
	case "user":
		return &domain.NodePosition{X: float64(baseX), Y: float64(baseY)}
	case "conversation":
		return &domain.NodePosition{X: float64(baseX + spacing), Y: float64(baseY)}
	case "decision":
		return &domain.NodePosition{X: float64(baseX + spacing*2), Y: float64(baseY)}
	case "execution_plan":
		return &domain.NodePosition{X: float64(baseX + spacing*3), Y: float64(baseY)}
	case "execution_step":
		return &domain.NodePosition{X: float64(baseX + spacing*4), Y: float64(baseY)}
	case "agent":
		return &domain.NodePosition{X: float64(baseX + spacing*5), Y: float64(baseY)}
	case "result":
		return &domain.NodePosition{X: float64(baseX + spacing*6), Y: float64(baseY)}
	case "message":
		// Messages can be stacked vertically under the conversation
		return &domain.NodePosition{X: float64(baseX + spacing), Y: float64(baseY + 100)}
	default:
		return &domain.NodePosition{X: float64(baseX), Y: float64(baseY + 200)}
	}
}
