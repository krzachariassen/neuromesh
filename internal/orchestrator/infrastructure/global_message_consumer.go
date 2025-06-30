package infrastructure

import (
	"context"
	"fmt"

	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
)

// GlobalMessageConsumer consumes messages from a shared queue and routes them using correlation IDs
type GlobalMessageConsumer struct {
	messageBus         messaging.AIMessageBus
	correlationTracker *CorrelationTracker
	logger             logging.Logger
}

// NewGlobalMessageConsumer creates a new instance of GlobalMessageConsumer
func NewGlobalMessageConsumer(messageBus messaging.AIMessageBus, tracker *CorrelationTracker) *GlobalMessageConsumer {
	return &GlobalMessageConsumer{
		messageBus:         messageBus,
		correlationTracker: tracker,
		logger:             logging.NewNoOpLogger(), // Default logger, can be injected later
	}
}

// StartConsumption starts consuming messages from the specified participant queue
func (gmc *GlobalMessageConsumer) StartConsumption(ctx context.Context, participantID string) error {
	// Subscribe to the message bus
	messageChannel, err := gmc.messageBus.Subscribe(ctx, participantID)
	if err != nil {
		return fmt.Errorf("failed to subscribe to message bus: %w", err)
	}

	// Start message processing goroutine
	go gmc.processMessages(ctx, messageChannel)

	return nil
}

// processMessages processes incoming messages from the message channel
func (gmc *GlobalMessageConsumer) processMessages(ctx context.Context, messageChannel <-chan *messaging.Message) {
	for {
		select {
		case <-ctx.Done():
			gmc.logger.Info("GlobalMessageConsumer: Stopping message processing due to context cancellation")
			return
		case message, ok := <-messageChannel:
			if !ok {
				gmc.logger.Info("GlobalMessageConsumer: Message channel closed, stopping processing")
				return
			}

			// Route the message
			gmc.RouteMessage(message)
		}
	}
}

// RouteMessage routes a message to the appropriate waiting request using correlation ID
func (gmc *GlobalMessageConsumer) RouteMessage(message *messaging.Message) bool {
	// Only route AgentToAI messages (responses from agents)
	if message.MessageType != messaging.MessageTypeAgentToAI {
		gmc.logger.Debug("GlobalMessageConsumer: Ignoring non-AgentToAI message",
			"messageType", message.MessageType,
			"correlationID", message.CorrelationID)
		return false
	}

	// Convert generic Message to AgentToAIMessage for the correlation tracker
	agentToAIMessage := &messaging.AgentToAIMessage{
		AgentID:       message.FromID,
		Content:       message.Content,
		MessageType:   message.MessageType,
		CorrelationID: message.CorrelationID,
		Context:       message.Metadata, // Use metadata as context
		NeedsHelp:     false,            // Default value, could be in metadata
	}

	// Check if NeedsHelp is specified in metadata
	if needsHelp, ok := message.Metadata["needsHelp"].(bool); ok {
		agentToAIMessage.NeedsHelp = needsHelp
	}

	// Route through the correlation tracker
	routed := gmc.correlationTracker.RouteResponse(agentToAIMessage)

	if routed {
		gmc.logger.Debug("GlobalMessageConsumer: Successfully routed message",
			"correlationID", message.CorrelationID,
			"agentID", message.FromID)
	} else {
		gmc.logger.Debug("GlobalMessageConsumer: No waiting request found for correlation ID",
			"correlationID", message.CorrelationID,
			"agentID", message.FromID)
	}

	return routed
}

// SetLogger allows injecting a custom logger
func (gmc *GlobalMessageConsumer) SetLogger(logger logging.Logger) {
	gmc.logger = logger
}
