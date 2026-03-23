package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"Go2NetSpectra/internal/ai"
	"Go2NetSpectra/internal/config"
)

func main() {
	configFile := flag.String("config", "configs/config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := ai.RunServer(ctx, cfg); err != nil {
		log.Fatalf("AI server exited with error: %v", err)
	}
}
