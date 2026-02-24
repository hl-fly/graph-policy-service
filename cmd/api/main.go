package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hector-leite/graph-policy-service/internal/config"
	"github.com/hector-leite/graph-policy-service/internal/domain/entity"
	"github.com/hector-leite/graph-policy-service/internal/domain/service/inferenceservice"
	"github.com/hector-leite/graph-policy-service/internal/server"
	"github.com/hector-leite/graph-policy-service/internal/server/handler"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		logger.Info("Received signal, initiating graceful shutdown", "signal", sig)
		cancel()
	}()

	cfg := config.LoadConfigs(logger)
	if cfg == nil {
		logger.Error("Failed to load configurations")
		os.Exit(1)
	}

	polictCache, err := lru.New[string, *entity.Policy](1000)
	if err != nil {
		logger.Error("Failed to create policy cache", "error", err)
		os.Exit(1)
	}

	inferenceService := inferenceservice.NewInferenceService(logger)

	inferenceHandler := handler.NewInferenceHandler(logger, polictCache, inferenceService)

	server := server.NewServer(
		server.WithLogger(logger),
		server.WithConfig(cfg),
		server.WithInferenceHandler(inferenceHandler),
	)

	server.Start(ctx)
}
