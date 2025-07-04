package domain

import (
	"time"
)

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
