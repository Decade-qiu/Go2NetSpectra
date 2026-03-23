package main

import (
	"Go2NetSpectra/internal/api"
	"Go2NetSpectra/internal/config"
	"context"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := api.RunQueryAPIServers(ctx, cfg); err != nil {
		log.Fatalf("Query API servers exited with error: %v", err)
	}
}
