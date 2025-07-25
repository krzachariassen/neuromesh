package application

import (
	"context"
	"fmt"
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

// AIExecutionEngine handles AI-native execution with agent coordination
type AIExecutionEngine struct {
	aiProvider         aiDomain.AIProvider
	aiMessageBus       messaging.AIMessageBus
	correlationTracker *infrastructure.CorrelationTracker
	repository         planningDomain.ExecutionPlanRepository
}

// NewAIExecutionEngine creates a new AI execution engine with repository for result storage
func NewAIExecutionEngine(aiProvider aiDomain.AIProvider, aiMessageBus messaging.AIMessageBus, correlationTracker *infrastructure.CorrelationTracker, repository planningDomain.ExecutionPlanRepository) *AIExecutionEngine {
	return &AIExecutionEngine{
		aiProvider:         aiProvider,
		aiMessageBus:       aiMessageBus,
		correlationTracker: correlationTracker,
		repository:         repository,
	}
}

// ExecuteWithAgents handles AI-native execution with bidirectional agent communication via events
// This is stateless and supports concurrent executions using correlation IDs
func (e *AIExecutionEngine) ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext string) (string, error) {
	// Generate unique correlation ID for this execution
	correlationID := fmt.Sprintf("exec-%s-%s", userID, uuid.New().String())

	// Get AI execution decision using improved system prompt
	systemPrompt := e.buildExecutionSystemPrompt(agentContext, executionPlan)
	userPrompt := fmt.Sprintf("Execute plan for user request: %s", userInput)

	// Get AI execution decision
	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI execution call failed: %w", err)
	}

	// Check if AI wants to send event to an agent
	if strings.Contains(response, EventPrefix) {
		return e.handleAgentEvent(ctx, response, userInput, userID, agentContext, correlationID)
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

AVAILABLE AGENTS:
%s

Your role is to EXECUTE the plan by coordinating with agents through events. You can:
1. Send events to agents to perform specific tasks
2. Process agent responses and coordinate next steps
3. Provide final results to users

When you need an agent to perform work, respond with:
%s
Agent: [agent-id from context]
Action: [specific action like "deploy", "analyze", "monitor"]
Content: [specific instructions for the agent]
Intent: [high-level goal like "deployment", "analysis"]

When providing final response to user, respond with:
%s
[Your response to the user]

Always use the execution plan as your guide and coordinate agents efficiently.`, executionPlan, agentContext, EventPrefix, UserResponsePrefix)
}

// handleAgentEvent processes AI's decision to send event to an agent during execution
func (e *AIExecutionEngine) handleAgentEvent(ctx context.Context, aiResponse, originalRequest, userID, agentContext, correlationID string) (string, error) {
	// Parse AI's agent event instruction
	agentID := e.extractSection(aiResponse, "Agent:")
	action := e.extractSection(aiResponse, "Action:")
	content := e.extractSection(aiResponse, "Content:")
	intent := e.extractSection(aiResponse, "Intent:")

	// Create AI-to-Agent event message with correlation ID
	eventMsg := &messaging.AIToAgentMessage{
		AgentID:       agentID,
		Content:       content,
		Intent:        intent,
		CorrelationID: correlationID,
		Context: map[string]interface{}{
			"original_request": originalRequest,
			"user_id":          userID,
			"action":           action,
			"execution_mode":   true,
		},
		Timeout: DefaultEventTimeout,
	}

	// Send event to agent via message bus
	err := e.aiMessageBus.SendToAgent(ctx, eventMsg)
	if err != nil {
		return "", fmt.Errorf("failed to send execution event to agent %s: %w", agentID, err)
	}

	// Wait for agent response using correlation tracker (stateless)
	agentResponse, err := e.waitForAgentResponseWithCorrelation(ctx, correlationID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to receive agent execution response: %w", err)
	}

	// Let AI process the agent response during execution
	return e.processAgentExecutionResponse(ctx, agentResponse, originalRequest, userID, agentContext)
}

// waitForAgentResponseWithCorrelation waits for an agent response using correlation tracking
func (e *AIExecutionEngine) waitForAgentResponseWithCorrelation(ctx context.Context, correlationID, userID string) (*messaging.AgentToAIMessage, error) {
	// Register request with correlation tracker
	timeout := 30 * time.Second
	responseChan := e.correlationTracker.RegisterRequest(correlationID, userID, timeout)

	// Subscribe to the execution response channel
	responseChannel, err := e.aiMessageBus.Subscribe(ctx, "ai-execution")
	if err != nil {
		e.correlationTracker.CleanupRequest(correlationID)
		return nil, fmt.Errorf("failed to subscribe for execution agent responses: %w", err)
	}

	// Start listening for agent responses and route them through correlation tracker
	go func() {
		defer func() {
			e.correlationTracker.CleanupRequest(correlationID)
		}()

		for {
			select {
			case msg, ok := <-responseChannel:
				if !ok {
					return
				}
				if msg != nil {
					if msg.MessageType == messaging.MessageTypeAgentToAI && msg.CorrelationID == correlationID {
						agentMsg := &messaging.AgentToAIMessage{
							AgentID:       msg.FromID,
							Content:       msg.Content,
							CorrelationID: msg.CorrelationID,
							MessageType:   msg.MessageType,
						}

						e.correlationTracker.RouteResponse(agentMsg)
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

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
Agent: [agent-id]
Action: [specific action]
Content: [specific instructions for the agent]
Intent: [high-level goal]

If providing final result to user, respond with:
%s
[Your execution result for the user]`, originalRequest, agentResponse.AgentID, agentResponse.Content, agentContext, EventPrefix, UserResponsePrefix)

	userPrompt := "Process the agent response and determine next execution step."

	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI execution processing failed: %w", err)
	}

	// Check if AI wants to coordinate with another agent
	if strings.Contains(response, EventPrefix) {
		correlationID := fmt.Sprintf("exec-%s-%s", userID, uuid.New().String())
		return e.handleAgentEvent(ctx, response, originalRequest, userID, agentContext, correlationID)
	}

	// Extract user response
	if strings.Contains(response, UserResponsePrefix) {
		return e.extractUserResponse(response), nil
	}

	return response, nil
}

// extractSection extracts a section from AI response
func (e *AIExecutionEngine) extractSection(response, section string) string {
	lines := strings.Split(response, "\n")
	for i, line := range lines {
		if strings.Contains(line, section) {
			if i+1 < len(lines) {
				return strings.TrimSpace(lines[i+1])
			}
		}
	}
	return ""
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

	// Extract step ID from correlation ID (format: step-1, step-2, etc.)
	stepID := agentResponse.CorrelationID

	// Determine result status based on agent response
	status := executionDomain.AgentResultStatusSuccess
	if agentResponse.Context != nil {
		if success, ok := agentResponse.Context["success"].(bool); ok && !success {
			status = executionDomain.AgentResultStatusFailed
		}
	}

	// Create agent result
	agentResult := executionDomain.NewAgentResultWithStatus(
		stepID, // ExecutionStepID
		agentResponse.AgentID,
		agentResponse.Content,
		agentResponse.Context, // Metadata
		status,
	)

	// Store in repository
	err := e.repository.StoreAgentResult(ctx, agentResult)
	if err != nil {
		return fmt.Errorf("failed to store agent result: %w", err)
	}

	return nil
}
