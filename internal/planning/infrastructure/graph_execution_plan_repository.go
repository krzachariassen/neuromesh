package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"neuromesh/internal/graph"
	"neuromesh/internal/planning/domain"
)

// GraphExecutionPlanRepository implements ExecutionPlanRepository using Neo4j graph
type GraphExecutionPlanRepository struct {
	graph graph.Graph
}

// NewGraphExecutionPlanRepository creates a new graph-based execution plan repository
func NewGraphExecutionPlanRepository(g graph.Graph) *GraphExecutionPlanRepository {
	return &GraphExecutionPlanRepository{
		graph: g,
	}
}

// EnsureSchema ensures that the required schema for ExecutionPlan domain is in place
func (r *GraphExecutionPlanRepository) EnsureSchema(ctx context.Context) error {
	// ExecutionPlan node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "execution_plan", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for execution_plan.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "execution_plan", "status"); err != nil {
		return fmt.Errorf("failed to create index for execution_plan.status: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "execution_plan", "priority"); err != nil {
		return fmt.Errorf("failed to create index for execution_plan.priority: %w", err)
	}

	// ExecutionStep node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "execution_step", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for execution_step.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "execution_step", "status"); err != nil {
		return fmt.Errorf("failed to create index for execution_step.status: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "execution_step", "step_number"); err != nil {
		return fmt.Errorf("failed to create index for execution_step.step_number: %w", err)
	}

	return nil
}

// Create persists a new execution plan to the graph
func (r *GraphExecutionPlanRepository) Create(ctx context.Context, plan *domain.ExecutionPlan) error {
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("invalid execution plan: %w", err)
	}

	// Create the execution plan node
	planData := plan.ToMap()

	if err := r.graph.AddNode(ctx, "execution_plan", plan.ID, planData); err != nil {
		return fmt.Errorf("failed to create execution plan node: %w", err)
	}

	// Create step nodes and relationships
	for _, step := range plan.Steps {
		if err := r.AddStep(ctx, step); err != nil {
			return fmt.Errorf("failed to create step %s: %w", step.ID, err)
		}

		// Create CONTAINS_STEP relationship
		relationshipProps := map[string]interface{}{
			"order": step.StepNumber,
		}
		if err := r.graph.AddEdge(ctx, "execution_plan", plan.ID, "execution_step", step.ID, "CONTAINS_STEP", relationshipProps); err != nil {
			return fmt.Errorf("failed to create CONTAINS_STEP relationship: %w", err)
		}

		// Create ASSIGNED_TO relationship to agent
		if step.AssignedAgent != "" {
			if err := r.graph.AddEdge(ctx, "execution_step", step.ID, "agent", step.AssignedAgent, "ASSIGNED_TO", nil); err != nil {
				return fmt.Errorf("failed to create ASSIGNED_TO relationship: %w", err)
			}
		}
	}

	return nil
}

// GetByID retrieves an execution plan by its ID
func (r *GraphExecutionPlanRepository) GetByID(ctx context.Context, id string) (*domain.ExecutionPlan, error) {
	planData, err := r.graph.GetNode(ctx, "execution_plan", id)
	if err != nil {
		if strings.Contains(err.Error(), "node not found") {
			return nil, fmt.Errorf("execution plan %s not found", id)
		}
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}

	plan, err := r.mapToExecutionPlan(planData)
	if err != nil {
		return nil, fmt.Errorf("failed to map execution plan: %w", err)
	}

	// Load steps
	steps, err := r.GetStepsByPlanID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load steps: %w", err)
	}
	plan.Steps = steps

	return plan, nil
}

// GetByAnalysisID retrieves an execution plan by analysis ID through graph relationship
func (r *GraphExecutionPlanRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*domain.ExecutionPlan, error) {
	// Get edges with target information from the analysis node
	edges, err := r.graph.GetEdgesWithTargets(ctx, "analysis", analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges from analysis: %w", err)
	}

	// Find the CREATES_PLAN relationship
	var planID string
	for _, edge := range edges {
		if edgeType, ok := edge["type"].(string); ok && edgeType == "CREATES_PLAN" {
			if targetType, ok := edge["target_type"].(string); ok && targetType == "execution_plan" {
				if targetID, ok := edge["target_id"].(string); ok {
					planID = targetID
					break
				}
			}
		}
	}

	if planID == "" {
		return nil, fmt.Errorf("no execution plan found for analysis %s", analysisID)
	}

	return r.GetByID(ctx, planID)
}

// Update updates an existing execution plan
func (r *GraphExecutionPlanRepository) Update(ctx context.Context, plan *domain.ExecutionPlan) error {
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("invalid execution plan: %w", err)
	}

	planData := plan.ToMap()

	if err := r.graph.UpdateNode(ctx, "execution_plan", plan.ID, planData); err != nil {
		return fmt.Errorf("failed to update execution plan: %w", err)
	}

	return nil
}

// LinkToAnalysis creates a relationship between analysis and execution plan
func (r *GraphExecutionPlanRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	// Create the CREATES_PLAN relationship edge
	if err := r.graph.AddEdge(ctx, "analysis", analysisID, "execution_plan", planID, "CREATES_PLAN", nil); err != nil {
		return fmt.Errorf("failed to create CREATES_PLAN relationship: %w", err)
	}

	return nil
}

// GetStepsByPlanID retrieves all steps for a given plan ID
func (r *GraphExecutionPlanRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*domain.ExecutionStep, error) {
	// Query for all execution steps that have the matching plan_id
	stepNodes, err := r.graph.QueryNodes(ctx, "execution_step", map[string]interface{}{
		"plan_id": planID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query steps by plan ID: %w", err)
	}

	// Convert step nodes to ExecutionStep objects
	var steps []*domain.ExecutionStep
	for _, stepData := range stepNodes {
		step, err := r.mapToExecutionStep(stepData)
		if err != nil {
			return nil, fmt.Errorf("failed to map execution step: %w", err)
		}
		steps = append(steps, step)
	}

	// Sort by step number
	for i := 0; i < len(steps)-1; i++ {
		for j := i + 1; j < len(steps); j++ {
			if steps[i].StepNumber > steps[j].StepNumber {
				steps[i], steps[j] = steps[j], steps[i]
			}
		}
	}

	return steps, nil
}

// AddStep adds a new step to the graph
func (r *GraphExecutionPlanRepository) AddStep(ctx context.Context, step *domain.ExecutionStep) error {
	if err := step.Validate(); err != nil {
		return fmt.Errorf("invalid execution step: %w", err)
	}

	stepData := step.ToMap()

	if err := r.graph.AddNode(ctx, "execution_step", step.ID, stepData); err != nil {
		return fmt.Errorf("failed to create execution step node: %w", err)
	}

	return nil
}

// UpdateStep updates an existing step
func (r *GraphExecutionPlanRepository) UpdateStep(ctx context.Context, step *domain.ExecutionStep) error {
	if err := step.Validate(); err != nil {
		return fmt.Errorf("invalid execution step: %w", err)
	}

	stepData := step.ToMap()

	if err := r.graph.UpdateNode(ctx, "execution_step", step.ID, stepData); err != nil {
		return fmt.Errorf("failed to update execution step: %w", err)
	}

	return nil
}

// AssignStepToAgent updates the agent assignment for a step
func (r *GraphExecutionPlanRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	// Get current step data to check for existing assignment
	stepData, err := r.graph.GetNode(ctx, "execution_step", stepID)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}

	// If there's already an assigned agent, remove the old relationship
	if currentAgent, ok := stepData["assigned_agent"].(string); ok && currentAgent != "" {
		// Delete the old relationship
		if err := r.graph.DeleteEdge(ctx, "execution_step", stepID, "agent", currentAgent, "ASSIGNED_TO"); err != nil {
			// Log the error but continue - the relationship might not exist
			// This is acceptable for our use case
		}
	}

	// Create new ASSIGNED_TO relationship
	if err := r.graph.AddEdge(ctx, "execution_step", stepID, "agent", agentID, "ASSIGNED_TO", nil); err != nil {
		return fmt.Errorf("failed to create new ASSIGNED_TO relationship: %w", err)
	}

	// Update step's assigned agent field
	updatedStepData := map[string]interface{}{
		"assigned_agent": agentID,
	}
	if err := r.graph.UpdateNode(ctx, "execution_step", stepID, updatedStepData); err != nil {
		return fmt.Errorf("failed to update step assigned agent: %w", err)
	}

	return nil
}

// Helper method to map graph data to ExecutionPlan
func (r *GraphExecutionPlanRepository) mapToExecutionPlan(data map[string]interface{}) (*domain.ExecutionPlan, error) {
	plan := &domain.ExecutionPlan{}

	if id, ok := data["id"].(string); ok {
		plan.ID = id
	} else {
		return nil, fmt.Errorf("missing or invalid id")
	}

	if name, ok := data["name"].(string); ok {
		plan.Name = name
	}

	if description, ok := data["description"].(string); ok {
		plan.Description = description
	}

	if status, ok := data["status"].(string); ok {
		plan.Status = domain.ExecutionPlanStatus(status)
	}

	if priority, ok := data["priority"].(string); ok {
		plan.Priority = domain.ExecutionPlanPriority(priority)
	}

	if canModify, ok := data["can_modify"].(bool); ok {
		plan.CanModify = canModify
	}

	// Handle time fields
	if createdAt, ok := data["created_at"].(time.Time); ok {
		plan.CreatedAt = createdAt
	}

	if approvedAt, ok := data["approved_at"].(time.Time); ok {
		plan.ApprovedAt = &approvedAt
	}

	if startedAt, ok := data["started_at"].(time.Time); ok {
		plan.StartedAt = &startedAt
	}

	if completedAt, ok := data["completed_at"].(time.Time); ok {
		plan.CompletedAt = &completedAt
	}

	if estimatedDuration, ok := data["estimated_duration"].(int); ok {
		plan.EstimatedDuration = estimatedDuration
	} else if estimatedDuration, ok := data["estimated_duration"].(float64); ok {
		plan.EstimatedDuration = int(estimatedDuration)
	}

	if actualDuration, ok := data["actual_duration"].(int); ok {
		plan.ActualDuration = actualDuration
	} else if actualDuration, ok := data["actual_duration"].(float64); ok {
		plan.ActualDuration = int(actualDuration)
	}

	plan.Steps = make([]*domain.ExecutionStep, 0)

	return plan, nil
}

// Helper method to map graph data to ExecutionStep
func (r *GraphExecutionPlanRepository) mapToExecutionStep(data map[string]interface{}) (*domain.ExecutionStep, error) {
	step := &domain.ExecutionStep{}

	if id, ok := data["id"].(string); ok {
		step.ID = id
	} else {
		return nil, fmt.Errorf("missing or invalid id")
	}

	if planID, ok := data["plan_id"].(string); ok {
		step.PlanID = planID
	}

	if stepNumber, ok := data["step_number"].(int); ok {
		step.StepNumber = stepNumber
	}

	if name, ok := data["name"].(string); ok {
		step.Name = name
	}

	if description, ok := data["description"].(string); ok {
		step.Description = description
	}

	if assignedAgent, ok := data["assigned_agent"].(string); ok {
		step.AssignedAgent = assignedAgent
	}

	if status, ok := data["status"].(string); ok {
		step.Status = domain.ExecutionStepStatus(status)
	}

	if inputs, ok := data["inputs"].(string); ok {
		step.Inputs = inputs
	}

	if outputs, ok := data["outputs"].(string); ok {
		step.Outputs = outputs
	}

	if canModify, ok := data["can_modify"].(bool); ok {
		step.CanModify = canModify
	}

	if isCritical, ok := data["is_critical"].(bool); ok {
		step.IsCritical = isCritical
	}

	if retryCount, ok := data["retry_count"].(int); ok {
		step.RetryCount = retryCount
	}

	if maxRetries, ok := data["max_retries"].(int); ok {
		step.MaxRetries = maxRetries
	}

	// Handle time fields
	if startedAt, ok := data["started_at"].(time.Time); ok {
		step.StartedAt = &startedAt
	}

	if completedAt, ok := data["completed_at"].(time.Time); ok {
		step.CompletedAt = &completedAt
	}

	// Handle numeric fields with type conversion
	if estimatedDuration, ok := data["estimated_duration"].(int); ok {
		step.EstimatedDuration = estimatedDuration
	} else if estimatedDuration, ok := data["estimated_duration"].(float64); ok {
		step.EstimatedDuration = int(estimatedDuration)
	}

	if actualDuration, ok := data["actual_duration"].(int); ok {
		step.ActualDuration = actualDuration
	} else if actualDuration, ok := data["actual_duration"].(float64); ok {
		step.ActualDuration = int(actualDuration)
	}

	if stepNumber, ok := data["step_number"].(float64); ok {
		step.StepNumber = int(stepNumber)
	}

	if retryCount, ok := data["retry_count"].(float64); ok {
		step.RetryCount = int(retryCount)
	}

	if maxRetries, ok := data["max_retries"].(float64); ok {
		step.MaxRetries = int(maxRetries)
	}

	if errorMessage, ok := data["error_message"].(string); ok {
		step.ErrorMessage = errorMessage
	}

	return step, nil
}
