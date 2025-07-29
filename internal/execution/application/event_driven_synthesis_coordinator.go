package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
)

// EventDrivenSynthesisCoordinator coordinates synthesis using proper event-driven architecture
type EventDrivenSynthesisCoordinator struct {
	eventRouter    messaging.EventRouter
	logger         logging.Logger
	coordinator    *ExecutionCoordinator
	mu            sync.RWMutex
	triggeredPlans map[string]bool
}

// NewEventDrivenSynthesisCoordinator creates a new event-driven synthesis coordinator  
func NewEventDrivenSynthesisCoordinator(
	eventRouter messaging.EventRouter,
	logger logging.Logger,
	coordinator *ExecutionCoordinator,
) *EventDrivenSynthesisCoordinator {
	return &EventDrivenSynthesisCoordinator{
		eventRouter:    eventRouter,
		logger:         logger,
		coordinator:    coordinator,
		triggeredPlans: make(map[string]bool),
	}
}

// StartListening starts listening for agent completion events
func (c *EventDrivenSynthesisCoordinator) StartListening(ctx context.Context) error {
	// Setup execution exchange
	err := c.eventRouter.SetupEventExchange(ctx, "execution")
	if err != nil {
		return fmt.Errorf("failed to setup execution exchange: %w", err)
	}

	// Subscribe to agent completion events  
	eventChan, err := c.eventRouter.SubscribeToEvents(ctx, "execution", "synthesis-coordinator", "agent.completed")
	if err != nil {
		return fmt.Errorf("failed to subscribe to agent completion events: %w", err)
	}

	// Process events asynchronously
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Synthesis coordinator stopping due to context cancellation")
				return
			case eventMsg := <-eventChan:
				if err := c.handleAgentCompleted(ctx, eventMsg); err != nil {
					c.logger.Error("Failed to handle agent completion event", err)
				}
			}
		}
	}()

	c.logger.Info("✅ Event-driven synthesis coordinator started")
	return nil
}

// PublishAgentCompleted publishes an agent completion event
func (c *EventDrivenSynthesisCoordinator) PublishAgentCompleted(ctx context.Context, planID, stepID, agentID string) error {
	event := messaging.AgentCompletedEvent{
		PlanID:  planID,
		StepID:  stepID,
		AgentID: agentID,
		Status:  "completed",
		Timestamp: time.Now().UTC(),
	}

	err := c.eventRouter.PublishEvent(ctx, "execution", "agent.completed", event)
	if err != nil {
		return fmt.Errorf("failed to publish agent completion event: %w", err)
	}

	c.logger.Info("✅ Published agent completion event", 
		"planID", planID, 
		"stepID", stepID, 
		"agentID", agentID)
	return nil
}

// handleAgentCompleted handles agent completion events
func (c *EventDrivenSynthesisCoordinator) handleAgentCompleted(ctx context.Context, eventMsg messaging.EventMessage) error {
	// Unmarshal the event
	var event messaging.AgentCompletedEvent
	if err := json.Unmarshal(eventMsg.EventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal agent completion event: %w", err)
	}

	c.logger.Info("🎯 Received agent completion event", 
		"planID", event.PlanID, 
		"stepID", event.StepID, 
		"agentID", event.AgentID,
		"status", event.Status)

	// Only process successful completions
	if event.Status != "completed" && event.Status != "success" {
		c.logger.Info("Ignoring non-successful agent completion", "status", event.Status)
		return nil
	}

	// Check if we've already triggered synthesis for this plan
	c.mu.Lock()
	if c.triggeredPlans[event.PlanID] {
		c.mu.Unlock()
		c.logger.Info("Synthesis already triggered for plan", "planID", event.PlanID)
		return nil
	}
	c.mu.Unlock()

	// Check if execution plan is complete
	isComplete, err := c.coordinator.IsExecutionPlanComplete(ctx, event.PlanID)
	if err != nil {
		return fmt.Errorf("failed to check execution plan completion: %w", err)
	}

	if !isComplete {
		c.logger.Info("Execution plan not yet complete, waiting for more agents", "planID", event.PlanID)
		return nil
	}

	// Mark plan as triggered to prevent duplicate synthesis
	c.mu.Lock()
	c.triggeredPlans[event.PlanID] = true
	c.mu.Unlock()

	// Trigger synthesis
	c.logger.Info("🚀 Triggering synthesis for completed execution plan", "planID", event.PlanID)
	_, err = c.coordinator.TriggerSynthesisWhenComplete(ctx, event.PlanID)
	if err != nil {
		// Remove from triggered plans if synthesis failed
		c.mu.Lock()
		delete(c.triggeredPlans, event.PlanID)
		c.mu.Unlock()
		return fmt.Errorf("failed to trigger synthesis for plan %s: %w", event.PlanID, err)
	}

	c.logger.Info("✅ Synthesis successfully triggered", "planID", event.PlanID)
	return nil
}
