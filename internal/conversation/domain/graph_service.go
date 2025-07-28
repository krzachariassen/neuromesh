package domain

import (
	"context"
)

// GraphNode represents a node in the conversation graph
type GraphNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data"`
	Position *NodePosition          `json:"position,omitempty"`
}

// GraphEdge represents an edge in the conversation graph
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// NodePosition represents the position of a node in the graph visualization
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GraphData represents the complete graph data for visualization
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// ConversationGraphService defines the interface for conversation graph operations
// This interface abstracts away any infrastructure details (Neo4j, etc.)
type ConversationGraphService interface {
	// GetConversationGraph retrieves the complete graph for a conversation
	GetConversationGraph(ctx context.Context, conversationID string) (*GraphData, error)

	// GetConversationSubgraph retrieves a filtered subgraph for a conversation
	GetConversationSubgraph(ctx context.Context, conversationID string, nodeTypes []string) (*GraphData, error)

	// GetGraphStats returns statistics about the conversation graph
	GetGraphStats(ctx context.Context, conversationID string) (map[string]interface{}, error)
}
