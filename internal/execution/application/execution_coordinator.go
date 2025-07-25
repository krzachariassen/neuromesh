package application

import (
	"context"
	"fmt"

	"neuromesh/internal/execution/domain"
	planningDomain "neuromesh/internal/planning/domain"
)

// ExecutionCoordinator coordinates the execution of multi-agent plans and triggers synthesis
type ExecutionCoordinator struct {
	executionRepo planningDomain.ExecutionPlanRepository
	synthesizer   domain.ResultSynthesizer
}

// NewExecutionCoordinator creates a new execution coordinator
func NewExecutionCoordinator(
	executionRepo planningDomain.ExecutionPlanRepository,
	synthesizer domain.ResultSynthesizer,
) *ExecutionCoordinator {
	return &ExecutionCoordinator{
		executionRepo: executionRepo,
		synthesizer:   synthesizer,
	}
}

// IsExecutionPlanComplete checks if all steps in an execution plan have completed successfully
func (c *ExecutionCoordinator) IsExecutionPlanComplete(ctx context.Context, planID string) (bool, error) {
	// Get all execution steps for the plan
	steps, err := c.executionRepo.GetStepsByPlanID(ctx, planID)
	if err != nil {
		return false, fmt.Errorf("failed to get execution steps: %w", err)
	}

	// Check each step
	for _, step := range steps {
		// Skip steps that are not completed
		if step.Status != planningDomain.ExecutionStepStatusCompleted {
			return false, nil
		}

		// Check if step has successful results
		results, err := c.executionRepo.GetAgentResultsByExecutionStep(ctx, step.ID)
		if err != nil {
			return false, fmt.Errorf("failed to get agent results for step %s: %w", step.ID, err)
		}

		// Check if any result is not successful
		for _, result := range results {
			if result.Status != domain.AgentResultStatusSuccess {
				return false, nil
			}
		}
	}

	return true, nil
}

// TriggerSynthesisWhenComplete triggers synthesis if the execution plan is complete
func (c *ExecutionCoordinator) TriggerSynthesisWhenComplete(ctx context.Context, planID string) (string, error) {
	// Check if execution plan is complete
	isComplete, err := c.IsExecutionPlanComplete(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("failed to check execution plan completion: %w", err)
	}

	// If not complete, return empty result
	if !isComplete {
		return "", nil
	}

	// Trigger synthesis
	synthesizedResult, err := c.synthesizer.SynthesizeResults(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("failed to synthesize results: %w", err)
	}

	return synthesizedResult, nil
}

// HandlePartialCompletion analyzes partial completion and returns execution statistics
func (c *ExecutionCoordinator) HandlePartialCompletion(ctx context.Context, planID string) (*domain.ExecutionStats, error) {
	// Get all execution steps for the plan
	steps, err := c.executionRepo.GetStepsByPlanID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution steps: %w", err)
	}

	stats := &domain.ExecutionStats{
		TotalSteps: len(steps),
	}

	// Analyze each step
	for _, step := range steps {
		switch step.Status {
		case planningDomain.ExecutionStepStatusCompleted:
			stats.CompletedSteps++
		case planningDomain.ExecutionStepStatusPending, planningDomain.ExecutionStepStatusAssigned:
			stats.PendingSteps++
		}

		// Only analyze results for completed steps
		if step.Status == planningDomain.ExecutionStepStatusCompleted {
			results, err := c.executionRepo.GetAgentResultsByExecutionStep(ctx, step.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get agent results for step %s: %w", step.ID, err)
			}

			// Count result types
			for _, result := range results {
				switch result.Status {
				case domain.AgentResultStatusSuccess:
					stats.SuccessfulResults++
				case domain.AgentResultStatusFailed:
					stats.FailedResults++
				case domain.AgentResultStatusPartial:
					stats.PartialResults++
				}
			}
		}
	}

	return stats, nil
}
