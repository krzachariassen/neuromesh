package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ExecutionPlanStatus represents the status of an execution plan
type ExecutionPlanStatus string

const (
	ExecutionPlanStatusDraft     ExecutionPlanStatus = "DRAFT"
	ExecutionPlanStatusApproved  ExecutionPlanStatus = "APPROVED"
	ExecutionPlanStatusExecuting ExecutionPlanStatus = "EXECUTING"
	ExecutionPlanStatusCompleted ExecutionPlanStatus = "COMPLETED"
	ExecutionPlanStatusFailed    ExecutionPlanStatus = "FAILED"
)

// ExecutionPlanPriority represents the priority level of an execution plan
type ExecutionPlanPriority string

const (
	ExecutionPlanPriorityLow      ExecutionPlanPriority = "LOW"
	ExecutionPlanPriorityMedium   ExecutionPlanPriority = "MEDIUM"
	ExecutionPlanPriorityHigh     ExecutionPlanPriority = "HIGH"
	ExecutionPlanPriorityCritical ExecutionPlanPriority = "CRITICAL"
)

// ExecutionPlan represents a unified plan containing both planning intelligence and execution metadata
// This consolidates the former PlanningResult and ExecutionPlan into a single entity
type ExecutionPlan struct {
	// Existing execution metadata
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Status      ExecutionPlanStatus   `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	Priority    ExecutionPlanPriority `json:"priority"`
	Steps       []*ExecutionStep      `json:"steps,omitempty"`

	// Planning intelligence (from former PlanningResult)
	RequestID       string       `json:"request_id"`
	Type            PlanningType `json:"type"`
	Intent          string       `json:"intent"`
	Category        string       `json:"category"`
	Confidence      int          `json:"confidence"`
	Reasoning       string       `json:"reasoning"`
	AvailableAgents []string     `json:"available_agents"`
	RequiredAgents  []string     `json:"required_agents"`
	AgentGap        []string     `json:"agent_gap"`

	// Unified timestamps
	PlanningCompletedAt *time.Time `json:"planning_completed_at,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	EstimatedDuration   int        `json:"estimated_duration"` // Duration in minutes
	ActualDuration      int        `json:"actual_duration"`    // Duration in minutes
	CanModify           bool       `json:"can_modify"`
}

// NewExecutionPlan creates a new execution plan with validation
func NewExecutionPlan(name, description string, priority ExecutionPlanPriority) *ExecutionPlan {
	return &ExecutionPlan{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Status:      ExecutionPlanStatusDraft,
		CreatedAt:   time.Now(),
		CanModify:   true,
		Priority:    priority,
		Steps:       make([]*ExecutionStep, 0),
	}
}

// NewUnifiedExecutionPlan creates a unified execution plan with both planning intelligence and execution metadata
func NewUnifiedExecutionPlan(
	planID, name, description string,
	priority ExecutionPlanPriority,
	requestID, intent, category string,
	confidence int,
	reasoning string,
	availableAgents, requiredAgents, agentGap []string,
	planningType PlanningType,
) *ExecutionPlan {
	return &ExecutionPlan{
		// Execution metadata
		ID:          planID,
		Name:        name,
		Description: description,
		Status:      ExecutionPlanStatusDraft,
		CreatedAt:   time.Now(),
		CanModify:   true,
		Priority:    priority,
		Steps:       make([]*ExecutionStep, 0),

		// Planning intelligence
		RequestID:       requestID,
		Type:            planningType,
		Intent:          intent,
		Category:        category,
		Confidence:      confidence,
		Reasoning:       reasoning,
		AvailableAgents: availableAgents,
		RequiredAgents:  requiredAgents,
		AgentGap:        agentGap,
	}
}

// Validate ensures the execution plan is valid
func (p *ExecutionPlan) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("execution plan ID cannot be empty")
	}
	if p.Name == "" {
		return fmt.Errorf("execution plan name cannot be empty")
	}
	if !p.Status.IsValid() {
		return fmt.Errorf("invalid execution plan status: %s", p.Status)
	}
	if !p.Priority.IsValid() {
		return fmt.Errorf("invalid execution plan priority: %s", p.Priority)
	}
	return nil
}

// AddStep adds a new step to the execution plan
func (p *ExecutionPlan) AddStep(step *ExecutionStep) error {
	if step == nil {
		return fmt.Errorf("step cannot be nil")
	}
	if err := step.Validate(); err != nil {
		return fmt.Errorf("invalid step: %w", err)
	}

	// Set step number based on current steps
	step.StepNumber = len(p.Steps) + 1
	step.PlanID = p.ID

	p.Steps = append(p.Steps, step)
	return nil
}

// GetStepByNumber retrieves a step by its step number
func (p *ExecutionPlan) GetStepByNumber(stepNumber int) *ExecutionStep {
	for _, step := range p.Steps {
		if step.StepNumber == stepNumber {
			return step
		}
	}
	return nil
}

// GetStepsByStatus retrieves all steps with a specific status
func (p *ExecutionPlan) GetStepsByStatus(status ExecutionStepStatus) []*ExecutionStep {
	var steps []*ExecutionStep
	for _, step := range p.Steps {
		if step.Status == status {
			steps = append(steps, step)
		}
	}
	return steps
}

// GetNextStep returns the next step to be executed
func (p *ExecutionPlan) GetNextStep() *ExecutionStep {
	for _, step := range p.Steps {
		if step.Status == ExecutionStepStatusAssigned {
			return step
		}
	}
	return nil
}

// Approve marks the plan as approved and sets the approval timestamp
func (p *ExecutionPlan) Approve() error {
	if p.Status != ExecutionPlanStatusDraft {
		return fmt.Errorf("can only approve plans in DRAFT status, current status: %s", p.Status)
	}

	p.Status = ExecutionPlanStatusApproved
	now := time.Now()
	p.ApprovedAt = &now
	p.CanModify = false // Lock plan after approval
	return nil
}

// Start marks the plan as executing and sets the start timestamp
func (p *ExecutionPlan) Start() error {
	if p.Status != ExecutionPlanStatusApproved {
		return fmt.Errorf("plan must be approved before starting")
	}
	p.Status = ExecutionPlanStatusExecuting
	now := time.Now()
	p.StartedAt = &now
	return nil
}

// Complete marks the plan as completed and calculates actual duration
func (p *ExecutionPlan) Complete() error {
	if p.Status != ExecutionPlanStatusExecuting {
		return fmt.Errorf("plan must be executing to complete")
	}
	p.Status = ExecutionPlanStatusCompleted
	now := time.Now()
	p.CompletedAt = &now

	// Calculate actual duration
	if p.StartedAt != nil {
		p.ActualDuration = int(now.Sub(*p.StartedAt).Minutes())
	}
	return nil
}

// Fail marks the plan as failed
func (p *ExecutionPlan) Fail() {
	p.Status = ExecutionPlanStatusFailed
	now := time.Now()
	p.CompletedAt = &now

	// Calculate actual duration
	if p.StartedAt != nil {
		p.ActualDuration = int(now.Sub(*p.StartedAt).Minutes())
	}
}

// IsComplete returns true if the plan is completed or failed
func (p *ExecutionPlan) IsComplete() bool {
	return p.Status == ExecutionPlanStatusCompleted || p.Status == ExecutionPlanStatusFailed
}

// IsExecutable returns true if the plan can be executed
func (p *ExecutionPlan) IsExecutable() bool {
	return p.Status == ExecutionPlanStatusApproved && len(p.Steps) > 0
}

// CanBeModified returns true if the plan can be modified
func (p *ExecutionPlan) CanBeModified() bool {
	return p.CanModify && (p.Status == ExecutionPlanStatusDraft || p.Status == ExecutionPlanStatusExecuting)
}

// ToMap converts the execution plan to a map for persistence
func (p *ExecutionPlan) ToMap() map[string]interface{} {
	data := map[string]interface{}{
		"id":                 p.ID,
		"name":               p.Name,
		"description":        p.Description,
		"status":             string(p.Status),
		"created_at":         p.CreatedAt.UTC(),
		"estimated_duration": p.EstimatedDuration,
		"actual_duration":    p.ActualDuration,
		"can_modify":         p.CanModify,
		"priority":           string(p.Priority),
	}

	if p.ApprovedAt != nil {
		data["approved_at"] = p.ApprovedAt.UTC()
	}
	if p.StartedAt != nil {
		data["started_at"] = p.StartedAt.UTC()
	}
	if p.CompletedAt != nil {
		data["completed_at"] = p.CompletedAt.UTC()
	}

	return data
}

// IsValid validates the ExecutionPlanStatus
func (s ExecutionPlanStatus) IsValid() bool {
	switch s {
	case ExecutionPlanStatusDraft, ExecutionPlanStatusApproved, ExecutionPlanStatusExecuting, ExecutionPlanStatusCompleted, ExecutionPlanStatusFailed:
		return true
	default:
		return false
	}
}

// IsValid validates the ExecutionPlanPriority
func (p ExecutionPlanPriority) IsValid() bool {
	switch p {
	case ExecutionPlanPriorityLow, ExecutionPlanPriorityMedium, ExecutionPlanPriorityHigh, ExecutionPlanPriorityCritical:
		return true
	default:
		return false
	}
}

// CompletePlanning marks the planning phase as completed
func (p *ExecutionPlan) CompletePlanning() error {
	if p.PlanningCompletedAt != nil {
		return fmt.Errorf("planning already completed")
	}

	now := time.Now()
	p.PlanningCompletedAt = &now
	return nil
}

// GetCompletePlanData returns complete plan data from unified entity
func (p *ExecutionPlan) GetCompletePlanData() map[string]interface{} {
	return map[string]interface{}{
		"id":                    p.ID,
		"name":                  p.Name,
		"description":           p.Description,
		"status":                p.Status,
		"priority":              p.Priority,
		"request_id":            p.RequestID,
		"type":                  p.Type,
		"intent":                p.Intent,
		"category":              p.Category,
		"confidence":            p.Confidence,
		"reasoning":             p.Reasoning,
		"available_agents":      p.AvailableAgents,
		"required_agents":       p.RequiredAgents,
		"agent_gap":             p.AgentGap,
		"created_at":            p.CreatedAt,
		"planning_completed_at": p.PlanningCompletedAt,
		"approved_at":           p.ApprovedAt,
		"started_at":            p.StartedAt,
		"completed_at":          p.CompletedAt,
		"steps":                 p.Steps,
	}
}

// GenerateID generates a new unique ID for execution plans
func GenerateID() string {
	return uuid.New().String()
}
