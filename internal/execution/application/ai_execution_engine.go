package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	aiDomain "neuromesh/internal/ai/domain"
	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
	planningDomain "neuromesh/internal/planning/domain"

	"github.com/google/uuid"
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
	}
}

// ExecuteWithAgents handles AI-native execution with bidirectional agent communication via events
// This is stateless and supports concurrent executions using execution step IDs as correlation IDs
func (e *AIExecutionEngine) ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext, planID string) (string, error) {
	// Get AI execution decision using improved system prompt
	systemPrompt := e.buildExecutionSystemPrompt(agentContext, executionPlan)
	userPrompt := fmt.Sprintf("Execute plan for user request: %s", userInput)

	// Get AI execution decision - use background context to prevent cancellation during async execution
	response, err := e.aiProvider.CallAI(context.Background(), systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI execution call failed: %w", err)
	}

	// Check if AI wants to send event to an agent
	if strings.Contains(response, EventPrefix) {
		return e.handleAgentEvent(ctx, response, userInput, userID, agentContext, planID)
	}

	// Extract direct user response
	if strings.Contains(response, UserResponsePrefix) {
		return e.extractUserResponse(response), nil
	}

	// Fallback - return AI response as-is
	return response, nil
}

// buildExecutionSystemPrompt creates the system prompt for AI execution
func (e *AIExecutionEngine) buildExecutionSystemPrompt(agentContext, executionPlan string) string {
	return fmt.Sprintf(`You are an AI execution engine that coordinates with multiple agents to execute plans.

EXECUTION PLAN:
%s

AVAILABLE AGENTS (JSON format):
%s

Your role is to EXECUTE the plan by coordinating with agents through events. You can:
1. Send events to agents to perform specific tasks
2. Process agent responses and coordinate next steps
3. Provide final results to users

The available agents are provided in JSON format. Extract the exact "id" field from the JSON for each agent.

When you need an agent to perform work, respond with EXACTLY this format:
%s
{
  "agent_id": "[use exact agent 'id' from the JSON above, like 'text-processor-001']",
  "action": "[specific action like 'word-count', 'text-analysis', 'character-count']",
  "content": "[specific instructions for the agent]",
  "intent": "[high-level goal like 'word_counting', 'text_analysis']"
}

IMPORTANT: Return valid JSON. Parse the available agents JSON and use the exact "id" field. For text processing, use "text-processor-001".

When providing final response to user, respond with:
%s
[Your response to the user]

Always use the execution plan as your guide and coordinate agents efficiently. Parse the JSON to get exact agent IDs.`, executionPlan, agentContext, EventPrefix, UserResponsePrefix)
}

// handleAgentEvent processes AI's decision to send event to an agent during execution
func (e *AIExecutionEngine) handleAgentEvent(ctx context.Context, aiResponse, originalRequest, userID, agentContext, planID string) (string, error) {
	// Parse AI's agent event instruction from JSON
	log.Printf("[DEBUG] AI Response for Agent Event:\n%s", aiResponse)

	// Extract JSON from the response (after the SEND_EVENT: prefix)
	jsonStart := strings.Index(aiResponse, "{")
	jsonEnd := strings.LastIndex(aiResponse, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return "", fmt.Errorf("no valid JSON found in AI response")
	}

	jsonStr := aiResponse[jsonStart : jsonEnd+1]
	log.Printf("[DEBUG] Extracted JSON: %s", jsonStr)

	var instruction AgentInstruction
	if err := json.Unmarshal([]byte(jsonStr), &instruction); err != nil {
		return "", fmt.Errorf("failed to parse agent instruction JSON: %w", err)
	}

	log.Printf("[DEBUG] Parsed instruction - Agent: '%s', Action: '%s', Content: '%s', Intent: '%s'",
		instruction.AgentID, instruction.Action, instruction.Content, instruction.Intent)

	// Find existing execution step from plan that matches this agent task
	stepID := ""
	if e.repository != nil && planID != "" {
		// Get existing steps for the plan
		steps, err := e.repository.GetStepsByPlanID(ctx, planID)
		if err != nil {
			log.Printf("Warning: Failed to get steps for plan %s: %v", planID, err)
		} else {
			// Find step that matches the agent and action
			for _, step := range steps {
				if step.AssignedAgent == instruction.AgentID {
					// Use the existing step ID from the plan
					stepID = step.ID
					log.Printf("[DEBUG] Using existing execution step - stepID=%s planID=%s agentID=%s", stepID, planID, instruction.AgentID)
					break
				}
			}
		}
	}

	// Fallback: create new step ID if no existing step found (shouldn't happen in normal flow)
	if stepID == "" {
		stepID = uuid.New().String()
		log.Printf("[DEBUG] Warning: No existing step found, using new stepID=%s for agentID=%s", stepID, instruction.AgentID)
	}

	// Create AI-to-Agent event message with step ID as correlation ID
	eventMsg := &messaging.AIToAgentMessage{
		AgentID:       instruction.AgentID,
		Content:       instruction.Content,
		Intent:        instruction.Intent,
		CorrelationID: stepID, // Use execution step ID as correlation ID
		Context: map[string]interface{}{
			"original_request": originalRequest,
			"user_id":          userID,
			"action":           instruction.Action,
			"execution_mode":   true,
			// Note: No plan_id sent to agent - graph lookup will find it via step_id
		},
		Timeout: DefaultEventTimeout,
	}

	// Note: No need to store correlation-to-plan mapping since stepID inherently belongs to planID
	// The graph repository can find the plan via step->plan relationship

	// Send event to agent via message bus
	err := e.aiMessageBus.SendToAgent(ctx, eventMsg)
	if err != nil {
		return "", fmt.Errorf("failed to send execution event to agent %s: %w", instruction.AgentID, err)
	}

	// Wait for agent response using correlation tracker directly (no redundant queue subscription)
	agentResponse, err := e.waitForAgentResponse(ctx, stepID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to receive agent execution response: %w", err)
	}

	// Let AI process the agent response during execution
	return e.processAgentExecutionResponse(ctx, agentResponse, originalRequest, userID, agentContext)
}

// waitForAgentResponse waits for an agent response using correlation tracking
// This relies on GlobalMessageConsumer to route agent responses via CorrelationTracker
func (e *AIExecutionEngine) waitForAgentResponse(ctx context.Context, correlationID, userID string) (*messaging.AgentToAIMessage, error) {
	// Register request with correlation tracker - no need for redundant queue subscription
	timeout := 30 * time.Second
	responseChan := e.correlationTracker.RegisterRequest(correlationID, userID, timeout)

	// Wait for response or timeout - GlobalMessageConsumer handles routing
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

// processAgentExecutionResponse lets AI decide what to do with agent response during execution
func (e *AIExecutionEngine) processAgentExecutionResponse(ctx context.Context, agentResponse *messaging.AgentToAIMessage, originalRequest, userID, agentContext string) (string, error) {
	// Store agent result if repository is available
	if e.repository != nil {
		err := e.storeAgentResult(ctx, agentResponse)
		if err != nil {
			// Log error but don't fail execution - storage is supplementary
			// In production, this would be logged properly
		}
	}

	systemPrompt := fmt.Sprintf(`You are an AI execution engine processing an agent response during plan execution.

Original user request: %s
Agent ID: %s
Agent response: %s
Agent context: %v

Based on the agent execution response, decide:
1. Do you need to coordinate with another agent to continue execution?
2. Do you need to ask the agent for clarification via event?
3. Can you provide final execution result to user?

If coordinating with another agent, respond with:
%s
{
  "agent_id": "[agent-id]",
  "action": "[specific action]", 
  "content": "[specific instructions for the agent]",
  "intent": "[high-level goal]"
}

If providing final result to user, respond with:
%s
[Your execution result for the user]`, originalRequest, agentResponse.AgentID, agentResponse.Content, agentContext, EventPrefix, UserResponsePrefix)

	userPrompt := "Process the agent response and determine next execution step."

	response, err := e.aiProvider.CallAI(context.Background(), systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI execution processing failed: %w", err)
	}

	// Check if AI wants to coordinate with another agent
	if strings.Contains(response, EventPrefix) {
		// Extract planID from agent response context
		planID := ""
		if agentResponse.Context != nil {
			if planIDValue, ok := agentResponse.Context["plan_id"].(string); ok {
				planID = planIDValue
			}
		}

		return e.handleAgentEvent(ctx, response, originalRequest, userID, agentContext, planID)
	}

	// Extract user response
	var finalResult string
	if strings.Contains(response, UserResponsePrefix) {
		finalResult = e.extractUserResponse(response)
	} else {
		finalResult = response
	}

	// Publish agent completion event AFTER full processing is complete
	// This fixes the race condition by ensuring synthesis happens after execution finishes
	log.Printf("[DEBUG] Using graph lookup to find plan_id from step_id: %s", agentResponse.CorrelationID)
	
	// Use graph-native lookup to find plan_id from step_id (correlation_id)
	planID := e.extractPlanIDFromContext(nil, agentResponse.CorrelationID)
	if planID != "" {
		log.Printf("[DEBUG] Publishing agent completion event - planID=%s stepID=%s agentID=%s", planID, agentResponse.CorrelationID, agentResponse.AgentID)
		e.publishAgentCompletionEvent(ctx, planID, agentResponse.CorrelationID, agentResponse.AgentID)
	} else {
		log.Printf("[DEBUG] Could not find plan_id via graph lookup for step_id: %s", agentResponse.CorrelationID)
	}

	return finalResult, nil
}

// extractUserResponse extracts the user response from AI output
func (e *AIExecutionEngine) extractUserResponse(response string) string {
	lines := strings.Split(response, "\n")
	var userResponse []string
	foundPrefix := false

	for _, line := range lines {
		if strings.Contains(line, UserResponsePrefix) {
			foundPrefix = true
			// Extract content after the prefix on the same line
			if afterPrefix := strings.TrimSpace(strings.TrimPrefix(line, UserResponsePrefix)); afterPrefix != "" {
				userResponse = append(userResponse, afterPrefix)
			}
			continue
		}
		if foundPrefix {
			userResponse = append(userResponse, line)
		}
	}

	return strings.TrimSpace(strings.Join(userResponse, "\n"))
}

// storeAgentResult stores an agent result in the repository for graph-native synthesis
func (e *AIExecutionEngine) storeAgentResult(ctx context.Context, agentResponse *messaging.AgentToAIMessage) error {
	if e.repository == nil {
		return nil // No repository configured, skip storage
	}

	// Extract step ID from correlation ID - use correlation ID directly for graph linkage
	stepID := agentResponse.CorrelationID

	// Graph-native approach: Find planID using graph traversal
	planID := e.extractPlanIDFromContext(agentResponse.Context, stepID)

	// Check if we can find an actual execution step with this correlation ID
	canLinkToStep := false
	if planID != "" {
		steps, err := e.repository.GetStepsByPlanID(ctx, planID)
		if err == nil {
			// Check if any step matches this correlation ID
			for _, step := range steps {
				if step.ID == stepID {
					canLinkToStep = true
					break
				}
			}
		}
	}

	// Determine result status based on agent response
	status := executionDomain.AgentResultStatusSuccess
	if agentResponse.Context != nil {
		if success, ok := agentResponse.Context["success"].(bool); ok && !success {
			status = executionDomain.AgentResultStatusFailed
		}
	}

	// Create agent result with correlation ID as step ID for graph linkage
	// Create metadata map with structured fields
	metadata := map[string]interface{}{
		"correlation_id":   agentResponse.CorrelationID,
		"original_context": agentResponse.Context,
		"plan_id":          planID, // Store planID in metadata for synthesis
	}

	// Copy context fields to metadata for backwards compatibility
	if agentResponse.Context != nil {
		for key, value := range agentResponse.Context {
			metadata[key] = value
		}
	}

	agentResult := executionDomain.NewAgentResultWithStatus(
		agentResponse.CorrelationID, // Use correlation ID directly as step ID
		agentResponse.AgentID,
		agentResponse.Content,
		metadata,
		status,
	)

	// Store in repository - the graph will maintain the relationship
	err := e.repository.StoreAgentResult(ctx, agentResult)
	if err != nil {
		return fmt.Errorf("failed to store agent result: %w", err)
	}

	// Only try to mark step as completed if we can link to an actual step
	if canLinkToStep && status == executionDomain.AgentResultStatusSuccess {
		err = e.markStepAsCompleted(ctx, stepID)
		if err != nil {
			// Log error but don't fail - step completion is important but not critical
			log.Printf("Warning: Failed to mark step %s as completed: %v", stepID, err)
		}
	}

	return nil
}

// markStepAsCompleted marks an execution step as completed
func (e *AIExecutionEngine) markStepAsCompleted(ctx context.Context, stepID string) error {
	// Use graph-native approach to find the plan ID
	planID := e.extractPlanIDFromContext(nil, stepID)
	if planID == "" {
		// If we can't extract planID, we can't efficiently find the step
		return fmt.Errorf("unable to extract plan ID from step ID %s", stepID)
	}

	// Get steps for the plan
	steps, err := e.repository.GetStepsByPlanID(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to get steps for plan %s: %w", planID, err)
	}

	// Find the specific step
	var targetStep *planningDomain.ExecutionStep
	for _, step := range steps {
		if step.ID == stepID {
			targetStep = step
			break
		}
	}

	if targetStep == nil {
		return fmt.Errorf("step %s not found in plan %s", stepID, planID)
	}

	// Handle step status progression based on current status
	switch targetStep.Status {
	case planningDomain.ExecutionStepStatusPending:
		// Step hasn't been started yet, mark as assigned first
		targetStep.Assign()
		// Start the step
		if err := targetStep.Start(); err != nil {
			return fmt.Errorf("failed to start step %s: %w", stepID, err)
		}
		// Complete the step with agent result content
		if err := targetStep.Complete("Agent execution completed"); err != nil {
			return fmt.Errorf("failed to complete step %s: %w", stepID, err)
		}
	case planningDomain.ExecutionStepStatusAssigned:
		// Step is assigned but not started, start it first
		if err := targetStep.Start(); err != nil {
			return fmt.Errorf("failed to start step %s: %w", stepID, err)
		}
		// Complete the step
		if err := targetStep.Complete("Agent execution completed"); err != nil {
			return fmt.Errorf("failed to complete step %s: %w", stepID, err)
		}
	case planningDomain.ExecutionStepStatusExecuting:
		// Step is already executing, just complete it
		if err := targetStep.Complete("Agent execution completed"); err != nil {
			return fmt.Errorf("failed to complete step %s: %w", stepID, err)
		}
	case planningDomain.ExecutionStepStatusCompleted:
		// Step is already completed, nothing to do
		return nil
	default:
		return fmt.Errorf("cannot complete step %s with status %s", stepID, targetStep.Status)
	}

	// Update in repository
	err = e.repository.UpdateStep(ctx, targetStep)
	if err != nil {
		return fmt.Errorf("failed to update step %s status: %w", stepID, err)
	}

	return nil
}

// extractPlanIDFromContext extracts plan ID using graph-native context lookup
func (e *AIExecutionEngine) extractPlanIDFromContext(context map[string]interface{}, stepID string) string {
	// First, try to get planID from context (for immediate context preservation)
	if context != nil {
		if planID, ok := context["plan_id"].(string); ok && planID != "" {
			return planID
		}
	}

	// Graph-native approach: Since stepID is now a real execution step ID,
	// we can query the step directly to get its plan ID
	if e.repository != nil {
		// Try to find execution steps by querying plans and their steps
		// This is a simplified approach - in a full implementation we'd have GetStepByID
		if planID := e.queryStepForPlanID(stepID); planID != "" {
			log.Printf("[DEBUG] Found planID from execution step - stepID=%s planID=%s", stepID, planID)
			return planID
		}
	}

	// Fallback: extract from stepID pattern (legacy approach)
	return e.extractPlanIDFromStepID(stepID)
}

// queryStepForPlanID queries execution steps to find the plan ID for a given step ID
func (e *AIExecutionEngine) queryStepForPlanID(stepID string) string {
	// Use the repository's GetPlanIDByCorrelationID method for graph-native lookup
	// The stepID is actually the execution step ID (which is the correlation ID in our design)
	planID, err := e.repository.GetPlanIDByCorrelationID(context.Background(), stepID)
	if err == nil && planID != "" {
		log.Printf("[DEBUG] Found planID via execution step lookup - stepID=%s planID=%s", stepID, planID)
		return planID
	}

	log.Printf("[DEBUG] Could not find planID for stepID %s: %v", stepID, err)
	return ""
}

// extractPlanIDFromStepID extracts plan ID from step ID
// This is a temporary solution - in a real system, planID should be passed explicitly
func (e *AIExecutionEngine) extractPlanIDFromStepID(stepID string) string {
	// For now, assume stepID format like "plan-123-step-1" or similar
	// This is a heuristic approach - in production, planID should be explicit
	parts := strings.Split(stepID, "-")
	if len(parts) >= 2 {
		// Try to find "plan-{id}" pattern
		for i := 0; i < len(parts)-1; i++ {
			if parts[i] == "plan" {
				return fmt.Sprintf("plan-%s", parts[i+1])
			}
		}
	}
	// Fallback: return empty string if pattern not recognized
	return ""
}

// publishAgentCompletionEvent publishes an agent completion event using clean domain events
// This fixes the race condition by publishing AFTER agent result storage
func (e *AIExecutionEngine) publishAgentCompletionEvent(ctx context.Context, planID, stepID, agentID string) {
	// Use clean domain event interface - no infrastructure exposure
	if e.messageBus != nil && planID != "" {
		err := PublishAgentCompletedEvent(ctx, e.messageBus, planID, stepID, agentID)
		if err != nil {
			log.Printf("Warning: Failed to publish agent completion event: %v", err)
		} else {
			log.Printf("✅ Published agent completion event - planID=%s stepID=%s agentID=%s", planID, stepID, agentID)
		}
	}
}
