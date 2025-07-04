package domain

// DecisionInterface defines the common interface for all decisions
type DecisionInterface interface {
	GetType() string
	GetReasoning() string
	GetTimestamp() string
}

// DecisionResult wraps decisions that can be either planning or execution decisions
type DecisionResult struct {
	IsExecution       bool
	PlanningDecision  interface{} // Will be *planningdomain.Decision
	ExecutionDecision interface{} // Will be *executiondomain.Decision
}
