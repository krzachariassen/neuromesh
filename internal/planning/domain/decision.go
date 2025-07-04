package domain

import (
	"time"
)

// DecisionType represents the type of decision made by the AI (planning domain)
type DecisionType string

const (
	DecisionTypeClarify DecisionType = "CLARIFY"
)

// Decision represents an AI planning decision
// Only planning-related fields are present here
// Execution-related fields are not included
//
type Decision struct {
	Type                  DecisionType `json:"type"`
	ClarificationQuestion string       `json:"clarification_question,omitempty"`
	Reasoning             string       `json:"reasoning"`
	Timestamp             time.Time    `json:"timestamp"`
}

// NewClarifyDecision creates a decision to ask for clarification
func NewClarifyDecision(clarificationQuestion, reasoning string) *Decision {
	return &Decision{
		Type:                  DecisionTypeClarify,
		ClarificationQuestion: clarificationQuestion,
		Reasoning:             reasoning,
		Timestamp:             time.Now(),
	}
}

// NeedsClarification returns true if clarification is needed
func (d *Decision) NeedsClarification() bool {
	return d.Type == DecisionTypeClarify
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
