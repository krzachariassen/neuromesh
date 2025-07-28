package domain

import (
	"time"

	"github.com/google/uuid"
)

// PlanningType represents the type of planning result
type PlanningType string

const (
	PlanningTypeClarify PlanningType = "CLARIFY"
	PlanningTypeExecute PlanningType = "EXECUTE"
	// PlanningTypeRespondDirectly removed - unified architecture uses only EXECUTE
)

// PlanningResult represents the result of AI planning for a user request
// This replaces the confusing "Decision" concept with proper planning terminology
type PlanningResult struct {
	ID        string       `json:"id"`
	RequestID string       `json:"request_id"`
	Type      PlanningType `json:"type"`

	// Agent Analysis
	AvailableAgents []string `json:"available_agents"`
	RequiredAgents  []string `json:"required_agents"`
	AgentGap        []string `json:"agent_gap"` // Required but not available

	// Planning Outcomes
	ExecutionPlanID       string `json:"execution_plan_id,omitempty"`
	ClarificationQuestion string `json:"clarification_question,omitempty"`
	DirectResponse        string `json:"direct_response,omitempty"`

	// Metadata
	Intent     string    `json:"intent"`
	Category   string    `json:"category"`
	Confidence int       `json:"confidence"`
	Reasoning  string    `json:"reasoning"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewExecutePlanningResult creates a planning result for execution
func NewExecutePlanningResult(requestID, intent, category string, confidence int, availableAgents, requiredAgents []string, executionPlanID, reasoning string) *PlanningResult {
	return &PlanningResult{
		ID:              uuid.New().String(),
		RequestID:       requestID,
		Type:            PlanningTypeExecute,
		AvailableAgents: availableAgents,
		RequiredAgents:  requiredAgents,
		AgentGap:        calculateAgentGap(availableAgents, requiredAgents),
		ExecutionPlanID: executionPlanID,
		Intent:          intent,
		Category:        category,
		Confidence:      confidence,
		Reasoning:       reasoning,
		Timestamp:       time.Now(),
	}
}

// NewClarificationPlanningResult creates a planning result for clarification
func NewClarificationPlanningResult(requestID, intent, category string, confidence int, clarificationQuestion, reasoning string) *PlanningResult {
	return &PlanningResult{
		ID:                    uuid.New().String(),
		RequestID:             requestID,
		Type:                  PlanningTypeClarify,
		AvailableAgents:       []string{},
		RequiredAgents:        []string{},
		AgentGap:              []string{},
		ClarificationQuestion: clarificationQuestion,
		Intent:                intent,
		Category:              category,
		Confidence:            confidence,
		Reasoning:             reasoning,
		Timestamp:             time.Now(),
	}
}

// Unified Architecture: No NewRespondDirectlyPlanningResult - everything goes through execution

// HasAgentGap returns true if there are required agents that are not available
func (p *PlanningResult) HasAgentGap() bool {
	return len(p.AgentGap) > 0
}

// calculateAgentGap determines which required agents are not available
func calculateAgentGap(availableAgents, requiredAgents []string) []string {
	available := make(map[string]bool)
	for _, agent := range availableAgents {
		available[agent] = true
	}

	var gap []string
	for _, required := range requiredAgents {
		if !available[required] && required != "none" && required != "" {
			gap = append(gap, required)
		}
	}

	// Return empty slice instead of nil for consistent behavior
	if gap == nil {
		gap = []string{}
	}

	return gap
}
