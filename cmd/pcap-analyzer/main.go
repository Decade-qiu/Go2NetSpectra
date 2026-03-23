package main

import (
	"fmt"
	"log"
	"os"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/offline"
)

func main() {
	// 1. Get pcap file path from command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/pcap-analyzer/main.go <path_to_pcap_file>")
		os.Exit(1)
	}
	pcapFilePath := os.Args[1]

	// 2. Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded successfully.")

	if err := offline.RunAnalyzer(cfg, pcapFilePath); err != nil {
		log.Fatalf("Offline analyzer exited with error: %v", err)
	}
}
