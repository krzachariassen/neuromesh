package domain

// Node represents a graph node for API responses
type Node struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Label    string                 `json:"label"`
	Position *NodePosition          `json:"position,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// Edge represents a graph edge for API responses
type Edge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Type   string                 `json:"type"`
	Label  string                 `json:"label,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// NodePosition represents the position of a node in the visualization
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GraphData represents the complete graph data for visualization
type GraphData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// ConversationResponse represents a conversation in API responses
type ConversationResponse struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ExecutionPlanResponse represents an execution plan in API responses
type ExecutionPlanResponse struct {
	ID             string                  `json:"id"`
	ConversationID string                  `json:"conversation_id"`
	Status         string                  `json:"status"`
	Steps          []ExecutionStepResponse `json:"steps"`
	CreatedAt      string                  `json:"created_at"`
	UpdatedAt      string                  `json:"updated_at"`
}

// ExecutionStepResponse represents an execution step in API responses
type ExecutionStepResponse struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Status   string                 `json:"status"`
	Input    map[string]interface{} `json:"input"`
	Output   map[string]interface{} `json:"output,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ChatRequest represents a request to start a new conversation
type ChatRequest struct {
	Message   string `json:"message"`
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
}

// ChatMessage represents a message in an existing conversation
type ChatMessage struct {
	Message string `json:"message"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	ConversationID string `json:"conversation_id"`
	SessionID      string `json:"session_id"`
	Response       string `json:"response"`
	ProjectID      string `json:"project_id"`
	UserID         string `json:"user_id"`
}
