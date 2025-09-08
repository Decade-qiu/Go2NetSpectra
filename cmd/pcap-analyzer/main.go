package main

import (
	"Go2NetSpectra/internal/engine/flowaggregator"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/pkg/pcap"
	"fmt"
	"log"
	"os"
)

func main() {
	// 1. Get pcap file path from command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/ns-probe/main.go <path_to_pcap_file>")
		os.Exit(1)
	}
	pcapFilePath := os.Args[1]

	// 2. Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded successfully.")

	// 3. Initialize modules
	aggregator, err := flowaggregator.NewFlowAggregator(cfg)
	if err != nil {
		log.Fatalf("Failed to create aggregator: %v", err)
	}
	log.Println("Flow aggregator initialized.")

	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		log.Fatalf("Failed to open pcap file: %v", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from '%s'...", pcapFilePath)

	// 4. Start the processing pipeline
	aggregator.Start()
	log.Println("Flow aggregator started with", cfg.Aggregator.NumWorkers, "workers.")

	// 5. Start reading packets and feeding them to the aggregator
	pcapReader.ReadPackets(aggregator.InputChannel)
	log.Println("Finished reading all packets from pcap file.")

	// 6. Graceful shutdown
	log.Println("Shutting down aggregator...")
	aggregator.Stop() // This closes the InputChannel and waits for all goroutines to finish.
	log.Println("Shutdown complete.")
}
