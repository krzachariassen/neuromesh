package domain

import (
	"time"

	"github.com/google/uuid"
)

// SynthesisResultStatus represents the status of synthesis result
type SynthesisResultStatus string

const (
	SynthesisResultStatusPending   SynthesisResultStatus = "PENDING"
	SynthesisResultStatusCompleted SynthesisResultStatus = "COMPLETED"
	SynthesisResultStatusFailed    SynthesisResultStatus = "FAILED"
)

// SynthesisResult represents the final synthesized result from multiple agent outputs
type SynthesisResult struct {
	ID           string                 `json:"id"`
	PlanID       string                 `json:"plan_id"`
	Content      string                 `json:"content"`
	Status       SynthesisResultStatus  `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewSynthesisResult creates a new synthesis result
func NewSynthesisResult(planID, content string) *SynthesisResult {
	return &SynthesisResult{
		ID:          uuid.New().String(),
		PlanID:      planID,
		Content:     content,
		Status:      SynthesisResultStatusCompleted,
		CreatedAt:   time.Now(),
		CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
		Metadata:    make(map[string]interface{}),
	}
}

// NewPendingSynthesisResult creates a synthesis result in pending state
func NewPendingSynthesisResult(planID string) *SynthesisResult {
	return &SynthesisResult{
		ID:        uuid.New().String(),
		PlanID:    planID,
		Status:    SynthesisResultStatusPending,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// Complete marks the synthesis result as completed
func (sr *SynthesisResult) Complete(content string) {
	sr.Content = content
	sr.Status = SynthesisResultStatusCompleted
	now := time.Now()
	sr.CompletedAt = &now
}

// Fail marks the synthesis result as failed
func (sr *SynthesisResult) Fail(errorMessage string) {
	sr.Status = SynthesisResultStatusFailed
	sr.ErrorMessage = errorMessage
	now := time.Now()
	sr.CompletedAt = &now
}
