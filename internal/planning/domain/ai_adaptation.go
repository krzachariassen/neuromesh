package domain

import (
	"time"
)

// AIPlanAdaptation represents how the AI dynamically adapts execution plans
type AIPlanAdaptation struct {
	PlanID          string                 `json:"plan_id"`
	AdaptationID    string                 `json:"adaptation_id"`
	AIReasoning     string                 `json:"ai_reasoning"`
	ConfidenceScore float64                `json:"confidence_score"`
	AdaptationType  string                 `json:"adaptation_type"`
	TriggeredBy     string                 `json:"triggered_by"`
	PreviousState   map[string]interface{} `json:"previous_state"`
	NewState        map[string]interface{} `json:"new_state"`
	Context         map[string]interface{} `json:"context"`
	Timestamp       time.Time              `json:"timestamp"`
}

// AIExecutionContext represents the AI's context when creating execution plans
type AIExecutionContext struct {
	ConversationHistory []string               `json:"conversation_history"`
	UserPatterns        []string               `json:"user_patterns"`
	PreviousDecisions   []string               `json:"previous_decisions"`
	EnvironmentContext  map[string]interface{} `json:"environment_context"`
	AgentAvailability   map[string]interface{} `json:"agent_availability"`
	PerformanceMetrics  map[string]interface{} `json:"performance_metrics"`
	ConfidenceFactors   map[string]float64     `json:"confidence_factors"`
}

// NewAIPlanAdaptation creates a new AI plan adaptation record
func NewAIPlanAdaptation(planID, adaptationID, reasoning, adaptationType, triggeredBy string, confidence float64) *AIPlanAdaptation {
	return &AIPlanAdaptation{
		PlanID:          planID,
		AdaptationID:    adaptationID,
		AIReasoning:     reasoning,
		ConfidenceScore: confidence,
		AdaptationType:  adaptationType,
		TriggeredBy:     triggeredBy,
		PreviousState:   make(map[string]interface{}),
		NewState:        make(map[string]interface{}),
		Context:         make(map[string]interface{}),
		Timestamp:       time.Now().UTC(),
	}
}
