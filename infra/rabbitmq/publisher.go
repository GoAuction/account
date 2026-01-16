package rabbitmq

import (
	"account/pkg/events"
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	service string
}

func NewRabbitMQPublisher(url, service string) (*RabbitMQPublisher, error) {
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

	if err := channel.Confirm(false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to enable publisher confirms: %w", err)
	}

	zap.L().Info("RabbitMQ publisher connected successfully")

	return &RabbitMQPublisher{
		conn:    conn,
		channel: channel,
		service: service,
	}, nil
}

func (p *RabbitMQPublisher) DeclareExchange(exchange string) error {
	return p.channel.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, exchange string, event *events.Event, headers events.Headers) error {
	if err := p.DeclareExchange(exchange); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	body, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	messageHeaders := amqp.Table{
		"x-trace-id":       headers.TraceID,
		"x-correlation-id": headers.CorrelationID,
		"x-service":        p.service,
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    event.Timestamp,
		Headers:      messageHeaders,
	}

	routingKey := event.GetRoutingKey()

	// Create a dedicated channel for this publish operation to avoid confirmation conflicts
	publishCh, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create publish channel: %w", err)
	}
	defer publishCh.Close()

	// Enable confirms on this channel
	if err := publishCh.Confirm(false); err != nil {
		return fmt.Errorf("failed to enable confirms: %w", err)
	}

	// Register for confirmations BEFORE publishing
	confirms := publishCh.NotifyPublish(make(chan amqp.Confirmation, 1))

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := publishCh.PublishWithContext(
		publishCtx,
		exchange,
		routingKey,
		false,
		false,
		msg,
	); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Wait for confirmation
	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("message was not acknowledged by broker")
		}
	case <-publishCtx.Done():
		return fmt.Errorf("publish confirmation timeout")
	}

	zap.L().Info("Event published successfully",
		zap.String("exchange", exchange),
		zap.String("routingKey", routingKey),
		zap.String("event", event.Event),
		zap.String("traceId", headers.TraceID),
	)

	return nil
}

func (p *RabbitMQPublisher) IsHealthy() bool {
	if p == nil || p.conn == nil || p.channel == nil {
		return false
	}

	return !p.conn.IsClosed() && !p.channel.IsClosed()
}

func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			zap.L().Error("Failed to close channel", zap.Error(err))
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			zap.L().Error("Failed to close connection", zap.Error(err))
			return err
		}
	}
	zap.L().Info("RabbitMQ publisher closed")
	return nil
}
