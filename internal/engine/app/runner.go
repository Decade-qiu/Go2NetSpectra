package app

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/manager"
	"Go2NetSpectra/internal/engine/streamaggregator"
	"Go2NetSpectra/pkg/pcap"
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

// RunOfflineAnalyzer runs the offline analyzer against a PCAP file.
func RunOfflineAnalyzer(cfg *config.Config, pcapFilePath string) error {
	managerImpl, err := manager.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}
	log.Println("Manager initialized.")

	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		return fmt.Errorf("failed to open pcap file: %w", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from %q...", pcapFilePath)

	managerImpl.Start()
	log.Println("Manager started.")

	pcapReader.ReadPackets(managerImpl.InputChannel())
	log.Println("Finished reading all packets from pcap file.")

	log.Println("Shutting down manager...")
	managerImpl.Stop()
	log.Println("Shutdown complete.")
	return nil
}
