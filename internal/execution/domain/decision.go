package domain

import (
	"time"
)

// DecisionType represents the type of decision made by the AI (execution domain)
type DecisionType string

const (
	DecisionTypeExecute DecisionType = "EXECUTE"
)

// Decision represents an AI execution decision
// Only execution-related fields are present here
// Planning-related fields are not included
//
type Decision struct {
	Type              DecisionType           `json:"type"`
	Action            string                 `json:"action,omitempty"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	ExecutionPlan     string                 `json:"execution_plan,omitempty"`
	AgentCoordination string                 `json:"agent_coordination,omitempty"`
	Reasoning         string                 `json:"reasoning"`
	Timestamp         time.Time              `json:"timestamp"`
}

// NewExecuteDecision creates a decision to execute a plan
func NewExecuteDecision(executionPlan, agentCoordination, reasoning string) *Decision {
	return &Decision{
		Type:              DecisionTypeExecute,
		ExecutionPlan:     executionPlan,
		AgentCoordination: agentCoordination,
		Reasoning:         reasoning,
		Timestamp:         time.Now(),
	}
}

// IsExecutable returns true if this decision can be executed
func (d *Decision) IsExecutable() bool {
	return d.Type == DecisionTypeExecute
}

// HasAction returns true if the decision has a specific action to execute
func (d *Decision) HasAction() bool {
	return d.Action != ""
}

// GetType returns the decision type as string
func (d *Decision) GetType() string {
	return string(d.Type)
}

// GetReasoning returns the reasoning
func (d *Decision) GetReasoning() string {
	return d.Reasoning
}

// GetTimestamp returns the timestamp as string
func (d *Decision) GetTimestamp() string {
	return d.Timestamp.Format(time.RFC3339)
}
