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

// StatelessAIConversationEngine is a stateless, correlation-driven conversation engine
// that supports concurrent conversations using correlation IDs for message routing
type StatelessAIConversationEngine struct {
	aiProvider         aiDomain.AIProvider
	aiMessageBus       messaging.AIMessageBus
	correlationTracker *infrastructure.CorrelationTracker
}

// NewStatelessAIConversationEngine creates a new stateless AI conversation engine
func NewStatelessAIConversationEngine(
	aiProvider aiDomain.AIProvider,
	aiMessageBus messaging.AIMessageBus,
	correlationTracker *infrastructure.CorrelationTracker,
) *StatelessAIConversationEngine {
	return &StatelessAIConversationEngine{
		aiProvider:         aiProvider,
		aiMessageBus:       aiMessageBus,
		correlationTracker: correlationTracker,
	}
}

// ProcessWithAgents handles AI-native execution with bidirectional agent communication via events
// This is stateless and supports concurrent conversations using correlation IDs
func (e *StatelessAIConversationEngine) ProcessWithAgents(ctx context.Context, userInput, userID, agentContext string) (string, error) {
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
func (e *StatelessAIConversationEngine) handleAgentEvent(ctx context.Context, aiResponse, originalRequest, userID, agentContext, correlationID string) (string, error) {
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
func (e *StatelessAIConversationEngine) waitForAgentResponseWithCorrelation(ctx context.Context, correlationID, userID string) (*messaging.AgentToAIMessage, error) {
	// Register request with correlation tracker
	timeout := 30 * time.Second
	responseChan := e.correlationTracker.RegisterRequest(correlationID, userID, timeout)

	// Start listening for agent responses and route them through correlation tracker
	go e.routeAgentResponses(ctx, correlationID)

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

// routeAgentResponses listens for agent responses and routes them through correlation tracker
func (e *StatelessAIConversationEngine) routeAgentResponses(ctx context.Context, correlationID string) {
	// Subscribe to orchestrator channel to receive agent responses
	responseChannel, err := e.aiMessageBus.Subscribe(ctx, "orchestrator")
	if err != nil {
		return
	}

	// Listen for agent responses and route them by correlation ID
	go func() {
		for {
			select {
			case msg := <-responseChannel:
				if msg != nil && msg.MessageType == messaging.MessageTypeAgentToAI && msg.CorrelationID == correlationID {
					// Convert to AgentToAIMessage and route through correlation tracker
					agentMsg := &messaging.AgentToAIMessage{
						AgentID:       msg.FromID,
						Content:       msg.Content,
						CorrelationID: msg.CorrelationID,
						MessageType:   msg.MessageType,
					}

					e.correlationTracker.RouteResponse(agentMsg)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// processAgentEventResponse lets AI decide what to do with agent response event
func (e *StatelessAIConversationEngine) processAgentEventResponse(ctx context.Context, agentResponse *messaging.AgentToAIMessage, originalRequest, userID, agentContext string) (string, error) {
	systemPrompt := fmt.Sprintf(`You are an AI orchestrator processing an agent response EVENT.

Original user request: %s
Agent response: %s

Available agents:
%s

Your role:
1. Analyze the agent's response to determine if it fully answers the user's request
2. If complete, provide a final response to the user using USER_RESPONSE: prefix
3. If incomplete, you can send additional events to agents or ask for clarification

Response format:
- For final user response: USER_RESPONSE: [your response to user]
- For agent event: Use the same SEND_EVENT format as before

Be conversational and helpful in your responses.`, originalRequest, agentResponse.Content, agentContext)

	userPrompt := fmt.Sprintf("Agent %s responded: %s", agentResponse.AgentID, agentResponse.Content)

	// Get AI decision on how to proceed
	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI processing of agent response failed: %w", err)
	}

	// Check if AI wants to send another event to an agent
	if strings.Contains(response, EventPrefix) {
		// Generate new correlation ID for subsequent agent interaction
		newCorrelationID := fmt.Sprintf("conv-%s-%s", userID, uuid.New().String())
		return e.handleAgentEvent(ctx, response, originalRequest, userID, agentContext, newCorrelationID)
	}

	// Extract final user response
	if strings.Contains(response, UserResponsePrefix) {
		return e.extractUserResponse(response), nil
	}

	// Fallback - return AI response as-is
	return response, nil
}

// buildSystemPrompt creates the system prompt for AI decision making
func (e *StatelessAIConversationEngine) buildSystemPrompt(agentContext string) string {
	return fmt.Sprintf(`You are an AI orchestrator that coordinates with specialized agents to help users.

Available agents:
%s

Your capabilities:
1. Analyze user requests and determine which agents can help
2. Send events to agents with specific tasks
3. Process agent responses and provide final answers to users

When you want to send an event to an agent, use this EXACT format:
SEND_EVENT:
Agent: [agent-id]
Action: [what you want the agent to do]
Content: [the specific content/data for the agent]
Intent: [brief description of what you're trying to achieve]

When you have a final response for the user, use this EXACT format:
USER_RESPONSE: [your response to the user]

Always be helpful, accurate, and conversational in your responses.`, agentContext)
}

// extractSection extracts a named section from AI response
func (e *StatelessAIConversationEngine) extractSection(response, sectionName string) string {
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
func (e *StatelessAIConversationEngine) extractUserResponse(response string) string {
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
