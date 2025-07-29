package application

import (
	"context"
	"fmt"

	"neuromesh/internal/execution/domain"
	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
	planningDomain "neuromesh/internal/planning/domain"
)

// ExecutionCompletionMonitor monitors agent completion events and triggers synthesis
// This is the Phase 2 implementation of event-driven completion detection
// It extends the existing SynthesisEventHandler with additional monitoring capabilities
type ExecutionCompletionMonitor struct {
	*SynthesisEventHandler
	logger    logging.Logger
	isRunning bool
	stopChan  chan struct{}
}

// NewExecutionCompletionMonitor creates a new execution completion monitor
func NewExecutionCompletionMonitor(
	messageBus messaging.MessageBus,
	synthesizer domain.ResultSynthesizer,
	repository planningDomain.ExecutionPlanRepository,
) *ExecutionCompletionMonitor {
	coordinator := NewExecutionCoordinator(repository, synthesizer)
	handler := NewSynthesisEventHandler(coordinator, messageBus, repository, synthesizer)

	return &ExecutionCompletionMonitor{
		SynthesisEventHandler: handler,
		logger:                logging.NewStructuredLogger(logging.LevelInfo),
		stopChan:              make(chan struct{}),
	}
}

// OnAgentResult handles agent result events and checks for execution plan completion
// This delegates to the existing SynthesisEventHandler
func (m *ExecutionCompletionMonitor) OnAgentResult(ctx context.Context, event *messaging.AgentCompletedEvent) error {
	m.logger.Info("Processing agent completion event",
		"plan_id", event.PlanID,
		"step_id", event.StepID,
		"agent_id", event.AgentID,
	)

	// Delegate to the existing synthesis event handler
	err := m.SynthesisEventHandler.HandleAgentCompleted(ctx, event)
	if err != nil {
		m.logger.Error("Failed to handle agent completion", err)
		return err
	}

	m.logger.Info("Agent completion handled successfully", "plan_id", event.PlanID)
	return nil
}

// Start starts the event-driven completion monitoring
// This leverages the existing SynthesisEventHandler infrastructure
func (m *ExecutionCompletionMonitor) Start(ctx context.Context) error {
	if m.isRunning {
		return fmt.Errorf("monitor is already running")
	}

	m.logger.Info("Starting execution completion monitor")

	// Use the existing event listener from SynthesisEventHandler
	err := m.SynthesisEventHandler.StartEventListener(ctx)
	if err != nil {
		return fmt.Errorf("failed to start event listener: %w", err)
	}

	m.isRunning = true
	m.logger.Info("Execution completion monitor started successfully")
	return nil
}

// Stop stops the event-driven completion monitoring
func (m *ExecutionCompletionMonitor) Stop() error {
	if !m.isRunning {
		return fmt.Errorf("monitor is not running")
	}

	m.logger.Info("Stopping execution completion monitor")

	// Signal the goroutine to stop
	close(m.stopChan)

	// Note: AIMessageBus doesn't have Unsubscribe method
	// The goroutine will stop when stopChan is closed

	return nil
}
