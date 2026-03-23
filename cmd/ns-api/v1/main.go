package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"Go2NetSpectra/internal/api"
	"Go2NetSpectra/internal/config"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := api.RunLegacyHTTPServer(ctx, cfg); err != nil {
		log.Fatalf("Legacy API server is unsupported after the Thrift cutover: %v", err)
	}
}
