package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewExecutionStep(t *testing.T) {
	name := "Deploy Application"
	description := "Deploy app using kubectl"
	assignedAgent := "kubernetes-agent"

	step := NewExecutionStep(name, description, assignedAgent)

	assert.NotEmpty(t, step.ID)
	assert.Equal(t, name, step.Name)
	assert.Equal(t, description, step.Description)
	assert.Equal(t, assignedAgent, step.AssignedAgent)
	assert.Equal(t, ExecutionStepStatusPending, step.Status)
	assert.True(t, step.CanModify)
	assert.False(t, step.IsCritical)
	assert.Equal(t, 0, step.RetryCount)
	assert.Equal(t, 3, step.MaxRetries) // Default max retries
}

func TestExecutionStep_Validate(t *testing.T) {
	tests := []struct {
		name    string
		step    *ExecutionStep
		wantErr bool
	}{
		{
			name: "valid step",
			step: &ExecutionStep{
				ID:            "step-123",
				Name:          "Deploy",
				Description:   "Deploy application",
				AssignedAgent: "agent-1",
				Status:        ExecutionStepStatusPending,
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			step: &ExecutionStep{
				Name:          "Deploy",
				Description:   "Deploy application",
				AssignedAgent: "agent-1",
				Status:        ExecutionStepStatusPending,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			step: &ExecutionStep{
				ID:            "step-123",
				Description:   "Deploy application",
				AssignedAgent: "agent-1",
				Status:        ExecutionStepStatusPending,
			},
			wantErr: true,
		},
		{
			name: "empty assigned agent",
			step: &ExecutionStep{
				ID:          "step-123",
				Name:        "Deploy",
				Description: "Deploy application",
				Status:      ExecutionStepStatusPending,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			step: &ExecutionStep{
				ID:            "step-123",
				Name:          "Deploy",
				Description:   "Deploy application",
				AssignedAgent: "agent-1",
				Status:        ExecutionStepStatus("INVALID"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.step.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionStep_StatusTransitions(t *testing.T) {
	step := NewExecutionStep("Deploy", "Deploy app", "agent-1")

	// Test Assign
	step.Assign()
	assert.Equal(t, ExecutionStepStatusAssigned, step.Status)

	// Test Start
	err := step.Start()
	assert.NoError(t, err)
	assert.Equal(t, ExecutionStepStatusExecuting, step.Status)
	assert.NotNil(t, step.StartedAt)

	// Test Complete
	outputs := `{"result": "success"}`
	err = step.Complete(outputs)
	assert.NoError(t, err)
	assert.Equal(t, ExecutionStepStatusCompleted, step.Status)
	assert.Equal(t, outputs, step.Outputs)
	assert.NotNil(t, step.CompletedAt)
	assert.GreaterOrEqual(t, step.ActualDuration, 0) // Duration can be 0 for very fast execution
}

func TestExecutionStep_StatusTransitions_Invalid(t *testing.T) {
	step := NewExecutionStep("Deploy", "Deploy app", "agent-1")

	// Cannot start without being assigned
	err := step.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be assigned")

	// Cannot complete without executing
	err = step.Complete("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be executing")
}

func TestExecutionStep_Fail(t *testing.T) {
	step := NewExecutionStep("Deploy", "Deploy app", "agent-1")
	step.Assign()
	step.Start()

	errorMsg := "Deployment failed"
	step.Fail(errorMsg)

	assert.Equal(t, ExecutionStepStatusFailed, step.Status)
	assert.Equal(t, errorMsg, step.ErrorMessage)
	assert.NotNil(t, step.CompletedAt)
}

func TestExecutionStep_Retry(t *testing.T) {
	step := NewExecutionStep("Deploy", "Deploy app", "agent-1")
	step.MaxRetries = 2

	// First retry
	err := step.Retry()
	assert.NoError(t, err)
	assert.Equal(t, 1, step.RetryCount)
	assert.Equal(t, ExecutionStepStatusPending, step.Status)

	// Second retry
	err = step.Retry()
	assert.NoError(t, err)
	assert.Equal(t, 2, step.RetryCount)

	// Third retry should fail
	err = step.Retry()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum retries exceeded")
}

func TestExecutionStep_CanRetry(t *testing.T) {
	step := NewExecutionStep("Deploy", "Deploy app", "agent-1")
	step.MaxRetries = 2
	step.Status = ExecutionStepStatusFailed

	// Can retry when failed and under limit
	assert.True(t, step.CanRetry())

	// Cannot retry when at limit
	step.RetryCount = 2
	assert.False(t, step.CanRetry())

	// Cannot retry when completed
	step.Status = ExecutionStepStatusCompleted
	step.RetryCount = 0
	assert.False(t, step.CanRetry())
}

func TestExecutionStep_IsComplete(t *testing.T) {
	step := NewExecutionStep("Deploy", "Deploy app", "agent-1")

	assert.False(t, step.IsComplete())

	step.Status = ExecutionStepStatusCompleted
	assert.True(t, step.IsComplete())

	step.Status = ExecutionStepStatusFailed
	assert.True(t, step.IsComplete())

	step.Status = ExecutionStepStatusSkipped
	assert.True(t, step.IsComplete())
}

func TestExecutionStep_CanBeModified(t *testing.T) {
	step := NewExecutionStep("Deploy", "Deploy app", "agent-1")

	// Can modify when pending and CanModify is true
	assert.True(t, step.CanBeModified())

	// Cannot modify when executing
	step.Status = ExecutionStepStatusExecuting
	assert.False(t, step.CanBeModified())

	// Cannot modify when CanModify is false
	step.Status = ExecutionStepStatusPending
	step.CanModify = false
	assert.False(t, step.CanBeModified())
}
