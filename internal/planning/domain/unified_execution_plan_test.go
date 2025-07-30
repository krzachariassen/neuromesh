package domain

import (
	"testing"
)

// TestUnifiedExecutionPlan_TDD validates the consolidated ExecutionPlan entity
func TestUnifiedExecutionPlan_TDD(t *testing.T) {
	t.Run("RED: should create unified execution plan with both planning and execution data", func(t *testing.T) {
		// This test will fail initially - we need to add planning fields to ExecutionPlan

		planID := "plan-123"
		requestID := "req-456"
		name := "Text Processing Plan"
		description := "Process user text with agents"
		priority := ExecutionPlanPriorityHigh

		// Planning intelligence data
		intent := "process_text_with_word_count"
		category := "text_processing"
		confidence := 95
		reasoning := "Text processing request with clear intent"
		availableAgents := []string{"text-processor-001", "analyzer-002"}
		requiredAgents := []string{"text-processor-001"}
		agentGap := []string{} // No gap
		planningType := PlanningTypeExecute

		// Create unified execution plan with both planning and execution data
		plan := NewUnifiedExecutionPlan(
			planID,
			name,
			description,
			priority,
			requestID,
			intent,
			category,
			confidence,
			reasoning,
			availableAgents,
			requiredAgents,
			agentGap,
			planningType,
		)

		// Validate execution metadata
		if plan.ID != planID {
			t.Errorf("Expected plan ID %s, got %s", planID, plan.ID)
		}
		if plan.Name != name {
			t.Errorf("Expected plan name %s, got %s", name, plan.Name)
		}
		if plan.Priority != priority {
			t.Errorf("Expected priority %s, got %s", priority, plan.Priority)
		}
		if plan.Status != ExecutionPlanStatusDraft {
			t.Errorf("Expected status %s, got %s", ExecutionPlanStatusDraft, plan.Status)
		}

		// Validate planning intelligence (these fields don't exist yet - will fail)
		if plan.RequestID != requestID {
			t.Errorf("Expected request ID %s, got %s", requestID, plan.RequestID)
		}
		if plan.Intent != intent {
			t.Errorf("Expected intent %s, got %s", intent, plan.Intent)
		}
		if plan.Category != category {
			t.Errorf("Expected category %s, got %s", category, plan.Category)
		}
		if plan.Confidence != confidence {
			t.Errorf("Expected confidence %d, got %d", confidence, plan.Confidence)
		}
		if plan.Reasoning != reasoning {
			t.Errorf("Expected reasoning %s, got %s", reasoning, plan.Reasoning)
		}
		if plan.Type != planningType {
			t.Errorf("Expected type %s, got %s", planningType, plan.Type)
		}

		// Validate agent arrays
		if len(plan.AvailableAgents) != len(availableAgents) {
			t.Errorf("Expected %d available agents, got %d", len(availableAgents), len(plan.AvailableAgents))
		}
		if len(plan.RequiredAgents) != len(requiredAgents) {
			t.Errorf("Expected %d required agents, got %d", len(requiredAgents), len(plan.RequiredAgents))
		}
		if len(plan.AgentGap) != len(agentGap) {
			t.Errorf("Expected %d agent gap, got %d", len(agentGap), len(plan.AgentGap))
		}
	})

	t.Run("RED: should transition from planning to execution phases", func(t *testing.T) {
		// Test planning completion and execution transition
		plan := NewUnifiedExecutionPlan(
			"plan-123",
			"Test Plan",
			"Test Description",
			ExecutionPlanPriorityMedium,
			"req-456",
			"test_intent",
			"test_category",
			90,
			"test reasoning",
			[]string{"agent1"},
			[]string{"agent1"},
			[]string{},
			PlanningTypeExecute,
		)

		// Initially should be in DRAFT status
		if plan.Status != ExecutionPlanStatusDraft {
			t.Errorf("Expected initial status %s, got %s", ExecutionPlanStatusDraft, plan.Status)
		}

		// Complete planning phase (this method doesn't exist yet - will fail)
		err := plan.CompletePlanning()
		if err != nil {
			t.Errorf("Failed to complete planning: %v", err)
		}

		// Planning completion timestamp should be set
		if plan.PlanningCompletedAt == nil {
			t.Error("Expected PlanningCompletedAt to be set after completing planning")
		}

		// Approve the plan
		err = plan.Approve()
		if err != nil {
			t.Errorf("Failed to approve plan: %v", err)
		}

		if plan.Status != ExecutionPlanStatusApproved {
			t.Errorf("Expected status %s after approval, got %s", ExecutionPlanStatusApproved, plan.Status)
		}

		if plan.ApprovedAt == nil {
			t.Error("Expected ApprovedAt to be set after approval")
		}
	})

	t.Run("RED: should eliminate need for separate PlanningResult entity", func(t *testing.T) {
		// This test validates that we no longer need PlanningResult
		// All planning data should be in ExecutionPlan

		plan := NewUnifiedExecutionPlan(
			"plan-123",
			"Unified Plan",
			"No more PlanningResult needed",
			ExecutionPlanPriorityHigh,
			"req-789",
			"unified_planning",
			"architecture_improvement",
			100,
			"Unified entity reduces complexity",
			[]string{"agent1", "agent2"},
			[]string{"agent1"},
			[]string{},
			PlanningTypeExecute,
		)

		// All data that was in PlanningResult should now be in ExecutionPlan
		if plan.RequestID == "" {
			t.Error("RequestID should be available in unified plan")
		}
		if plan.Intent == "" {
			t.Error("Intent should be available in unified plan")
		}
		if plan.Category == "" {
			t.Error("Category should be available in unified plan")
		}
		if plan.Confidence == 0 {
			t.Error("Confidence should be available in unified plan")
		}
		if plan.Reasoning == "" {
			t.Error("Reasoning should be available in unified plan")
		}
		if len(plan.AvailableAgents) == 0 {
			t.Error("AvailableAgents should be available in unified plan")
		}
		if len(plan.RequiredAgents) == 0 {
			t.Error("RequiredAgents should be available in unified plan")
		}

		// Method to get complete plan data without joins (doesn't exist yet - will fail)
		planData := plan.GetCompletePlanData()
		if planData == nil {
			t.Error("Should be able to get complete plan data from unified entity")
		}
	})
}

// TestExecutionPlanConsolidation_BreakingChanges validates breaking changes are acceptable
func TestExecutionPlanConsolidation_BreakingChanges(t *testing.T) {
	t.Run("RED: PlanningResult entity should be deprecated", func(t *testing.T) {
		// This test documents that PlanningResult will be removed
		// No backward compatibility - breaking changes are acceptable

		// After consolidation, this should fail to compile:
		// planningResult := &PlanningResult{} // This will be removed

		// Instead, everything should be in ExecutionPlan
		plan := &ExecutionPlan{
			ID:          "plan-123",
			Name:        "Consolidated Plan",
			Description: "All data in one place",
			// Planning fields will be added here
		}

		// Validate we can work with single entity
		if plan.ID == "" {
			t.Error("ExecutionPlan should contain all necessary data")
		}
	})
}
