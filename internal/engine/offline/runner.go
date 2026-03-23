// Package offline runs the offline packet-analysis workflow against pcap
// fixtures without pulling live-capture dependencies into the stream engine.
package offline

import (
	"fmt"
	"log"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/manager"
	"Go2NetSpectra/pkg/pcap"
)

// RunAnalyzer runs the offline analyzer against a pcap file.
func RunAnalyzer(cfg *config.Config, pcapFilePath string) error {
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
