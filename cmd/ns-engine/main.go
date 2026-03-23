package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/app"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.RunStreamEngine(ctx, cfg); err != nil {
		log.Fatalf("ns-engine exited with error: %v", err)
	}
}
