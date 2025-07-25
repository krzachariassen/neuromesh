package domain

// ExecutionStats represents statistics about an execution plan's progress
type ExecutionStats struct {
	TotalSteps        int `json:"total_steps"`
	CompletedSteps    int `json:"completed_steps"`
	PendingSteps      int `json:"pending_steps"`
	SuccessfulResults int `json:"successful_results"`
	FailedResults     int `json:"failed_results"`
	PartialResults    int `json:"partial_results"`
}

// CalculateCompletionPercentage returns the completion percentage (0-100)
func (s *ExecutionStats) CalculateCompletionPercentage() float64 {
	if s.TotalSteps == 0 {
		return 0.0
	}
	return float64(s.CompletedSteps) / float64(s.TotalSteps) * 100.0
}

// IsComplete returns true if all steps are completed
func (s *ExecutionStats) IsComplete() bool {
	return s.TotalSteps > 0 && s.CompletedSteps == s.TotalSteps && s.PendingSteps == 0
}

// HasFailures returns true if there are any failed results
func (s *ExecutionStats) HasFailures() bool {
	return s.FailedResults > 0
}
