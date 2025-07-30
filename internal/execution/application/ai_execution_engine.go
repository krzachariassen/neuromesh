package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	aiDomain "neuromesh/internal/ai/domain"
	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
	planningDomain "neuromesh/internal/planning/domain"
)

const (
	EventPrefix         = "SEND_EVENT:"
	UserResponsePrefix  = "USER_RESPONSE:"
	DefaultEventTimeout = 30 * time.Second
)

// AgentInstruction represents the JSON structure for AI-to-agent instructions
type AgentInstruction struct {
	AgentID string `json:"agent_id"`
	Action  string `json:"action"`
	Content string `json:"content"`
	Intent  string `json:"intent"`
}

// AIExecutionEngine handles AI-native execution with agent coordination
type AIExecutionEngine struct {
	aiProvider         aiDomain.AIProvider
	aiMessageBus       messaging.AIMessageBus
	correlationTracker *infrastructure.CorrelationTracker
	repository         planningDomain.ExecutionPlanRepository
	messageBus         messaging.MessageBus // Use clean MessageBus interface for domain events
	recentAssignments  map[string][]string  // Track recent agent assignments to prevent loops
}

// NewAIExecutionEngine creates a new AI execution engine with clean messaging abstraction
func NewAIExecutionEngine(
	aiProvider aiDomain.AIProvider,
	aiMessageBus messaging.AIMessageBus,
	correlationTracker *infrastructure.CorrelationTracker,
	repository planningDomain.ExecutionPlanRepository,
	messageBus messaging.MessageBus,
) *AIExecutionEngine {
	return &AIExecutionEngine{
		aiProvider:         aiProvider,
		aiMessageBus:       aiMessageBus,
		correlationTracker: correlationTracker,
		repository:         repository,
		messageBus:         messageBus,
		recentAssignments:  make(map[string][]string),
	}
}

// ExecuteWithAgents handles plan-driven execution by following the predetermined execution plan
// This is stateless and supports concurrent executions using execution step IDs as correlation IDs
func (e *AIExecutionEngine) ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext, planID string) (string, error) {
	log.Printf("[DEBUG] 📋 Starting plan-driven execution for planID=%s", planID)

	// Update execution plan status to EXECUTING when execution starts
	plan, err := e.repository.GetByID(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("failed to get execution plan for status update: %w", err)
	}

	// Update status to EXECUTING if it's in DRAFT state
	if plan.Status == planningDomain.ExecutionPlanStatusDraft {
		plan.Status = planningDomain.ExecutionPlanStatusExecuting
		startTime := time.Now()
		plan.StartedAt = &startTime
		err = e.repository.Update(ctx, plan)
		if err != nil {
			log.Printf("Warning: Failed to update execution plan status to EXECUTING for plan %s: %v", planID, err)
		}
	}

	// PLAN-DRIVEN EXECUTION: Execute steps according to the predetermined plan
	return e.executePlanSteps(ctx, plan, userInput, userID, agentContext)
}

// executePlanSteps executes the plan steps in sequence following the predetermined execution plan
func (e *AIExecutionEngine) executePlanSteps(ctx context.Context, plan *planningDomain.ExecutionPlan, userInput, userID, agentContext string) (string, error) {
	log.Printf("[DEBUG] 📋 Executing plan with %d steps", len(plan.Steps))

	// Use the steps that are already in the plan - no need for additional repository call
	steps := plan.Steps
	if len(steps) == 0 {
		return "", fmt.Errorf("no steps found in execution plan %s", plan.ID)
	}

	var results []string

	// Execute steps in sequence (for now - later we can add parallel execution)
	for _, step := range steps {
		log.Printf("[DEBUG] 🚀 Executing step %s: %s (agent: %s)", step.ID, step.Description, step.AssignedAgent)

		// Execute the step - trust the predetermined plan's agent assignments
		result, err := e.executeStep(ctx, step, userInput, userID, agentContext, plan.ID)
		if err != nil {
			log.Printf("[ERROR] Step %s execution failed: %v", step.ID, err)
			return "", fmt.Errorf("failed to execute step %s: %w", step.ID, err)
		}

		results = append(results, result)
		log.Printf("[DEBUG] ✅ Step %s completed: %s", step.ID, result)
	}

	// All steps completed - the EDA system (ai_result_synthesizer.go) will handle synthesis
	// via the agent completion events we published for each step
	log.Printf("[DEBUG] 📋 Plan-driven execution completed for planID=%s - synthesis handled by EDA", plan.ID)

	// For now, return a simple completion message
	// The actual synthesis will be handled asynchronously by the EDA system
	return fmt.Sprintf("Execution plan %s completed successfully with %d steps", plan.ID, len(results)), nil
}

// executeStep executes a single execution step by sending it to the appropriate agent
func (e *AIExecutionEngine) executeStep(ctx context.Context, step *planningDomain.ExecutionStep, userInput, userID, agentContext, planID string) (string, error) {
	log.Printf("[DEBUG] 📨 Sending step to agent %s: %s", step.AssignedAgent, step.Description)

	// Mark step as started
	if err := step.Start(); err != nil {
		return "", fmt.Errorf("failed to start step %s: %w", step.ID, err)
	}
	if err := e.repository.UpdateStep(ctx, step); err != nil {
		log.Printf("Warning: Failed to update step status to executing: %v", err)
	}

	// Create agent instruction based on the step
	eventMsg := &messaging.AIToAgentMessage{
		AgentID:       step.AssignedAgent,
		Content:       step.Description, // Use the step description as agent instruction
		Intent:        step.Name,        // Use the step name as intent
		CorrelationID: step.ID,          // Use step ID as correlation ID
		Context: map[string]interface{}{
			"original_request": userInput,
			"user_id":          userID,
			"task":             step.Name,
			"execution_mode":   true,
			"plan_id":          planID,
		},
		Timeout: DefaultEventTimeout,
	}

	// Send event to agent via message bus
	err := e.aiMessageBus.SendToAgent(ctx, eventMsg)
	if err != nil {
		return "", fmt.Errorf("failed to send step to agent %s: %w", step.AssignedAgent, err)
	}

	// Wait for agent response
	agentResponse, err := e.waitForAgentResponse(ctx, step.ID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to receive agent response for step %s: %w", step.ID, err)
	}

	// Process the agent response
	result, err := e.processAgentExecutionResponse(ctx, agentResponse, step)
	if err != nil {
		return "", fmt.Errorf("failed to process agent response for step %s: %w", step.ID, err)
	}

	// Store agent result
	if e.repository != nil {
		err = e.storeAgentResult(ctx, agentResponse)
		if err != nil {
			log.Printf("Warning: Failed to store agent result: %v", err)
		}
	}

	// Mark step as completed
	if err := step.Complete(result); err != nil {
		log.Printf("Warning: Failed to complete step: %v", err)
	} else {
		if err := e.repository.UpdateStep(ctx, step); err != nil {
			log.Printf("Warning: Failed to update step status to completed: %v", err)
		}
	}

	// Publish completion event
	e.publishAgentCompletionEvent(ctx, planID, step.ID, step.AssignedAgent)

	return result, nil
}

// processAgentExecutionResponse processes the agent's response for plan-driven execution
func (e *AIExecutionEngine) processAgentExecutionResponse(ctx context.Context, agentResponse *messaging.AgentToAIMessage, step *planningDomain.ExecutionStep) (string, error) {
	log.Printf("[DEBUG] Processing agent response for step %s: %s", step.ID, agentResponse.Content)

	// Check for agent errors
	if agentResponse.Context != nil {
		if success, ok := agentResponse.Context["success"].(bool); ok && !success {
			errorMsg := "Agent execution failed"
			if errMsg, ok := agentResponse.Context["error"].(string); ok {
				errorMsg = errMsg
			}
			return "", fmt.Errorf("agent %s failed to execute step %s: %s", step.AssignedAgent, step.Name, errorMsg)
		}
	}

	// For plan-driven execution, we trust the agent response and return it as-is
	// The agent should provide properly formatted output based on the step description
	result := strings.TrimSpace(agentResponse.Content)

	if result == "" {
		return "", fmt.Errorf("agent %s returned empty response for step %s", step.AssignedAgent, step.Name)
	}

	log.Printf("[DEBUG] ✅ Agent response processed successfully for step %s", step.ID)
	return result, nil
}

// waitForAgentResponse waits for an agent response using correlation tracking
func (e *AIExecutionEngine) waitForAgentResponse(ctx context.Context, correlationID, userID string) (*messaging.AgentToAIMessage, error) {
	// Register request with correlation tracker
	timeout := 30 * time.Second
	responseChan := e.correlationTracker.RegisterRequest(correlationID, userID, timeout)

	// Wait for response or timeout
	select {
	case response := <-responseChan:
		if response != nil {
			return response, nil
		}
		return nil, fmt.Errorf("received nil execution response for correlation %s", correlationID)
	case <-ctx.Done():
		e.correlationTracker.CleanupRequest(correlationID)
		return nil, ctx.Err()
	case <-time.After(timeout):
		e.correlationTracker.CleanupRequest(correlationID)
		return nil, fmt.Errorf("timeout waiting for agent execution response (correlation: %s)", correlationID)
	}
}

// storeAgentResult stores an agent result in the repository
func (e *AIExecutionEngine) storeAgentResult(ctx context.Context, agentResponse *messaging.AgentToAIMessage) error {
	if e.repository == nil {
		return nil // No repository configured, skip storage
	}

	// Create metadata with context information
	metadata := map[string]interface{}{
		"correlation_id": agentResponse.CorrelationID,
	}

	// Copy context fields to metadata
	if agentResponse.Context != nil {
		for key, value := range agentResponse.Context {
			metadata[key] = value
		}
	}

	// Determine result status based on agent response
	status := executionDomain.AgentResultStatusSuccess
	if agentResponse.Context != nil {
		if success, ok := agentResponse.Context["success"].(bool); ok && !success {
			status = executionDomain.AgentResultStatusFailed
		}
	}

	// Create agent result using the correlation ID as execution step ID
	agentResult := executionDomain.NewAgentResultWithStatus(
		agentResponse.CorrelationID, // Use correlation ID as execution step ID
		agentResponse.AgentID,
		agentResponse.Content,
		metadata,
		status,
	)

	// Store the agent result
	err := e.repository.StoreAgentResult(ctx, agentResult)
	if err != nil {
		return fmt.Errorf("failed to store agent result: %w", err)
	}

	log.Printf("[DEBUG] ✅ Stored agent result - stepID=%s agentID=%s status=%s",
		agentResponse.CorrelationID, agentResponse.AgentID, status)

	return nil
}

// publishAgentCompletionEvent publishes an agent completion event
func (e *AIExecutionEngine) publishAgentCompletionEvent(ctx context.Context, planID, stepID, agentID string) {
	// Use clean domain event interface
	if e.messageBus != nil && planID != "" {
		err := PublishAgentCompletedEvent(ctx, e.messageBus, planID, stepID, agentID)
		if err != nil {
			log.Printf("Warning: Failed to publish agent completion event: %v", err)
		} else {
			log.Printf("✅ Published agent completion event - planID=%s stepID=%s agentID=%s", planID, stepID, agentID)
		}
	}
}
