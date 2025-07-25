package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AgentResultStatus represents the status of an agent execution result
type AgentResultStatus string

const (
	// AgentResultStatusSuccess indicates the agent completed successfully
	AgentResultStatusSuccess AgentResultStatus = "success"

	// AgentResultStatusFailed indicates the agent execution failed
	AgentResultStatusFailed AgentResultStatus = "failed"

	// AgentResultStatusPartial indicates the agent provided partial results
	AgentResultStatusPartial AgentResultStatus = "partial"
)

// AgentResult represents the result of an agent execution for a specific execution step
// This is a core domain entity that stores the output from agent executions in our graph-native architecture
type AgentResult struct {
	// ID is the unique identifier for this agent result
	ID string `json:"id"`

	// ExecutionStepID is the ID of the execution step this result belongs to
	ExecutionStepID string `json:"execution_step_id"`

	// AgentID is the ID of the agent that produced this result
	AgentID string `json:"agent_id"`

	// Content is the actual result content from the agent
	Content string `json:"content"`

	// Status indicates whether the agent execution was successful, failed, or partial
	Status AgentResultStatus `json:"status"`

	// Metadata contains additional information about the agent execution
	// This can include execution time, confidence scores, model versions, etc.
	Metadata map[string]interface{} `json:"metadata"`

	// Timestamp indicates when this result was created
	Timestamp time.Time `json:"timestamp"`
}

// NewAgentResult creates a new AgentResult with success status
// This is the primary constructor for successful agent results
func NewAgentResult(executionStepID, agentID, content string, metadata map[string]interface{}) *AgentResult {
	return NewAgentResultWithStatus(executionStepID, agentID, content, metadata, AgentResultStatusSuccess)
}

// NewAgentResultWithStatus creates a new AgentResult with specified status
// This constructor allows for creating results with any status (success, failed, partial)
func NewAgentResultWithStatus(executionStepID, agentID, content string, metadata map[string]interface{}, status AgentResultStatus) *AgentResult {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &AgentResult{
		ID:              generateAgentResultID(),
		ExecutionStepID: executionStepID,
		AgentID:         agentID,
		Content:         content,
		Status:          status,
		Metadata:        metadata,
		Timestamp:       time.Now().UTC(), // Always use UTC for consistency
	}
}

// generateAgentResultID creates a unique ID for an agent result
func generateAgentResultID() string {
	return "result_" + uuid.New().String()
}

// Validate checks if the AgentResult has all required fields and valid values
// This ensures data integrity before persisting to the graph
func (ar *AgentResult) Validate() error {
	if strings.TrimSpace(ar.ID) == "" {
		return fmt.Errorf("ID is required")
	}

	if strings.TrimSpace(ar.ExecutionStepID) == "" {
		return fmt.Errorf("ExecutionStepID is required")
	}

	if strings.TrimSpace(ar.AgentID) == "" {
		return fmt.Errorf("AgentID is required")
	}

	// Validate status against known values
	if !ar.isValidStatus() {
		return fmt.Errorf("invalid status: %s", ar.Status)
	}

	return nil
}

// isValidStatus checks if the status is one of the predefined valid statuses
func (ar *AgentResult) isValidStatus() bool {
	validStatuses := []AgentResultStatus{
		AgentResultStatusSuccess,
		AgentResultStatusFailed,
		AgentResultStatusPartial,
	}

	for _, validStatus := range validStatuses {
		if ar.Status == validStatus {
			return true
		}
	}

	return false
}

// IsSuccessful returns true if the agent execution was successful
func (ar *AgentResult) IsSuccessful() bool {
	return ar.Status == AgentResultStatusSuccess
}

// IsFailed returns true if the agent execution failed
func (ar *AgentResult) IsFailed() bool {
	return ar.Status == AgentResultStatusFailed
}

// IsPartial returns true if the agent provided partial results
func (ar *AgentResult) IsPartial() bool {
	return ar.Status == AgentResultStatusPartial
}

// MarkAsFailed updates the agent result status to failed and adds error information
// This is used when an agent execution encounters an error
func (ar *AgentResult) MarkAsFailed(errorMessage string) {
	ar.Status = AgentResultStatusFailed
	ar.addMetadata("error", errorMessage)
	ar.addMetadata("failed_at", time.Now().UTC())
}

// MarkAsPartial updates the agent result status to partial and adds reason
func (ar *AgentResult) MarkAsPartial(reason string) {
	ar.Status = AgentResultStatusPartial
	ar.addMetadata("partial_reason", reason)
}

// AddExecutionMetadata adds execution-related metadata to the result
func (ar *AgentResult) AddExecutionMetadata(key string, value interface{}) {
	ar.addMetadata(key, value)
}

// addMetadata safely adds metadata to the result
func (ar *AgentResult) addMetadata(key string, value interface{}) {
	if ar.Metadata == nil {
		ar.Metadata = make(map[string]interface{})
	}
	ar.Metadata[key] = value
}

// GetMetadata safely retrieves metadata value
func (ar *AgentResult) GetMetadata(key string) (interface{}, bool) {
	if ar.Metadata == nil {
		return nil, false
	}
	value, exists := ar.Metadata[key]
	return value, exists
}
