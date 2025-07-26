package application

import (
	"context"
	"encoding/json"
	"fmt"

	"neuromesh/internal/execution/domain"
	"neuromesh/internal/messaging"
	planningDomain "neuromesh/internal/planning/domain"
)

// AgentCompletedEvent represents an event when an agent completes execution
type AgentCompletedEvent struct {
	PlanID  string `json:"plan_id"`
	StepID  string `json:"step_id"`
	AgentID string `json:"agent_id"`
}

// SynthesisEventHandler handles agent completion events and triggers synthesis
type SynthesisEventHandler struct {
	coordinator   *ExecutionCoordinator
	messageBus    messaging.AIMessageBus
	repository    planningDomain.ExecutionPlanRepository
	synthesizer   domain.ResultSynthesizer
}

// NewSynthesisEventHandler creates a new synthesis event handler
func NewSynthesisEventHandler(
	coordinator *ExecutionCoordinator,
	messageBus messaging.AIMessageBus,
	repository planningDomain.ExecutionPlanRepository,
	synthesizer domain.ResultSynthesizer,
) *SynthesisEventHandler {
	return &SynthesisEventHandler{
		coordinator: coordinator,
		messageBus:  messageBus,
		repository:  repository,
		synthesizer: synthesizer,
	}
}

// HandleAgentCompleted handles agent completion events
func (h *SynthesisEventHandler) HandleAgentCompleted(ctx context.Context, event *AgentCompletedEvent) error {
	// Validate dependencies
	if h.coordinator == nil {
		return fmt.Errorf("coordinator is nil")
	}

	// Check if execution plan is complete
	isComplete, err := h.coordinator.IsExecutionPlanComplete(ctx, event.PlanID)
	if err != nil {
		return fmt.Errorf("failed to check execution plan completion: %w", err)
	}

	// If plan is not complete, nothing to do
	if !isComplete {
		return nil
	}

	// Trigger synthesis
	_, err = h.coordinator.TriggerSynthesisWhenComplete(ctx, event.PlanID)
	if err != nil {
		return fmt.Errorf("failed to trigger synthesis for plan %s: %w", event.PlanID, err)
	}

	return nil
}

// StartEventListener starts listening for agent completion events
func (h *SynthesisEventHandler) StartEventListener(ctx context.Context) error {
	// Validate dependencies
	if h.messageBus == nil {
		return fmt.Errorf("message bus is nil")
	}

	// Subscribe to agent completion events
	eventChan, err := h.messageBus.Subscribe(ctx, "synthesis-coordination")
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Process events asynchronously
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-eventChan:
				if msg.MessageType == messaging.MessageTypeAgentCompleted {
					var event AgentCompletedEvent
					if err := json.Unmarshal([]byte(msg.Content), &event); err != nil {
						// Log error but continue processing
						fmt.Printf("Warning: Failed to unmarshal agent completion event: %v\n", err)
						continue
					}

					// Handle the event
					if err := h.HandleAgentCompleted(ctx, &event); err != nil {
						// Log error but continue processing
						fmt.Printf("Warning: Failed to handle agent completion event: %v\n", err)
						continue
					}
				}
			}
		}
	}()

	return nil
}

// PublishAgentCompletedEvent publishes an agent completion event to the message bus
func PublishAgentCompletedEvent(ctx context.Context, messageBus messaging.AIMessageBus, planID, stepID, agentID string) error {
	// Validate dependencies
	if messageBus == nil {
		return fmt.Errorf("messageBus is nil")
	}

	// Create the event
	event := &AgentCompletedEvent{
		PlanID:  planID,
		StepID:  stepID,
		AgentID: agentID,
	}

	// Marshal to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal agent completed event: %w", err)
	}

	// Create message
	msg := &messaging.UserToAIMessage{
		UserID:        "synthesis-coordination",
		Content:       string(eventData),
		CorrelationID: fmt.Sprintf("synthesis-%s", planID),
		Context: map[string]interface{}{
			"event_type": "agent.completed",
			"plan_id":    planID,
			"step_id":    stepID,
			"agent_id":   agentID,
		},
	}

	// Send the message
	return messageBus.SendUserToAI(ctx, msg)
}
