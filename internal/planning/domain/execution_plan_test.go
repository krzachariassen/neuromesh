package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionPlan(t *testing.T) {
	name := "Deploy Application"
	description := "Deploy application to Kubernetes cluster"
	priority := ExecutionPlanPriorityHigh

	plan := NewExecutionPlan(name, description, priority)

	assert.NotEmpty(t, plan.ID)
	assert.Equal(t, name, plan.Name)
	assert.Equal(t, description, plan.Description)
	assert.Equal(t, ExecutionPlanStatusDraft, plan.Status)
	assert.Equal(t, priority, plan.Priority)
	assert.True(t, plan.CanModify)
	assert.WithinDuration(t, time.Now(), plan.CreatedAt, time.Second)
	assert.Empty(t, plan.Steps)
}

func TestExecutionPlan_Validate(t *testing.T) {
	tests := []struct {
		name    string
		plan    *ExecutionPlan
		wantErr bool
	}{
		{
			name: "valid plan",
			plan: &ExecutionPlan{
				ID:          "plan-123",
				Name:        "Test Plan",
				Description: "Test Description",
				Status:      ExecutionPlanStatusDraft,
				Priority:    ExecutionPlanPriorityMedium,
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			plan: &ExecutionPlan{
				Name:        "Test Plan",
				Description: "Test Description",
				Status:      ExecutionPlanStatusDraft,
				Priority:    ExecutionPlanPriorityMedium,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			plan: &ExecutionPlan{
				ID:          "plan-123",
				Description: "Test Description",
				Status:      ExecutionPlanStatusDraft,
				Priority:    ExecutionPlanPriorityMedium,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			plan: &ExecutionPlan{
				ID:          "plan-123",
				Name:        "Test Plan",
				Description: "Test Description",
				Status:      ExecutionPlanStatus("INVALID"),
				Priority:    ExecutionPlanPriorityMedium,
			},
			wantErr: true,
		},
		{
			name: "invalid priority",
			plan: &ExecutionPlan{
				ID:          "plan-123",
				Name:        "Test Plan",
				Description: "Test Description",
				Status:      ExecutionPlanStatusDraft,
				Priority:    ExecutionPlanPriority("INVALID"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionPlan_AddStep(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)
	step := NewExecutionStep("Step 1", "First step", "agent-1")

	err := plan.AddStep(step)

	require.NoError(t, err)
	assert.Len(t, plan.Steps, 1)
	assert.Equal(t, step, plan.Steps[0])
	assert.Equal(t, 1, step.StepNumber)
	assert.Equal(t, plan.ID, step.PlanID)
}

func TestExecutionPlan_AddStep_InvalidStep(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)

	err := plan.AddStep(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "step cannot be nil")
}

func TestExecutionPlan_GetStepByNumber(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)
	step1 := NewExecutionStep("Step 1", "First step", "agent-1")
	step2 := NewExecutionStep("Step 2", "Second step", "agent-2")

	plan.AddStep(step1)
	plan.AddStep(step2)

	foundStep := plan.GetStepByNumber(2)
	assert.Equal(t, step2, foundStep)

	notFound := plan.GetStepByNumber(99)
	assert.Nil(t, notFound)
}

func TestExecutionPlan_GetStepsByStatus(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)
	step1 := NewExecutionStep("Step 1", "First step", "agent-1")
	step2 := NewExecutionStep("Step 2", "Second step", "agent-2")

	step1.Status = ExecutionStepStatusPending
	step2.Status = ExecutionStepStatusCompleted

	plan.AddStep(step1)
	plan.AddStep(step2)

	pendingSteps := plan.GetStepsByStatus(ExecutionStepStatusPending)
	assert.Len(t, pendingSteps, 1)
	assert.Equal(t, step1, pendingSteps[0])
}

func TestExecutionPlan_GetNextStep(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)
	step1 := NewExecutionStep("Step 1", "First step", "agent-1")
	step2 := NewExecutionStep("Step 2", "Second step", "agent-2")

	step1.Status = ExecutionStepStatusCompleted
	step2.Status = ExecutionStepStatusPending

	plan.AddStep(step1)
	plan.AddStep(step2)

	nextStep := plan.GetNextStep()
	assert.Equal(t, step2, nextStep)
}

func TestExecutionPlan_StatusTransitions(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)

	// Test Approve
	plan.Approve()
	assert.Equal(t, ExecutionPlanStatusApproved, plan.Status)
	assert.NotNil(t, plan.ApprovedAt)

	// Test Start
	err := plan.Start()
	assert.NoError(t, err)
	assert.Equal(t, ExecutionPlanStatusExecuting, plan.Status)
	assert.NotNil(t, plan.StartedAt)

	// Test Complete
	err = plan.Complete()
	assert.NoError(t, err)
	assert.Equal(t, ExecutionPlanStatusCompleted, plan.Status)
	assert.NotNil(t, plan.CompletedAt)
	assert.GreaterOrEqual(t, plan.ActualDuration, 0) // Duration can be 0 for very fast execution
}

func TestExecutionPlan_StatusTransitions_Invalid(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)

	// Cannot start without approval
	err := plan.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be approved")

	// Cannot complete without executing
	err = plan.Complete()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be executing")
}

func TestExecutionPlan_IsExecutable(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)
	step := NewExecutionStep("Step 1", "First step", "agent-1")
	plan.AddStep(step)

	// Not executable until approved
	assert.False(t, plan.IsExecutable())

	plan.Approve()
	assert.True(t, plan.IsExecutable())
}

func TestExecutionPlan_CanBeModified(t *testing.T) {
	plan := NewExecutionPlan("Test Plan", "Description", ExecutionPlanPriorityMedium)

	// Can modify when draft
	assert.True(t, plan.CanBeModified())

	// Cannot modify when completed
	plan.Status = ExecutionPlanStatusCompleted
	assert.False(t, plan.CanBeModified())
}
