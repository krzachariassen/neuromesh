package domain

import (
	"context"
	"fmt"
	"time"
)

// ResultSynthesizer defines the interface for synthesizing agent results into cohesive outputs
type ResultSynthesizer interface {
	// SynthesizeResults takes all agent results for an execution plan and creates a synthesized output
	// The synthesis process uses AI to intelligently combine agent results into a coherent response
	SynthesizeResults(ctx context.Context, planID string) (string, error)
	
	// GetSynthesisContext retrieves and structures all data needed for synthesis
	// This includes agent results, execution plan details, and contextual metadata
	GetSynthesisContext(ctx context.Context, planID string) (*SynthesisContext, error)
}

// SynthesisContext contains all the context needed for synthesizing agent results
type SynthesisContext struct {
	// ExecutionPlanID is the ID of the execution plan being synthesized
	ExecutionPlanID string `json:"execution_plan_id"`
	
	// AgentResults contains all the agent results for this execution plan
	// These results are ordered by step execution sequence
	AgentResults []*AgentResult `json:"agent_results"`
	
	// CreatedAt indicates when this synthesis context was created
	CreatedAt time.Time `json:"created_at"`
	
	// Metadata contains additional context information for synthesis
	// Common fields: user_request, execution_type, priority, domain_context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewSynthesisContext creates a new synthesis context with validation
func NewSynthesisContext(planID string, results []*AgentResult) *SynthesisContext {
	return &SynthesisContext{
		ExecutionPlanID: planID,
		AgentResults:    results,
		CreatedAt:       time.Now(),
		Metadata:        make(map[string]interface{}),
	}
}

// AddMetadata adds metadata to the synthesis context
func (sc *SynthesisContext) AddMetadata(key string, value interface{}) {
	if sc.Metadata == nil {
		sc.Metadata = make(map[string]interface{})
	}
	sc.Metadata[key] = value
}

// GetSuccessfulResults returns only the agent results that completed successfully
func (sc *SynthesisContext) GetSuccessfulResults() []*AgentResult {
	var successful []*AgentResult
	for _, result := range sc.AgentResults {
		if result.Status == AgentResultStatusSuccess {
			successful = append(successful, result)
		}
	}
	return successful
}

// Validate ensures the synthesis context has all required fields
func (sc *SynthesisContext) Validate() error {
	if sc.ExecutionPlanID == "" {
		return fmt.Errorf("ExecutionPlanID cannot be empty")
	}
	
	if sc.AgentResults == nil {
		return fmt.Errorf("AgentResults cannot be nil")
	}
	
	if sc.CreatedAt.IsZero() {
		return fmt.Errorf("CreatedAt cannot be zero")
	}
	
	return nil
}
