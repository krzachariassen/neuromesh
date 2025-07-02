package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	aiDomain "neuromesh/internal/ai/domain"
	"neuromesh/internal/messaging"
	"neuromesh/internal/orchestrator/infrastructure"
)

const (
	// AI prompting constants
	EventPrefix        = "SEND_EVENT:"
	UserResponsePrefix = "USER_RESPONSE:"

	// Event timeout
	DefaultEventTimeout = 30 * time.Second
)

// AIConversationEngine is a stateless, correlation-driven conversation engine
// that supports concurrent conversations using correlation IDs for message routing
type AIConversationEngine struct {
	aiProvider         aiDomain.AIProvider
	aiMessageBus       messaging.AIMessageBus
	correlationTracker *infrastructure.CorrelationTracker
}

// NewAIConversationEngine creates a new stateless AI conversation engine
func NewAIConversationEngine(
	aiProvider aiDomain.AIProvider,
	aiMessageBus messaging.AIMessageBus,
	correlationTracker *infrastructure.CorrelationTracker,
) *AIConversationEngine {
	return &AIConversationEngine{
		aiProvider:         aiProvider,
		aiMessageBus:       aiMessageBus,
		correlationTracker: correlationTracker,
	}
}

// ProcessWithAgents handles AI-native execution with bidirectional agent communication via events
// This is stateless and supports concurrent conversations using correlation IDs
func (e *AIConversationEngine) ProcessWithAgents(ctx context.Context, userInput, userID, agentContext string) (string, error) {
	// Generate unique correlation ID for this conversation
	correlationID := fmt.Sprintf("conv-%s-%s", userID, uuid.New().String())

	// Get AI decision using improved system prompt
	systemPrompt := e.buildSystemPrompt(agentContext)
	userPrompt := fmt.Sprintf("User request: %s", userInput)

	// Get AI decision
	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI call failed: %w", err)
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

// handleAgentEvent processes AI's decision to send event to an agent
func (e *AIConversationEngine) handleAgentEvent(ctx context.Context, aiResponse, originalRequest, userID, agentContext, correlationID string) (string, error) {

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
		},
		Timeout: DefaultEventTimeout,
	}

	// Send event to agent via message bus
	err := e.aiMessageBus.SendToAgent(ctx, eventMsg)
	if err != nil {
		return "", fmt.Errorf("failed to send event to agent %s: %w", agentID, err)
	}

	// Wait for agent response using correlation tracker (stateless)
	agentResponse, err := e.waitForAgentResponseWithCorrelation(ctx, correlationID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to receive agent response: %w", err)
	}

	// Let AI process the agent response
	return e.processAgentEventResponse(ctx, agentResponse, originalRequest, userID, agentContext)
}

// waitForAgentResponseWithCorrelation waits for an agent response using correlation tracking
func (e *AIConversationEngine) waitForAgentResponseWithCorrelation(ctx context.Context, correlationID, userID string) (*messaging.AgentToAIMessage, error) {

	// Register request with correlation tracker
	timeout := 30 * time.Second
	responseChan := e.correlationTracker.RegisterRequest(correlationID, userID, timeout)

	// Subscribe to the same channel as the original working engine
	responseChannel, err := e.aiMessageBus.Subscribe(ctx, "ai-orchestrator")
	if err != nil {
		e.correlationTracker.CleanupRequest(correlationID)
		return nil, fmt.Errorf("failed to subscribe for agent responses: %w", err)
	}

	// Start listening for agent responses and route them through correlation tracker
	go func() {
		defer func() {
			// Clean up subscription when done
			e.correlationTracker.CleanupRequest(correlationID)
		}()

		for {
			select {
			case msg, ok := <-responseChannel:
				if !ok {
					// Channel is closed, stop listening
					return
				}
				if msg != nil {
					if msg.MessageType == messaging.MessageTypeAgentToAI && msg.CorrelationID == correlationID {
						// Convert to AgentToAIMessage and route through correlation tracker
						agentMsg := &messaging.AgentToAIMessage{
							AgentID:       msg.FromID,
							Content:       msg.Content,
							CorrelationID: msg.CorrelationID,
							MessageType:   msg.MessageType,
						}

						if e.correlationTracker.RouteResponse(agentMsg) {
						} else {
						}
						return
					} else {
						// Message doesn't match correlation ID, continue waiting
					}
				}
				// Note: removed the nil message log since it's just noise from closed channels
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
		return nil, fmt.Errorf("received nil response for correlation %s", correlationID)
	case <-ctx.Done():
		e.correlationTracker.CleanupRequest(correlationID)
		return nil, ctx.Err()
	case <-time.After(timeout):
		e.correlationTracker.CleanupRequest(correlationID)
		return nil, fmt.Errorf("timeout waiting for agent response (correlation: %s)", correlationID)
	}
}

// processAgentEventResponse lets AI decide what to do with agent response event
func (e *AIConversationEngine) processAgentEventResponse(ctx context.Context, agentResponse *messaging.AgentToAIMessage, originalRequest, userID, agentContext string) (string, error) {
	systemPrompt := fmt.Sprintf(`You are an AI orchestrator processing an agent response EVENT.

Original user request: %s
Agent ID: %s
Agent response: %s
Agent context: %v

Based on the agent response EVENT, decide:
1. Do you need to send another event to another agent?
2. Do you need to ask the agent for clarification via event?
3. Can you provide final response to user?

If sending event to another agent, respond with:
%s
Agent: [agent-id]
Action: [capability-name]
Content: [natural language instruction]
Intent: [what you want]

If ready to respond to user, respond with:
%s
[your final response incorporating agent results]`,
		originalRequest, agentResponse.AgentID, agentResponse.Content, agentResponse.Context, EventPrefix, UserResponsePrefix)

	userPrompt := "Process the agent response event and decide next action."

	// Get AI decision on how to proceed
	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI processing of agent response failed: %w", err)
	}

	// Check if AI wants to send another event to an agent
	if strings.Contains(response, EventPrefix) {
		// For now, just indicate multi-agent coordination
		return "AI is coordinating multiple agents via events: " + response, nil
	}

	// Extract final user response
	if strings.Contains(response, UserResponsePrefix) {
		return e.extractUserResponse(response), nil
	}

	// Fallback - return AI response as-is
	return response, nil
}

// buildSystemPrompt creates the system prompt for AI decision making
func (e *AIConversationEngine) buildSystemPrompt(agentContext string) string {
	prompt := fmt.Sprintf(`You are an AI orchestrator with access to these agents:

%s

You orchestrate using EVENTS. When you need an agent:

1. Analyze user request
2. Send event to appropriate agent
3. Wait for agent response event
4. Process response and decide next action
5. Provide final response to user

When calling an agent, respond EXACTLY with:
%s
Agent: [agent-id]
Action: [capability-name]
Content: [natural language instruction to agent]
Intent: [what you want the agent to do]

When ready to respond to user, respond with:
%s
[your response to the user]`, agentContext, EventPrefix, UserResponsePrefix)

	return prompt
}

// extractSection extracts a named section from AI response
func (e *AIConversationEngine) extractSection(response, sectionName string) string {
	lines := strings.Split(response, "\n")
	var sectionContent strings.Builder
	inSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, sectionName) {
			inSection = true
			// Extract content after the colon
			if colonIndex := strings.Index(line, ":"); colonIndex != -1 && len(line) > colonIndex+1 {
				sectionContent.WriteString(strings.TrimSpace(line[colonIndex+1:]))
			}
			continue
		}
		if inSection {
			if strings.Contains(line, ":") && (strings.HasPrefix(line, "Agent:") || strings.HasPrefix(line, "Action:") || strings.HasPrefix(line, "Content:") || strings.HasPrefix(line, "Intent:")) {
				break
			}
			if line != "" {
				if sectionContent.Len() > 0 {
					sectionContent.WriteString(" ")
				}
				sectionContent.WriteString(line)
			}
		}
	}

	return sectionContent.String()
}

// extractUserResponse extracts the user response from AI output
func (e *AIConversationEngine) extractUserResponse(response string) string {
	if idx := strings.Index(response, UserResponsePrefix); idx != -1 {
		userResponse := strings.TrimSpace(response[idx+len(UserResponsePrefix):])
		// Remove any trailing sections that might be for internal processing
		if endIdx := strings.Index(userResponse, "SEND_EVENT:"); endIdx != -1 {
			userResponse = strings.TrimSpace(userResponse[:endIdx])
		}
		return userResponse
	}
	return response
}
