package main

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/manager"
	"Go2NetSpectra/pkg/pcap"
	"fmt"
	"log"
	"os"
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

	// 3. Initialize modules
	managerImpl, err := manager.NewManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}
	log.Println("Manager initialized.")

	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		log.Fatalf("Failed to open pcap file: %v", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from '%s'...", pcapFilePath)

	// 4. Start the processing pipeline
	managerImpl.Start()
	log.Println("Manager started.")

	// 5. Start reading packets and feeding them to the manager
	pcapReader.ReadPackets(managerImpl.InputChannel())
	log.Println("Finished reading all packets from pcap file.")

	// 6. Graceful shutdown
	log.Println("Shutting down manager...")
	managerImpl.Stop()
	log.Println("Shutdown complete.")
}
