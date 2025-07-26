package web

// UI Data Transfer Objects (DTOs) for the React frontend
// These structs define the JSON API contracts between Go backend and TypeScript frontend

// GraphDataResponse represents graph data for React Flow visualization
type GraphDataResponse struct {
	ConversationID string      `json:"conversation_id"`
	Nodes          []GraphNode `json:"nodes"`
	Edges          []GraphEdge `json:"edges"`
}

// GraphNode represents a node in the graph visualization
type GraphNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // 'user' | 'conversation' | 'plan' | 'step' | 'agent' | 'result'
	Data     map[string]interface{} `json:"data"`
	Position *NodePosition          `json:"position,omitempty"`
}

// GraphEdge represents an edge in the graph visualization
type GraphEdge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Type   string                 `json:"type"` // 'created' | 'executed' | 'synthesized'
	Data   map[string]interface{} `json:"data,omitempty"`
}

// NodePosition represents the position of a node in the graph
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ExecutionPlanResponse represents execution plan data for UI
type ExecutionPlanResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Status      string              `json:"status"`
	CreatedAt   string              `json:"created_at"`
	Steps       []ExecutionStepData `json:"steps"`
}

// ExecutionStepData represents a single execution step for UI
type ExecutionStepData struct {
	StepNumber  int     `json:"step_number"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	AgentName   string  `json:"agent_name"`
	Status      string  `json:"status"`
	CompletedAt *string `json:"completed_at,omitempty"`
}

// ConversationHistoryResponse represents conversation history for UI
type ConversationHistoryResponse struct {
	SessionID     string             `json:"session_id"`
	Conversations []ConversationData `json:"conversations"`
	Messages      []MessageData      `json:"messages"`
}

// ConversationData represents a conversation for UI
type ConversationData struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// MessageData represents a message for UI
type MessageData struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	Role           string                 `json:"role"` // 'user' | 'assistant'
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      string                 `json:"created_at"`
}

// AgentStatusResponse represents agent status for UI
type AgentStatusResponse struct {
	Agents []AgentData `json:"agents"`
}

// AgentData represents agent information for UI
type AgentData struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
