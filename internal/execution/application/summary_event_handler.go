package application

import (
	"context"
	"fmt"

	"neuromesh/internal/messaging"
	planningDomain "neuromesh/internal/planning/domain"
)

// SummaryEventHandler handles agent completion events and triggers conversation summarization
// This replaces the old SynthesisEventHandler with clean conversation summary domain logic
type SummaryEventHandler struct {
	messageBus             messaging.MessageBus
	repository             planningDomain.ExecutionPlanRepository
	conversationSummarizer *ConversationSummarizer
}

// NewSummaryEventHandler creates a new summary event handler
func NewSummaryEventHandler(
	messageBus messaging.MessageBus,
	repository planningDomain.ExecutionPlanRepository,
	conversationSummarizer *ConversationSummarizer,
) *SummaryEventHandler {
	return &SummaryEventHandler{
		messageBus:             messageBus,
		repository:             repository,
		conversationSummarizer: conversationSummarizer,
	}
}

// StartEventListener starts listening for agent completion events using clean domain events
func (h *SummaryEventHandler) StartEventListener(ctx context.Context) error {
	// Validate dependencies
	if h.messageBus == nil {
		return fmt.Errorf("message bus is nil")
	}

	// Subscribe to agent completion domain events using clean interface
	eventChan, err := h.messageBus.SubscribeToDomainEvents(ctx, "summary-coordinator", "execution.agent.completed")
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

// HandleAgentCompleted handles agent completion events and triggers conversation summarization
func (h *SummaryEventHandler) HandleAgentCompleted(ctx context.Context, event *messaging.AgentCompletedEvent) error {
	// Validate dependencies
	if h.conversationSummarizer == nil {
		return fmt.Errorf("conversation summarizer is nil")
	}
	if h.repository == nil {
		return fmt.Errorf("repository is nil")
	}
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Check if all agents for this execution plan have completed
	complete, err := h.isExecutionPlanComplete(ctx, event.PlanID)
	if err != nil {
		return fmt.Errorf("failed to check execution plan completion: %w", err)
	}

	if !complete {
		// Not all agents completed yet, wait for more
		return nil
	}

	// All agents completed - create conversation summary
	// Get the actual conversation ID linked to this execution plan
	conversationID, err := h.repository.GetConversationIDByPlanID(ctx, event.PlanID)
	if err != nil {
		return fmt.Errorf("failed to get conversation ID for plan %s: %w", event.PlanID, err)
	}

	summary, err := h.conversationSummarizer.SummarizeConversation(ctx, conversationID, event.PlanID)
	if err != nil {
		return fmt.Errorf("failed to create conversation summary: %w", err)
	}

	// Update execution plan status to COMPLETED after successful summarization
	plan, err := h.repository.GetByID(ctx, event.PlanID)
	if err != nil {
		fmt.Printf("Warning: Failed to get execution plan for status update %s: %v\n", event.PlanID, err)
	} else {
		plan.Status = planningDomain.ExecutionPlanStatusCompleted
		err = h.repository.Update(ctx, plan)
		if err != nil {
			// Log error but don't fail - summarization already completed successfully
			fmt.Printf("Warning: Failed to update execution plan status for plan %s: %v\n", event.PlanID, err)
		}
	}

	// Log summarization completion
	fmt.Printf("✅ Conversation summary completed for plan %s (summary ID: %s)\n", event.PlanID, summary.ID)

	return nil
}

// isExecutionPlanComplete checks if all agents for an execution plan have completed
func (h *SummaryEventHandler) isExecutionPlanComplete(ctx context.Context, planID string) (bool, error) {
	plan, err := h.repository.GetByID(ctx, planID)
	if err != nil {
		return false, fmt.Errorf("failed to get execution plan: %w", err)
	}

	// Check if all steps are completed
	for _, step := range plan.Steps {
		if step.Status != planningDomain.ExecutionStepStatusCompleted {
			return false, nil
		}
	}

	return true, nil
}

// PublishAgentCompletedEvent publishes an agent completion event to trigger conversation summarization
func PublishAgentCompletedEvent(ctx context.Context, messageBus messaging.MessageBus, planID, stepID, agentID string) error {
	// Validate dependencies
	if messageBus == nil {
		return fmt.Errorf("messageBus is nil")
	}
	if planID == "" {
		return fmt.Errorf("planID cannot be empty")
	}

	// Create the agent completion event
	event := &messaging.AgentCompletedEvent{
		PlanID:  planID,
		StepID:  stepID,
		AgentID: agentID,
	}

	// Publish as domain event using the clean interface
	return messageBus.PublishDomainEvent(ctx, "execution.agent.completed", event)
}
