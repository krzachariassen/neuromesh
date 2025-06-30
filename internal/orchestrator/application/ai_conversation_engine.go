package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	aiDomain "neuromesh/internal/ai/domain"
	"neuromesh/internal/messaging"
)

const (
	// AI prompting constants
	EventPrefix        = "SEND_EVENT:"
	UserResponsePrefix = "USER_RESPONSE:"

	// Event timeout
	DefaultEventTimeout = 30 * time.Second

	// Event timeout configuration
	TextProcessorAgentID = "text-processor"
)

// AIConversationEngine orchestrates AI-native conversations with agents using events
// This replaces the rigid ExecutionCoordinator with AI-mediated execution via RabbitMQ events
type AIConversationEngine struct {
	aiProvider       aiDomain.AIProvider
	aiMessageBus     messaging.AIMessageBus
	conversationID   string
	responseChannel  <-chan *messaging.Message
	subscriptionOnce sync.Once
	channelMutex     sync.RWMutex
}

// NewAIConversationEngine creates a new AI conversation engine
func NewAIConversationEngine(aiProvider aiDomain.AIProvider, aiMessageBus messaging.AIMessageBus) *AIConversationEngine {
	engine := &AIConversationEngine{
		aiProvider:   aiProvider,
		aiMessageBus: aiMessageBus,
	}

	// Prepare queue for receiving agent responses
	// Use a background context since this is initialization
	ctx := context.Background()
	if err := aiMessageBus.PrepareAgentQueue(ctx, "ai-orchestrator"); err != nil {
		// Log error but don't fail - queue can be prepared later if needed
		fmt.Printf("Warning: Failed to prepare orchestrator queue: %v\n", err)
	}

	return engine
}

// buildSystemPrompt creates the AI orchestration system prompt
func (e *AIConversationEngine) buildSystemPrompt(agentContext string) string {
	return fmt.Sprintf(`You are an AI orchestrator with access to these agents:

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
}

// ProcessWithAgents handles AI-native execution with bidirectional agent communication via events
func (e *AIConversationEngine) ProcessWithAgents(ctx context.Context, userInput, userID, agentContext string) (string, error) {
	e.conversationID = fmt.Sprintf("conv-%s-%d", userID, time.Now().Unix())

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
		return e.handleAgentEvent(ctx, response, userInput, userID, agentContext)
	}

	// Extract direct user response
	if strings.Contains(response, UserResponsePrefix) {
		return e.extractUserResponse(response), nil
	}

	// Fallback - return AI response as-is
	return response, nil
}

// handleAgentEvent processes AI's decision to send event to an agent
func (e *AIConversationEngine) handleAgentEvent(ctx context.Context, aiResponse, originalRequest, userID, agentContext string) (string, error) {
	// Parse AI's agent event instruction
	agentID := e.extractSection(aiResponse, "Agent:")
	action := e.extractSection(aiResponse, "Action:")
	content := e.extractSection(aiResponse, "Content:")
	intent := e.extractSection(aiResponse, "Intent:")

	// Create AI-to-Agent event message
	eventMsg := &messaging.AIToAgentMessage{
		AgentID:       agentID,
		Content:       content,
		Intent:        intent,
		CorrelationID: e.conversationID,
		Context: map[string]interface{}{
			"original_request": originalRequest,
			"user_id":          userID,
			"action":           action,
		},
		Timeout: DefaultEventTimeout,
	}

	// Send event to agent via RabbitMQ
	err := e.aiMessageBus.SendToAgent(ctx, eventMsg)
	if err != nil {
		return "", fmt.Errorf("failed to send event to agent %s: %w", agentID, err)
	}

	// Wait for real agent response via RabbitMQ events (Step 1.6: Real Bidirectional Events)
	agentResponse, err := e.waitForAgentResponse(ctx, eventMsg.CorrelationID)
	if err != nil {
		return "", fmt.Errorf("failed to receive agent response: %w", err)
	}

	// Let AI process the agent response via event
	return e.processAgentEventResponse(ctx, agentResponse, originalRequest, userID, agentContext)
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

	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI call failed: %w", err)
	}

	// Check if AI wants to send event to another agent
	if strings.Contains(response, EventPrefix) {
		// For now, just indicate multi-agent coordination
		return "AI is coordinating multiple agents via events: " + response, nil
	}

	// Extract user response
	return e.extractUserResponse(response), nil
}

// ensureSubscription ensures we have a single persistent subscription channel
func (e *AIConversationEngine) ensureSubscription(ctx context.Context) error {
	e.channelMutex.Lock()
	defer e.channelMutex.Unlock()

	if e.responseChannel == nil {
		var err error
		e.responseChannel, err = e.aiMessageBus.Subscribe(ctx, "ai-orchestrator")
		if err != nil {
			return fmt.Errorf("failed to create subscription: %w", err)
		}
	}
	return nil
}

// waitForAgentResponse waits for a real agent response via RabbitMQ events
func (e *AIConversationEngine) waitForAgentResponse(ctx context.Context, correlationID string) (*messaging.AgentToAIMessage, error) {
	// Ensure we have a persistent subscription
	if err := e.ensureSubscription(ctx); err != nil {
		return nil, err
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, DefaultEventTimeout)
	defer cancel()

	// Wait for agent response with timeout
	for {
		select {
		case message := <-e.responseChannel:
			// Check if this message is a response to our request
			if message != nil && message.CorrelationID == correlationID {
				// Parse the message as an agent response
				agentResponse, err := e.parseAgentResponseMessage(message)
				if err != nil {
					return nil, fmt.Errorf("failed to parse agent response: %w", err)
				}
				return agentResponse, nil
			}
			// If it's not our correlation ID, continue waiting
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for agent response (correlation ID: %s)", correlationID)
		}
	}
}

// parseAgentResponseMessage converts a generic message to AgentToAIMessage
func (e *AIConversationEngine) parseAgentResponseMessage(message *messaging.Message) (*messaging.AgentToAIMessage, error) {
	// For now, create a basic agent response from the message
	// In a full implementation, this would parse JSON payload
	return &messaging.AgentToAIMessage{
		AgentID:       message.FromID,
		Content:       message.Content,
		MessageType:   messaging.MessageTypeResponse,
		CorrelationID: message.CorrelationID,
		Context:       map[string]interface{}{"status": "completed"},
		NeedsHelp:     false,
	}, nil
}

// Helper methods
func (e *AIConversationEngine) extractSection(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), prefix))
		}
	}
	return ""
}

func (e *AIConversationEngine) extractUserResponse(response string) string {
	if idx := strings.Index(response, UserResponsePrefix); idx != -1 {
		return strings.TrimSpace(response[idx+len(UserResponsePrefix):])
	}
	return response
}
