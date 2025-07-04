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

// EmergentAgentPattern represents AI-discovered patterns in agent coordination
type EmergentAgentPattern struct {
	PatternID        string                 `json:"pattern_id"`
	UserContext      string                 `json:"user_context"`
	AgentSelection   []string               `json:"agent_selection"`
	CoordinationFlow map[string]interface{} `json:"coordination_flow"`
	SuccessRate      float64                `json:"success_rate"`
	AIConfidence     float64                `json:"ai_confidence"`
	LearnedFrom      []string               `json:"learned_from"`  // Conversation IDs
	ReinforcedBy     []string               `json:"reinforced_by"` // Additional conversations
	Context          map[string]interface{} `json:"context"`
	Timestamp        time.Time              `json:"timestamp"`
}

// AIReasoningPattern represents patterns in AI decision-making
type AIReasoningPattern struct {
	PatternID       string                 `json:"pattern_id"`
	UserIntent      string                 `json:"user_intent"`
	DecisionPath    []string               `json:"decision_path"`
	ReasoningChain  []string               `json:"reasoning_chain"`
	ConfidenceFlow  []float64              `json:"confidence_flow"`
	Context         map[string]interface{} `json:"context"`
	Frequency       int                    `json:"frequency"`
	SuccessRate     float64                `json:"success_rate"`
	LastSeen        time.Time              `json:"last_seen"`
	ConversationIDs []string               `json:"conversation_ids"`
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

// NewEmergentAgentPattern creates a new emergent agent pattern record
func NewEmergentAgentPattern(patternID, userContext string, agents []string, coordinationFlow map[string]interface{}, successRate, confidence float64) *EmergentAgentPattern {
	return &EmergentAgentPattern{
		PatternID:        patternID,
		UserContext:      userContext,
		AgentSelection:   agents,
		CoordinationFlow: coordinationFlow,
		SuccessRate:      successRate,
		AIConfidence:     confidence,
		LearnedFrom:      make([]string, 0),
		ReinforcedBy:     make([]string, 0),
		Context:          make(map[string]interface{}),
		Timestamp:        time.Now().UTC(),
	}
}

// NewAIReasoningPattern creates a new AI reasoning pattern record
func NewAIReasoningPattern(patternID, userIntent string, decisionPath, reasoningChain []string, confidenceFlow []float64) *AIReasoningPattern {
	return &AIReasoningPattern{
		PatternID:       patternID,
		UserIntent:      userIntent,
		DecisionPath:    decisionPath,
		ReasoningChain:  reasoningChain,
		ConfidenceFlow:  confidenceFlow,
		Context:         make(map[string]interface{}),
		Frequency:       1,
		SuccessRate:     0.0,
		LastSeen:        time.Now().UTC(),
		ConversationIDs: make([]string, 0),
	}
}
