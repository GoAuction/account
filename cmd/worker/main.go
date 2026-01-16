package main

import (
	"account/domain"
	"account/infra/rabbitmq"
	"account/internal/consumers"
	"account/pkg/config"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := zapConfig.Build()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	zap.L().Info("Account Worker Service starting...")

	// Load application config
	appConfig := config.Read()
	zap.L().Info("Worker config loaded",
		zap.String("serviceName", appConfig.ServiceName),
		zap.String("rabbitMQURL", appConfig.RabbitMQURL),
	)

	// Build DSN from config
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		appConfig.PostgresHost,
		appConfig.PostgresUsername,
		appConfig.PostgresPassword,
		appConfig.PostgresDatabase,
		appConfig.PostgresPort,
		appConfig.PostgresSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Error("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}
	zap.L().Info("Database connection established")

	zap.L().Info("Database migration started")
	db.AutoMigrate(&domain.Profile{})
	zap.L().Info("Database migration completed")

	// Validate RabbitMQ URL
	if appConfig.RabbitMQURL == "" {
		zap.L().Fatal("RABBITMQ_URL is required for worker service")
	}
	// Initialize bid event handler
	userHandler := consumers.NewUserEventHandler(
		db,
		zap.L(),
	)

	// Configure bid consumer
	// This consumes events from the "bid" service
	userConsumerConfig := rabbitmq.ConsumerConfig{
		Exchange:       "identity.user",
		QueueName:      "identity.user.all.v1",
		RoutingKeys:    []string{"identity.#"},
		ServiceName:    appConfig.ServiceName,
		PrefetchCount:  10,
		WorkerPoolSize: 20,
	}

	userConsumer, err := rabbitmq.NewConsumer(appConfig.RabbitMQURL, userConsumerConfig)
	if err != nil {
		zap.L().Fatal("Failed to create user consumer", zap.Error(err))
	}
	defer userConsumer.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start bid consumer in goroutine
	go func() {
		zap.L().Info("Starting user event consumer...")
		if err := userConsumer.Consume(ctx, userHandler.HandleEvent); err != nil {
			if err != context.Canceled {
				zap.L().Error("user consumer error", zap.Error(err))
			}
		}
	}()

	zap.L().Info("Worker service started successfully. Waiting for events...")
	zap.L().Info("Consuming from exchanges",
		zap.String("userExchange", "identity.user"),
	)
	zap.L().Info("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigChan
	zap.L().Info("Shutdown signal received, stopping worker service...")
	cancel()

	zap.L().Info("Worker service stopped gracefully")
}
