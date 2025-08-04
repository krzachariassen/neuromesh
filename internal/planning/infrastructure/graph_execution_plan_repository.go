package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	executionDomain "neuromesh/internal/execution/domain"
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

// Agent Result operations - Implementation for graph-native result synthesis

// StoreAgentResult stores an agent result in the graph with relationship to execution step
func (r *GraphExecutionPlanRepository) StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error {
	// Validate the agent result before storing
	if err := result.Validate(); err != nil {
		return fmt.Errorf("agent result validation failed: %w", err)
	}

	// Serialize metadata to JSON string for Neo4j storage
	var metadataJSON string
	if result.Metadata != nil && len(result.Metadata) > 0 {
		metadataBytes, err := json.Marshal(result.Metadata)
		if err != nil {
			return fmt.Errorf("failed to serialize metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	} else {
		metadataJSON = "{}"
	}

	// Create the agent result node properties
	properties := map[string]interface{}{
		"execution_step_id": result.ExecutionStepID,
		"agent_id":          result.AgentID,
		"content":           result.Content,
		"status":            string(result.Status),
		"metadata":          metadataJSON,
		"timestamp":         result.Timestamp.Format(time.RFC3339Nano),
	}

	// Create the agent result node
	if err := r.graph.AddNode(ctx, "agent_result", result.ID, properties); err != nil {
		return fmt.Errorf("failed to create agent result node: %w", err)
	}

	// Create relationship from execution step to agent result
	if err := r.graph.AddEdge(ctx, "execution_step", result.ExecutionStepID, "agent_result", result.ID, "HAS_RESULT", nil); err != nil {
		return fmt.Errorf("failed to create HAS_RESULT relationship: %w", err)
	}

	return nil
}

// GetAgentResultByID retrieves a specific agent result by its ID
func (r *GraphExecutionPlanRepository) GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error) {
	resultData, err := r.graph.GetNode(ctx, "agent_result", resultID)
	if err != nil {
		if strings.Contains(err.Error(), "node not found") {
			return nil, fmt.Errorf("agent result %s not found", resultID)
		}
		return nil, fmt.Errorf("failed to get agent result: %w", err)
	}

	result, err := r.mapNodeDataToAgentResult(resultData)
	if err != nil {
		return nil, fmt.Errorf("failed to map agent result: %w", err)
	}

	return result, nil
}

// GetAgentResultsByExecutionStep retrieves all agent results for a specific execution step
func (r *GraphExecutionPlanRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error) {
	// Get all edges from the execution step to agent results
	edges, err := r.graph.GetEdgesWithTargets(ctx, "execution_step", stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges for execution step %s: %w", stepID, err)
	}

	results := make([]*executionDomain.AgentResult, 0, len(edges))
	for _, edge := range edges {
		// Only process HAS_RESULT relationships to agent_result nodes
		if edgeType, ok := edge["type"].(string); ok && edgeType == "HAS_RESULT" {
			if targetType, ok := edge["target_type"].(string); ok && targetType == "agent_result" {
				if targetID, ok := edge["target_id"].(string); ok {
					// Get the agent result node
					resultData, err := r.graph.GetNode(ctx, "agent_result", targetID)
					if err != nil {
						return nil, fmt.Errorf("failed to get agent result node %s: %w", targetID, err)
					}

					result, err := r.mapNodeDataToAgentResult(resultData)
					if err != nil {
						return nil, fmt.Errorf("failed to map agent result %s: %w", targetID, err)
					}
					results = append(results, result)
				}
			}
		}
	}

	return results, nil
}

// GetAgentResultsByExecutionPlan retrieves all agent results for an entire execution plan
func (r *GraphExecutionPlanRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error) {
	// First get all execution steps for the plan
	planEdges, err := r.graph.GetEdgesWithTargets(ctx, "execution_plan", planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges for execution plan %s: %w", planID, err)
	}

	results := make([]*executionDomain.AgentResult, 0)

	// For each execution step, get its agent results
	for _, planEdge := range planEdges {
		if edgeType, ok := planEdge["type"].(string); ok && edgeType == "CONTAINS_STEP" {
			if targetType, ok := planEdge["target_type"].(string); ok && targetType == "execution_step" {
				if stepID, ok := planEdge["target_id"].(string); ok {
					stepResults, err := r.GetAgentResultsByExecutionStep(ctx, stepID)
					if err != nil {
						return nil, fmt.Errorf("failed to get agent results for step %s in plan %s: %w", stepID, planID, err)
					}
					results = append(results, stepResults...)
				}
			}
		}
	}

	return results, nil
}

// mapNodeDataToAgentResult converts node data to an AgentResult domain object
func (r *GraphExecutionPlanRepository) mapNodeDataToAgentResult(nodeData map[string]interface{}) (*executionDomain.AgentResult, error) {
	id, ok := nodeData["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid id in agent result")
	}

	executionStepID, ok := nodeData["execution_step_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid execution_step_id in agent result")
	}

	agentID, ok := nodeData["agent_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid agent_id in agent result")
	}

	content, ok := nodeData["content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid content in agent result")
	}

	statusStr, ok := nodeData["status"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid status in agent result")
	}
	status := executionDomain.AgentResultStatus(statusStr)

	timestampStr, ok := nodeData["timestamp"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid timestamp in agent result")
	}

	timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Handle metadata - deserialize from JSON string
	var metadata map[string]interface{}
	if metadataStr, exists := nodeData["metadata"]; exists && metadataStr != nil {
		if metadataJSON, ok := metadataStr.(string); ok {
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
			}
		}
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	result := &executionDomain.AgentResult{
		ID:              id,
		ExecutionStepID: executionStepID,
		AgentID:         agentID,
		Content:         content,
		Status:          status,
		Metadata:        metadata,
		Timestamp:       timestamp,
	}

	return result, nil
}

// GetPlanIDByCorrelationID finds the execution plan ID for a given correlation ID
// This implements the graph-native approach for linking agent results to execution plans
func (r *GraphExecutionPlanRepository) GetPlanIDByCorrelationID(ctx context.Context, correlationID string) (string, error) {
	// In our graph model, correlation IDs are stored as execution step IDs
	// First, find the execution step with this correlation ID
	stepFilters := map[string]interface{}{
		"id": correlationID,
	}

	steps, err := r.graph.QueryNodes(ctx, "execution_step", stepFilters)
	if err != nil {
		return "", fmt.Errorf("failed to query execution step by correlation ID: %w", err)
	}

	if len(steps) == 0 {
		// No matching execution step found
		return "", nil
	}

	// Get the first matching step
	step := steps[0]

	// Extract plan_id from the step properties
	planID, exists := step["plan_id"]
	if !exists {
		return "", fmt.Errorf("execution step %s does not have plan_id property", correlationID)
	}

	planIDStr, ok := planID.(string)
	if !ok {
		return "", fmt.Errorf("plan_id is not a string: %T", planID)
	}

	return planIDStr, nil
}

// StoreConversationSummary stores a conversation summary in the graph with conversation-aware relationships
func (r *GraphExecutionPlanRepository) StoreConversationSummary(ctx context.Context, summary *executionDomain.ConversationSummary) error {
	if summary == nil {
		return fmt.Errorf("conversation summary cannot be nil")
	}

	if summary.ID == "" {
		return fmt.Errorf("conversation summary ID cannot be empty")
	}

	if summary.PlanID == "" {
		return fmt.Errorf("conversation summary plan ID cannot be empty")
	}

	// Create conversation summary node properties
	properties := map[string]interface{}{
		"plan_id":         summary.PlanID,
		"conversation_id": summary.ConversationID,
		"summary":         summary.Summary,    // Full technical summary (was Content)
		"user_result":     summary.UserResult, // User-friendly answer (was UserAnswer)
		"status":          string(summary.Status),
		"created_at":      summary.CreatedAt.Format(time.RFC3339Nano),
	}

	// Add completion timestamp if available
	if summary.CompletedAt != nil {
		properties["completed_at"] = summary.CompletedAt.Format(time.RFC3339Nano)
	}

	// Add error message if present
	if summary.ErrorMessage != "" {
		properties["error_message"] = summary.ErrorMessage
	}

	// Add metadata if present (serialize to JSON for Neo4j compatibility)
	if len(summary.Metadata) > 0 {
		if metadataJSON, err := json.Marshal(summary.Metadata); err == nil {
			properties["metadata"] = string(metadataJSON)
		}
		// If marshaling fails, skip metadata - it's supplementary information
	}

	// Create the conversation summary node
	if err := r.graph.AddNode(ctx, "conversation_summary", summary.ID, properties); err != nil {
		return fmt.Errorf("failed to create conversation summary node: %w", err)
	}

	// Create relationship from conversation to conversation summary (primary relationship)
	if summary.ConversationID != "" {
		if err := r.graph.AddEdge(ctx, "conversation", summary.ConversationID, "conversation_summary", summary.ID, "HAS_CONVERSATION_SUMMARY", nil); err != nil {
			return fmt.Errorf("failed to create conversation HAS_CONVERSATION_SUMMARY relationship: %w", err)
		}
	}

	// Create relationship from conversation summary to execution plan (traceability)
	if err := r.graph.AddEdge(ctx, "conversation_summary", summary.ID, "execution_plan", summary.PlanID, "GENERATED_FROM", nil); err != nil {
		return fmt.Errorf("failed to create GENERATED_FROM relationship: %w", err)
	}

	return nil
}

// GetConversationSummaryByPlanID retrieves a conversation summary by plan ID
func (r *GraphExecutionPlanRepository) GetConversationSummaryByPlanID(ctx context.Context, planID string) (*executionDomain.ConversationSummary, error) {
	if planID == "" {
		return nil, fmt.Errorf("plan ID cannot be empty")
	}

	// Get edges with targets from the execution plan
	planEdges, err := r.graph.GetEdgesWithTargets(ctx, "execution_plan", planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges for execution plan %s: %w", planID, err)
	}

	// Look for conversation summary edge
	for _, planEdge := range planEdges {
		if edgeType, ok := planEdge["type"].(string); ok && edgeType == "HAS_CONVERSATION_SUMMARY" {
			if targetType, ok := planEdge["target_type"].(string); ok && targetType == "conversation_summary" {
				if summaryID, ok := planEdge["target_id"].(string); ok {
					// Get the conversation summary node
					summaryData, err := r.graph.GetNode(ctx, "conversation_summary", summaryID)
					if err != nil {
						if strings.Contains(err.Error(), "node not found") {
							return nil, nil // No conversation summary found
						}
						return nil, fmt.Errorf("failed to get conversation summary node %s: %w", summaryID, err)
					}

					return r.mapNodeDataToConversationSummary(summaryData)
				}
			}
		}
	}

	return nil, nil // No conversation summary found
}

// mapNodeDataToConversationSummary converts Neo4j node data to ConversationSummary domain entity
func (r *GraphExecutionPlanRepository) mapNodeDataToConversationSummary(nodeData map[string]interface{}) (*executionDomain.ConversationSummary, error) {
	id, ok := nodeData["id"].(string)
	if !ok {
		return nil, fmt.Errorf("conversation summary id is not a string")
	}

	planID, ok := nodeData["plan_id"].(string)
	if !ok {
		return nil, fmt.Errorf("conversation summary plan_id is not a string")
	}

	conversationID, ok := nodeData["conversation_id"].(string)
	if !ok {
		return nil, fmt.Errorf("conversation summary conversation_id is not a string")
	}

	summary, ok := nodeData["summary"].(string)
	if !ok {
		return nil, fmt.Errorf("conversation summary summary is not a string")
	}

	userResult, ok := nodeData["user_result"].(string)
	if !ok {
		return nil, fmt.Errorf("conversation summary user_result is not a string")
	}

	statusStr, ok := nodeData["status"].(string)
	if !ok {
		return nil, fmt.Errorf("conversation summary status is not a string")
	}

	status := executionDomain.ConversationSummaryStatus(statusStr)

	createdAtData, ok := nodeData["created_at"]
	var createdAt time.Time
	if ok {
		switch v := createdAtData.(type) {
		case time.Time:
			createdAt = v
		case string:
			var err error
			createdAt, err = time.Parse(time.RFC3339, v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at: %w", err)
			}
		}
	}

	// Handle optional fields
	var completedAt *time.Time
	if completedAtData, exists := nodeData["completed_at"]; exists && completedAtData != nil {
		switch v := completedAtData.(type) {
		case time.Time:
			completedAt = &v
		case string:
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				completedAt = &parsed
			}
		}
	}

	errorMessage := ""
	if errMsg, exists := nodeData["error_message"]; exists && errMsg != nil {
		if errStr, ok := errMsg.(string); ok {
			errorMessage = errStr
		}
	}

	metadata := make(map[string]interface{})
	if metaData, exists := nodeData["metadata"]; exists && metaData != nil {
		if metaMap, ok := metaData.(map[string]interface{}); ok {
			metadata = metaMap
		}
	}

	return &executionDomain.ConversationSummary{
		ID:             id,
		ConversationID: conversationID,
		PlanID:         planID,
		Summary:        summary,
		UserResult:     userResult,
		Status:         status,
		CreatedAt:      createdAt,
		CompletedAt:    completedAt,
		ErrorMessage:   errorMessage,
		Metadata:       metadata,
	}, nil
}

// Delete removes an execution plan from the graph (implementing unified repository)
func (r *GraphExecutionPlanRepository) Delete(ctx context.Context, id string) error {
	// First, remove all related edges and nodes
	// Get all execution steps for this plan and delete them
	stepNodes, err := r.graph.QueryNodes(ctx, "execution_step", map[string]interface{}{
		"execution_plan_id": id,
	})
	if err != nil {
		return fmt.Errorf("failed to query execution steps: %w", err)
	}

	// Delete each step and its relationships
	for _, stepData := range stepNodes {
		stepID, ok := stepData["id"].(string)
		if !ok {
			continue // Skip invalid step IDs
		}
		if err := r.graph.DeleteNode(ctx, "execution_step", stepID); err != nil {
			return fmt.Errorf("failed to delete execution step %s: %w", stepID, err)
		}
	}

	// Delete the execution plan node
	if err := r.graph.DeleteNode(ctx, "execution_plan", id); err != nil {
		return fmt.Errorf("failed to delete execution plan: %w", err)
	}

	return nil
}

// GetByRequestID retrieves execution plans by request ID (from former PlanningResultRepository)
func (r *GraphExecutionPlanRepository) GetByRequestID(ctx context.Context, requestID string) ([]*domain.ExecutionPlan, error) {
	// Query for execution plan with the given request_id
	planNodes, err := r.graph.QueryNodes(ctx, "execution_plan", map[string]interface{}{
		"request_id": requestID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query execution plan by request ID: %w", err)
	}

	if len(planNodes) == 0 {
		return []*domain.ExecutionPlan{}, nil
	}

	// Get all matching plans
	var plans []*domain.ExecutionPlan
	for _, planData := range planNodes {
		if id, ok := planData["id"].(string); ok {
			plan, err := r.GetByID(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("failed to get plan %s: %w", id, err)
			}
			if plan != nil {
				plans = append(plans, plan)
			}
		}
	}

	return plans, nil
}

// LinkToRequest links execution plan to a request (from former PlanningResultRepository)
func (r *GraphExecutionPlanRepository) LinkToRequest(ctx context.Context, planID, requestID string) error {
	return r.graph.AddEdge(ctx, "execution_plan", planID, "request", requestID, "LINKED_TO_REQUEST", nil)
}

// LinkToConversation is deprecated - use Conversation domain's LinkExecutionPlan instead
// This method now does nothing to avoid duplicate relationship creation
// Following DDD: Conversation domain owns the conversation->execution_plan relationship
func (r *GraphExecutionPlanRepository) LinkToConversation(ctx context.Context, planID, conversationID string) error {
	// No-op: Conversation domain handles this relationship via LinkExecutionPlan
	// This maintains interface compatibility while avoiding duplicate relationships
	return nil
}

// GetConversationIDByPlanID retrieves the conversation ID linked to an execution plan
func (r *GraphExecutionPlanRepository) GetConversationIDByPlanID(ctx context.Context, planID string) (string, error) {
	if planID == "" {
		return "", fmt.Errorf("plan ID cannot be empty")
	}

	// Get edges that point TO the execution plan (incoming edges: conversation -> execution_plan)
	planEdges, err := r.graph.GetEdgesWithSources(ctx, "execution_plan", planID)
	if err != nil {
		return "", fmt.Errorf("failed to get edges for execution plan %s: %w", planID, err)
	}

	// Look for HAS_EXECUTION_PLAN edge from conversation
	for _, planEdge := range planEdges {
		if edgeType, ok := planEdge["type"].(string); ok && edgeType == "HAS_EXECUTION_PLAN" {
			if sourceType, ok := planEdge["source_type"].(string); ok && sourceType == "Conversation" {
				if conversationID, ok := planEdge["source_id"].(string); ok {
					return conversationID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no conversation linked to execution plan %s", planID)
}
