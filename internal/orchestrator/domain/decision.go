package domain

import (
	"time"

	"github.com/google/uuid"
)

// DecisionType represents the type of decision made by the AI
type DecisionType string

const (
	DecisionTypeClarify DecisionType = "CLARIFY"
	DecisionTypeExecute DecisionType = "EXECUTE"
)

// Decision represents an AI decision about how to handle a user request
type Decision struct {
	ID                    string                 `json:"id"`
	RequestID             string                 `json:"request_id"`
	AnalysisID            string                 `json:"analysis_id"`
	Type                  DecisionType           `json:"type"`
	Action                string                 `json:"action,omitempty"`
	Parameters            map[string]interface{} `json:"parameters,omitempty"`
	ClarificationQuestion string                 `json:"clarification_question,omitempty"`
	ExecutionPlan         string                 `json:"execution_plan,omitempty"`
	AgentCoordination     string                 `json:"agent_coordination,omitempty"`
	Reasoning             string                 `json:"reasoning"`
	Timestamp             time.Time              `json:"timestamp"`
}

// NewClarifyDecision creates a decision to ask for clarification
func NewClarifyDecision(requestID, analysisID, clarificationQuestion, reasoning string) *Decision {
	return &Decision{
		ID:                    uuid.New().String(),
		RequestID:             requestID,
		AnalysisID:            analysisID,
		Type:                  DecisionTypeClarify,
		ClarificationQuestion: clarificationQuestion,
		Reasoning:             reasoning,
		Timestamp:             time.Now(),
	}
}

// NewExecuteDecision creates a decision to execute a plan
func NewExecuteDecision(requestID, analysisID, executionPlan, agentCoordination, reasoning string) *Decision {
	return &Decision{
		ID:                uuid.New().String(),
		RequestID:         requestID,
		AnalysisID:        analysisID,
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

// NeedsClarification returns true if clarification is needed
func (d *Decision) NeedsClarification() bool {
	return d.Type == DecisionTypeClarify
}
