package application

import (
	"context"
	"fmt"
	"time"

	"neuromesh/internal/execution/domain"
	"neuromesh/internal/messaging"
	planningDomain "neuromesh/internal/planning/domain"
)

// SynthesisEventHandler handles agent completion events and triggers synthesis using clean domain events
type SynthesisEventHandler struct {
	coordinator *ExecutionCoordinator
	messageBus  messaging.MessageBus  // Use MessageBus with domain events, not EventRouter
	repository  planningDomain.ExecutionPlanRepository
	synthesizer domain.ResultSynthesizer
}

// NewSynthesisEventHandler creates a new synthesis event handler
func NewSynthesisEventHandler(
	coordinator *ExecutionCoordinator,
	messageBus messaging.MessageBus,
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
func (h *SynthesisEventHandler) HandleAgentCompleted(ctx context.Context, event *messaging.AgentCompletedEvent) error {
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

// StartEventListener starts listening for agent completion events using clean domain events
func (h *SynthesisEventHandler) StartEventListener(ctx context.Context) error {
	// Validate dependencies
	if h.messageBus == nil {
		return fmt.Errorf("message bus is nil")
	}

	// Subscribe to agent completion domain events using clean interface
	eventChan, err := h.messageBus.SubscribeToDomainEvents(ctx, "synthesis-coordinator", "execution.agent.completed")
	if err != nil {
		return fmt.Errorf("failed to subscribe to agent completion events: %w", err)
	}

	// Process events asynchronously
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case domainEvent := <-eventChan:
				// Unmarshal domain event data
				var event messaging.AgentCompletedEvent
				if err := domainEvent.UnmarshalEventData(&event); err != nil {
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
	}()

	return nil
}

// PublishAgentCompletedEvent publishes an agent completion event using clean domain events
func PublishAgentCompletedEvent(ctx context.Context, messageBus messaging.MessageBus, planID, stepID, agentID string) error {
	// Validate dependencies
	if messageBus == nil {
		return fmt.Errorf("messageBus is nil")
	}

	// Create the domain event
	event := &messaging.AgentCompletedEvent{
		PlanID:    planID,
		StepID:    stepID,
		AgentID:   agentID,
		Status:    "completed",
		Timestamp: time.Now().UTC(),
	}

	// Publish using clean domain event interface - no infrastructure leakage
	err := messageBus.PublishDomainEvent(ctx, "execution.agent.completed", event)
	if err != nil {
		return fmt.Errorf("failed to publish agent completed event: %w", err)
	}

	return nil
}
