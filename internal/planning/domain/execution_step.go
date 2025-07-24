package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ExecutionStepStatus represents the status of an execution step
type ExecutionStepStatus string

const (
	ExecutionStepStatusPending   ExecutionStepStatus = "PENDING"
	ExecutionStepStatusAssigned  ExecutionStepStatus = "ASSIGNED"
	ExecutionStepStatusExecuting ExecutionStepStatus = "EXECUTING"
	ExecutionStepStatusCompleted ExecutionStepStatus = "COMPLETED"
	ExecutionStepStatusFailed    ExecutionStepStatus = "FAILED"
	ExecutionStepStatusSkipped   ExecutionStepStatus = "SKIPPED"
)

// ExecutionStep represents an individual step within an execution plan
type ExecutionStep struct {
	ID                string              `json:"id"`
	PlanID            string              `json:"plan_id"`     // For graph relationship
	StepNumber        int                 `json:"step_number"` // Execution order
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	AssignedAgent     string              `json:"assigned_agent"` // Agent ID for graph relationship
	Status            ExecutionStepStatus `json:"status"`
	EstimatedDuration int                 `json:"estimated_duration"` // Duration in minutes
	ActualDuration    int                 `json:"actual_duration"`    // Duration in minutes
	Inputs            string              `json:"inputs"`             // JSON of input parameters
	Outputs           string              `json:"outputs"`            // JSON of output results
	ErrorMessage      string              `json:"error_message"`      // Error details if failed
	CanModify         bool                `json:"can_modify"`         // Can this step be modified during execution?
	IsCritical        bool                `json:"is_critical"`        // Is this step critical to overall success?
	RetryCount        int                 `json:"retry_count"`        // Number of times this step has been retried
	MaxRetries        int                 `json:"max_retries"`        // Maximum allowed retries
	StartedAt         *time.Time          `json:"started_at"`         // When step execution started
	CompletedAt       *time.Time          `json:"completed_at"`       // When step execution completed
}

// NewExecutionStep creates a new execution step with validation
func NewExecutionStep(name, description, assignedAgent string) *ExecutionStep {
	return &ExecutionStep{
		ID:            uuid.New().String(),
		Name:          name,
		Description:   description,
		AssignedAgent: assignedAgent,
		Status:        ExecutionStepStatusPending,
		CanModify:     true,
		IsCritical:    false,
		RetryCount:    0,
		MaxRetries:    3, // Default max retries
	}
}

// Validate ensures the execution step is valid
func (s *ExecutionStep) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("execution step ID cannot be empty")
	}
	if s.Name == "" {
		return fmt.Errorf("execution step name cannot be empty")
	}
	if s.AssignedAgent == "" {
		return fmt.Errorf("assigned agent cannot be empty")
	}
	if !s.Status.IsValid() {
		return fmt.Errorf("invalid execution step status: %s", s.Status)
	}
	return nil
}

// Assign marks the step as assigned
func (s *ExecutionStep) Assign() {
	s.Status = ExecutionStepStatusAssigned
}

// Start marks the step as executing and sets the start timestamp
func (s *ExecutionStep) Start() error {
	if s.Status != ExecutionStepStatusAssigned {
		return fmt.Errorf("step must be assigned before starting")
	}
	s.Status = ExecutionStepStatusExecuting
	now := time.Now()
	s.StartedAt = &now
	return nil
}

// Complete marks the step as completed and calculates actual duration
func (s *ExecutionStep) Complete(outputs string) error {
	if s.Status != ExecutionStepStatusExecuting {
		return fmt.Errorf("step must be executing to complete")
	}
	s.Status = ExecutionStepStatusCompleted
	s.Outputs = outputs
	now := time.Now()
	s.CompletedAt = &now

	// Calculate actual duration
	if s.StartedAt != nil {
		s.ActualDuration = int(now.Sub(*s.StartedAt).Minutes())
	}
	return nil
}

// Fail marks the step as failed
func (s *ExecutionStep) Fail(errorMessage string) {
	s.Status = ExecutionStepStatusFailed
	s.ErrorMessage = errorMessage
	now := time.Now()
	s.CompletedAt = &now

	// Calculate actual duration
	if s.StartedAt != nil {
		s.ActualDuration = int(now.Sub(*s.StartedAt).Minutes())
	}
}

// Retry resets the step for retry if allowed
func (s *ExecutionStep) Retry() error {
	if s.RetryCount >= s.MaxRetries {
		return fmt.Errorf("maximum retries exceeded (%d)", s.MaxRetries)
	}

	s.RetryCount++
	s.Status = ExecutionStepStatusPending
	s.ErrorMessage = ""
	s.StartedAt = nil
	s.CompletedAt = nil

	return nil
}

// CanRetry returns true if the step can be retried
func (s *ExecutionStep) CanRetry() bool {
	return s.Status == ExecutionStepStatusFailed && s.RetryCount < s.MaxRetries
}

// IsComplete returns true if the step is completed, failed, or skipped
func (s *ExecutionStep) IsComplete() bool {
	return s.Status == ExecutionStepStatusCompleted ||
		s.Status == ExecutionStepStatusFailed ||
		s.Status == ExecutionStepStatusSkipped
}

// CanBeModified returns true if the step can be modified
func (s *ExecutionStep) CanBeModified() bool {
	return s.CanModify && s.Status == ExecutionStepStatusPending
}

// IsValid validates the ExecutionStepStatus
func (s ExecutionStepStatus) IsValid() bool {
	switch s {
	case ExecutionStepStatusPending, ExecutionStepStatusAssigned, ExecutionStepStatusExecuting,
		ExecutionStepStatusCompleted, ExecutionStepStatusFailed, ExecutionStepStatusSkipped:
		return true
	default:
		return false
	}
}

// ToMap converts the execution step to a map for persistence
func (s *ExecutionStep) ToMap() map[string]interface{} {
	data := map[string]interface{}{
		"id":                 s.ID,
		"plan_id":            s.PlanID,
		"step_number":        s.StepNumber,
		"name":               s.Name,
		"description":        s.Description,
		"assigned_agent":     s.AssignedAgent,
		"status":             string(s.Status),
		"estimated_duration": s.EstimatedDuration,
		"actual_duration":    s.ActualDuration,
		"inputs":             s.Inputs,
		"outputs":            s.Outputs,
		"error_message":      s.ErrorMessage,
		"can_modify":         s.CanModify,
		"is_critical":        s.IsCritical,
		"retry_count":        s.RetryCount,
		"max_retries":        s.MaxRetries,
	}

	if s.StartedAt != nil {
		data["started_at"] = s.StartedAt.UTC()
	}
	if s.CompletedAt != nil {
		data["completed_at"] = s.CompletedAt.UTC()
	}

	return data
}
