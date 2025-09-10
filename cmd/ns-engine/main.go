package main

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/streamaggregator"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting ns-engine...")

	// 1. Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded successfully.")

	// 2. Initialize a new StreamAggregator
	streamAgg, err := streamaggregator.NewStreamAggregator(cfg)
	if err != nil {
		log.Fatalf("Failed to create stream aggregator: %v", err)
	}

	// 3. Start the aggregator
	streamAgg.Start()

	// 4. Wait for a shutdown signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	log.Println("Shutdown signal received, stopping aggregator...")
	streamAgg.Stop()
	log.Println("Shutdown complete.")
}