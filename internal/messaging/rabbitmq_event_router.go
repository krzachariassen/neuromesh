package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"neuromesh/internal/logging"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQEventRouter implements EventRouter using RabbitMQ exchanges and routing keys
type RabbitMQEventRouter struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  logging.Logger
}

// NewRabbitMQEventRouter creates a new RabbitMQ-based event router
func NewRabbitMQEventRouter(conn *amqp.Connection, logger logging.Logger) (*RabbitMQEventRouter, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel for event router: %w", err)
	}

	return &RabbitMQEventRouter{
		conn:    conn,
		channel: channel,
		logger:  logger,
	}, nil
}

// SetupEventExchange ensures an exchange exists for event routing
func (r *RabbitMQEventRouter) SetupEventExchange(ctx context.Context, exchangeName string) error {
	err := r.channel.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type - using topic for routing key patterns
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare event exchange %s: %w", exchangeName, err)
	}

	r.logger.Info("✅ Event exchange setup complete", map[string]interface{}{
		"exchange": exchangeName,
		"type":     "topic",
	})
	return nil
}

// PublishEvent publishes an event to a specific exchange with routing key
func (r *RabbitMQEventRouter) PublishEvent(ctx context.Context, exchange, routingKey string, event interface{}) error {
	// Marshal event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create event message wrapper
	eventMsg := EventMessage{
		Exchange:      exchange,
		RoutingKey:    routingKey,
		EventType:     routingKey, // Use routing key as event type for simplicity
		EventData:     json.RawMessage(eventData),
		Timestamp:     time.Now(),
		CorrelationID: fmt.Sprintf("event-%d", time.Now().UnixNano()),
		Metadata: map[string]interface{}{
			"source": "neuromesh-execution",
		},
	}

	// Marshal the full event message
	msgData, err := json.Marshal(eventMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal event message: %w", err)
	}

	// Publish to exchange with routing key
	err = r.channel.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         msgData,
			DeliveryMode: amqp.Persistent, // Make message persistent
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish event to exchange %s with routing key %s: %w", exchange, routingKey, err)
	}

	r.logger.Info("📤 Event published", map[string]interface{}{
		"exchange":    exchange,
		"routing_key": routingKey,
		"event_type":  routingKey,
	})
	return nil
}

// SubscribeToEvents subscribes to events matching routing key pattern
func (r *RabbitMQEventRouter) SubscribeToEvents(ctx context.Context, exchange, queueName, routingKey string) (<-chan EventMessage, error) {
	// Declare queue for this subscription
	queue, err := r.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// Bind queue to exchange with routing key pattern
	err = r.channel.QueueBind(
		queue.Name,  // queue name
		routingKey,  // routing key pattern
		exchange,    // exchange
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue %s to exchange %s with routing key %s: %w", queueName, exchange, routingKey, err)
	}

	// Start consuming
	msgs, err := r.channel.Consume(
		queue.Name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer for queue %s: %w", queueName, err)
	}

	// Convert AMQP delivery channel to EventMessage channel
	eventChan := make(chan EventMessage, 100)
	go func() {
		defer close(eventChan)
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-msgs:
				if !ok {
					return
				}

				var eventMsg EventMessage
				if err := json.Unmarshal(delivery.Body, &eventMsg); err != nil {
					r.logger.Error("Failed to unmarshal event message", err,
						"queue", queueName,
					)
					continue
				}

				select {
				case eventChan <- eventMsg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	r.logger.Info("📥 Event subscription established", map[string]interface{}{
		"exchange":    exchange,
		"queue":       queueName,
		"routing_key": routingKey,
	})

	return eventChan, nil
}

// Close cleans up event router resources
func (r *RabbitMQEventRouter) Close() error {
	if r.channel != nil {
		return r.channel.Close()
	}
	return nil
}
