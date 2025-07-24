package infrastructure

import (
	"context"
	"fmt"

	"neuromesh/internal/graph"
)

// GraphOrchestratorRepository implements orchestrator-specific graph operations
type GraphOrchestratorRepository struct {
	graph graph.Graph
}

// NewGraphOrchestratorRepository creates a new graph-based orchestrator repository
func NewGraphOrchestratorRepository(g graph.Graph) *GraphOrchestratorRepository {
	return &GraphOrchestratorRepository{
		graph: g,
	}
}

// EnsureSchema ensures that the required schema for Orchestrator domain is in place
func (r *GraphOrchestratorRepository) EnsureSchema(ctx context.Context) error {
	// Phase 1: Core Planning nodes - Analysis, ExecutionPlan, ExecutionStep, Decision

	// Analysis node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "Analysis", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for Analysis.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Analysis", "request_id"); err != nil {
		return fmt.Errorf("failed to create index for Analysis.request_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Analysis", "status"); err != nil {
		return fmt.Errorf("failed to create index for Analysis.status: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Analysis", "category"); err != nil {
		return fmt.Errorf("failed to create index for Analysis.category: %w", err)
	}

	// ExecutionPlan node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "ExecutionPlan", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for ExecutionPlan.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionPlan", "analysis_id"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionPlan.analysis_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionPlan", "status"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionPlan.status: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionPlan", "priority"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionPlan.priority: %w", err)
	}

	// ExecutionStep node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "ExecutionStep", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for ExecutionStep.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionStep", "plan_id"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionStep.plan_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionStep", "status"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionStep.status: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionStep", "assigned_agent_id"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionStep.assigned_agent_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionStep", "step_number"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionStep.step_number: %w", err)
	}

	// Decision node constraints and indexes
	if err := r.graph.CreateUniqueConstraint(ctx, "Decision", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for Decision.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Decision", "request_id"); err != nil {
		return fmt.Errorf("failed to create index for Decision.request_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Decision", "analysis_id"); err != nil {
		return fmt.Errorf("failed to create index for Decision.analysis_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Decision", "plan_id"); err != nil {
		return fmt.Errorf("failed to create index for Decision.plan_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Decision", "type"); err != nil {
		return fmt.Errorf("failed to create index for Decision.type: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "Decision", "status"); err != nil {
		return fmt.Errorf("failed to create index for Decision.status: %w", err)
	}

	// AgentCommunication node constraints and indexes (Phase 2)
	if err := r.graph.CreateUniqueConstraint(ctx, "AgentCommunication", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for AgentCommunication.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "AgentCommunication", "plan_id"); err != nil {
		return fmt.Errorf("failed to create index for AgentCommunication.plan_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "AgentCommunication", "from_agent_id"); err != nil {
		return fmt.Errorf("failed to create index for AgentCommunication.from_agent_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "AgentCommunication", "to_agent_id"); err != nil {
		return fmt.Errorf("failed to create index for AgentCommunication.to_agent_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "AgentCommunication", "message_type"); err != nil {
		return fmt.Errorf("failed to create index for AgentCommunication.message_type: %w", err)
	}

	// PlanModification node constraints and indexes (Phase 2)
	if err := r.graph.CreateUniqueConstraint(ctx, "PlanModification", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for PlanModification.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "PlanModification", "plan_id"); err != nil {
		return fmt.Errorf("failed to create index for PlanModification.plan_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "PlanModification", "modification_type"); err != nil {
		return fmt.Errorf("failed to create index for PlanModification.modification_type: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "PlanModification", "status"); err != nil {
		return fmt.Errorf("failed to create index for PlanModification.status: %w", err)
	}

	// ExecutionContext node constraints and indexes (Phase 3)
	if err := r.graph.CreateUniqueConstraint(ctx, "ExecutionContext", "id"); err != nil {
		return fmt.Errorf("failed to create unique constraint for ExecutionContext.id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionContext", "plan_id"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionContext.plan_id: %w", err)
	}

	if err := r.graph.CreateIndex(ctx, "ExecutionContext", "environment"); err != nil {
		return fmt.Errorf("failed to create index for ExecutionContext.environment: %w", err)
	}

	// Ensure core relationship types exist by creating schema relationships
	if err := r.ensureRelationshipTypes(ctx); err != nil {
		return fmt.Errorf("failed to ensure relationship types: %w", err)
	}

	return nil
}

// ensureRelationshipTypes creates schema relationships to register relationship types permanently
func (r *GraphOrchestratorRepository) ensureRelationshipTypes(ctx context.Context) error {
	// Core relationship types are defined by creating schema relationships
	// This ensures the relationship types exist in the graph schema

	schemaAnalysisID := "schema_analysis"
	schemaPlanID := "schema_execution_plan"
	schemaStepID := "schema_execution_step"
	schemaDecisionID := "schema_decision"

	// Create schema nodes for defining relationships
	schemaNodes := map[string]map[string]interface{}{
		"Analysis": {
			"id":         schemaAnalysisID,
			"request_id": "schema_request",
			"intent":     "Schema Definition Analysis",
			"status":     "schema",
		},
		"ExecutionPlan": {
			"id":          schemaPlanID,
			"analysis_id": schemaAnalysisID,
			"name":        "Schema Definition Plan",
			"status":      "schema",
		},
		"ExecutionStep": {
			"id":      schemaStepID,
			"plan_id": schemaPlanID,
			"name":    "Schema Definition Step",
			"status":  "schema",
		},
		"Decision": {
			"id":          schemaDecisionID,
			"request_id":  "schema_request",
			"analysis_id": schemaAnalysisID,
			"type":        "SCHEMA",
			"status":      "schema",
		},
	}

	// Create schema nodes if they don't exist
	for nodeType, nodeData := range schemaNodes {
		nodeID := nodeData["id"].(string)
		if err := r.graph.AddNode(ctx, nodeType, nodeID, nodeData); err != nil {
			// Node might already exist, check if it's actually an error
			existingNode, getErr := r.graph.GetNode(ctx, nodeType, nodeID)
			if getErr != nil || existingNode == nil {
				return fmt.Errorf("failed to create schema %s node: %w", nodeType, err)
			}
		}
	}

	// Create schema relationships to register relationship types
	schemaRelationships := []struct {
		fromType, fromID, toType, toID, relType string
	}{
		// Core Planning Relationships
		{"Analysis", schemaAnalysisID, "ExecutionPlan", schemaPlanID, "CREATES_PLAN"},
		{"Analysis", schemaAnalysisID, "Decision", schemaDecisionID, "INFORMS_DECISION"},
		{"ExecutionPlan", schemaPlanID, "ExecutionStep", schemaStepID, "CONTAINS_STEP"},
		{"ExecutionStep", schemaStepID, "ExecutionStep", schemaStepID, "DEPENDS_ON"},
		{"ExecutionStep", schemaStepID, "ExecutionStep", schemaStepID, "NEXT_STEP"},

		// Decision and Coordination
		{"Decision", schemaDecisionID, "ExecutionPlan", schemaPlanID, "APPROVES_PLAN"},
		{"Decision", schemaDecisionID, "ExecutionPlan", schemaPlanID, "MODIFIES_PLAN"},
	}

	for _, rel := range schemaRelationships {
		relData := map[string]interface{}{
			"schema":      true,
			"description": fmt.Sprintf("Schema definition relationship: %s", rel.relType),
		}

		if err := r.graph.AddEdge(ctx, rel.fromType, rel.fromID, rel.toType, rel.toID, rel.relType, relData); err != nil {
			// Relationship might already exist, which is fine
			continue
		}
	}

	return nil
}
