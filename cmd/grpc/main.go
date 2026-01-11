package main

import (
	"account/domain"
	"account/infra/grpc"
	"account/pkg/config"
	accountv1 "account/proto/gen"
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

	zap.L().Info("Account gRPC Service starting...")

	appConfig := config.Read()

	grpcServer, err := grpc.NewServer(appConfig)
	if err != nil {
		zap.L().Error("failed to create grpc server", zap.Error(err))
		os.Exit(1)
	}

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

	accountService := grpc.NewAccountServiceServer(db)
	accountv1.RegisterAccountServiceServer(grpcServer.GetGRPCServer(), accountService)

	zap.L().Info("starting gRPC server...", zap.String("port", appConfig.GRPCPort))
	go func() {
		if err := grpcServer.Start(); err != nil {
			zap.L().Error("failed to start grpc server", zap.Error(err))
			os.Exit(1)
		}
	}()

	gracefulShutdown(grpcServer)
}

func gracefulShutdown(grpcServer *grpc.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	zap.L().Info("Shutting down server...")

	if err := grpcServer.GracefulStop(); err != nil {
		zap.L().Error("Error during server shutdown", zap.Error(err))
	}

	zap.L().Info("Server gracefully stopped")
}
