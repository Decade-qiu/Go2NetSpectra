package main

import (
	"Go2NetSpectra/internal/engine/flowaggregator"
	"Go2NetSpectra/internal/pkg/config"
	"Go2NetSpectra/internal/snapshot"
	"Go2NetSpectra/pkg/pcap"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
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
	writer := snapshot.NewWriter()
	numWorkers := runtime.NumCPU()
	aggregator, err := flowaggregator.NewFlowAggregator(cfg, numWorkers)
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
	log.Println("Flow aggregator started with", numWorkers, "workers.")

	var wg sync.WaitGroup

	// Start the snapshot writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for s := range aggregator.OutputChannel {
			log.Printf("Snapshot received for aggregator '%s', writing to disk...", s.AggregatorName)
			if err := writer.WriteSnapshot(s, cfg.Aggregator.StorageRootPath); err != nil {
				log.Printf("Error writing snapshot: %v", err)
			}
		}
		log.Println("Snapshot writer finished.")
	}()

	// Start reading packets and feeding them to the aggregator
	pcapReader.ReadPackets(aggregator.InputChannel)
	log.Println("Finished reading all packets from pcap file.")

	// 5. Graceful shutdown
	log.Println("Shutting down aggregator...")
	aggregator.Stop() // This will close the InputChannel, which in turn stops the workers and snapshotter.
	
	wg.Wait() // Wait for the snapshot writer to finish processing any remaining snapshots.
	log.Println("Shutdown complete.")
}