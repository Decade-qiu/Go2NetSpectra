package app

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/streamaggregator"
	"context"
	"fmt"
	"log"
)

// RunStreamEngine starts the stream aggregator and blocks until shutdown.
func RunStreamEngine(ctx context.Context, cfg *config.Config) error {
	log.Println("Starting ns-engine...")

	streamAgg, err := streamaggregator.NewStreamAggregator(cfg)
	if err != nil {
		return fmt.Errorf("failed to create stream aggregator: %w", err)
	}

	if err := streamAgg.Start(); err != nil {
		return fmt.Errorf("failed to start stream aggregator: %w", err)
	}
	<-ctx.Done()

	log.Println("Shutdown signal received, stopping aggregator...")
	streamAgg.Stop()
	log.Println("Shutdown complete.")
	return nil
}
