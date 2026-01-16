package rabbitmq

import (
	"account/pkg/events"
	"context"
	"encoding/json"
	"fmt"

	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type EventHandler func(ctx context.Context, event *events.Event) error

type Consumer struct {
	conn           *amqp.Connection
	channel        *amqp.Channel
	queueName      string
	serviceName    string
	workerPoolSize int
}

type ConsumerConfig struct {
	Exchange       string
	QueueName      string
	RoutingKeys    []string
	ServiceName    string
	PrefetchCount  int
	WorkerPoolSize int
}

func NewConsumer(url string, config ConsumerConfig) (*Consumer, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 5; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		zap.L().Warn("Failed to connect to RabbitMQ, retrying...",
			zap.Int("attempt", i+1),
			zap.Error(err))
		time.Sleep(time.Second * time.Duration(i+1))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	prefetchCount := config.PrefetchCount
	if prefetchCount == 0 {
		prefetchCount = 10
	}
	if err := channel.Qos(prefetchCount, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	if err := channel.ExchangeDeclare(
		config.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	dlxName := config.Exchange + ".dlx"
	if err := channel.ExchangeDeclare(
		dlxName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLX: %w", err)
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange": dlxName,
	}
	queue, err := channel.QueueDeclare(
		config.QueueName,
		true,
		false,
		false,
		false,
		queueArgs,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Declare dead letter queue
	dlqName := config.QueueName + ".dlq"
	_, err = channel.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	for _, routingKey := range config.RoutingKeys {
		if err := channel.QueueBind(
			dlqName,
			routingKey,
			dlxName,
			false,
			nil,
		); err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to bind DLQ: %w", err)
		}
	}

	for _, routingKey := range config.RoutingKeys {
		if err := channel.QueueBind(
			queue.Name,
			routingKey,
			config.Exchange,
			false,
			nil,
		); err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to bind queue: %w", err)
		}
	}

	workerPoolSize := config.WorkerPoolSize
	if workerPoolSize == 0 {
		workerPoolSize = 5
	}

	zap.L().Info("RabbitMQ consumer created successfully",
		zap.String("queue", config.QueueName),
		zap.String("exchange", config.Exchange),
		zap.Strings("routingKeys", config.RoutingKeys),
		zap.Int("workerPoolSize", workerPoolSize),
	)

	return &Consumer{
		conn:           conn,
		channel:        channel,
		queueName:      config.QueueName,
		serviceName:    config.ServiceName,
		workerPoolSize: workerPoolSize,
	}, nil
}

func (c *Consumer) Consume(ctx context.Context, handler EventHandler) error {
	msgs, err := c.channel.Consume(
		c.queueName,
		c.serviceName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	zap.L().Info("Started consuming messages",
		zap.String("queue", c.queueName),
		zap.Int("workerPoolSize", c.workerPoolSize),
	)

	semaphore := make(chan struct{}, c.workerPoolSize)

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Consumer context cancelled, stopping...")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				zap.L().Warn("Message channel closed")
				return fmt.Errorf("message channel closed")
			}

			semaphore <- struct{}{}

			go func(m amqp.Delivery) {
				defer func() { <-semaphore }()
				c.handleMessage(ctx, m, handler)
			}(msg)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery, handler EventHandler) {
	traceID, _ := msg.Headers["x-trace-id"].(string)
	correlationID, _ := msg.Headers["x-correlation-id"].(string)
	service, _ := msg.Headers["x-service"].(string)

	zap.L().Info("Received message",
		zap.String("queue", c.queueName),
		zap.String("routingKey", msg.RoutingKey),
		zap.String("traceId", traceID),
		zap.String("correlationId", correlationID),
		zap.String("sourceService", service),
	)

	var event events.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		zap.L().Error("Failed to unmarshal event",
			zap.Error(err),
			zap.String("traceId", traceID),
		)
		msg.Nack(false, false)
		return
	}

	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := handler(processCtx, &event); err != nil {
		zap.L().Error("Failed to process event",
			zap.Error(err),
			zap.String("event", event.Event),
			zap.String("traceId", traceID),
		)

		msg.Nack(false, false)
		return
	}

	if err := msg.Ack(false); err != nil {
		zap.L().Error("Failed to acknowledge message",
			zap.Error(err),
			zap.String("traceId", traceID),
		)
	} else {
		zap.L().Info("Successfully processed event",
			zap.String("event", event.Event),
			zap.String("traceId", traceID),
		)
	}
}

func (c *Consumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			zap.L().Error("Failed to close channel", zap.Error(err))
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			zap.L().Error("Failed to close connection", zap.Error(err))
			return err
		}
	}
	zap.L().Info("RabbitMQ consumer closed")
	return nil
}
