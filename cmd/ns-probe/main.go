package main

import (
	"Go2NetSpectra/internal/engine/protocol"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/probe"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/nats-io/nats.go"
)

const (
	// NATS connection details (hardcoded for now, should be moved to config)
	natsURL   = nats.DefaultURL // Assumes NATS server is running on localhost:4222
	natsSubject = "gons.packets.raw"
)

const (
	snapshotLen int32 = 1600
	promiscuous       = true
	timeout          = pcap.BlockForever
)

func main() {
	// --- Command-Line Flag Parsing ---
	mode := flag.String("mode", "sub", "Operating mode: 'pub' to capture and publish, 'sub' to subscribe and print.")
	iface := flag.String("iface", "", "Interface to capture packets from (required for pub mode).")
	flag.Parse()

	// --- Mode Dispatch ---
	switch *mode {
	case "pub":
		runProbe(*iface)
	case "sub":
		runSubscriber()
	default:
		fmt.Fprintf(os.Stderr, "Invalid mode: %s\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

// runProbe contains the logic for capturing packets and publishing them to NATS.
func runProbe(interfaceName string) {
	if interfaceName == "" {
		log.Println("Error: -iface flag is required for probe mode.")
		flag.Usage()
		os.Exit(1)
	}
	log.Printf("Starting ns-probe in PROBE mode on interface: %s", interfaceName)

	// Initialize NATS Publisher
	pub, err := probe.NewPublisher(natsURL, natsSubject)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer pub.Close()

	// Open device for live capture
	handle, err := pcap.OpenLive(interfaceName, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatalf("Error opening device %s: %v", interfaceName, err)
	}
	defer handle.Close()

	log.Println("Capture started successfully. Publishing packets to NATS...")

	// Set up a channel to handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start processing packets in a separate goroutine
	go func() {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		packetsPublished := 0
		for packet := range packetSource.Packets() {
			info, err := protocol.ParsePacket(packet)
			if err != nil {
				continue // Skip non-IP packets
			}
			if err := pub.Publish(info); err != nil {
				log.Printf("Failed to publish packet: %v", err)
			}
			packetsPublished++
			if packetsPublished%1000 == 0 {
				log.Printf("%d packets published...", packetsPublished)
			}
		}
	}()

	// Wait for a shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, cleaning up...")
}

// runSubscriber contains the logic for subscribing to NATS and printing messages.
func runSubscriber() {
	log.Println("Starting ns-probe in SUBSCRIBER mode...")

	// Create a new subscriber
	sub, err := probe.NewSubscriber(natsURL)
	if err != nil {
		log.Fatalf("Failed to create subscriber: %v", err)
	}
	defer sub.Close()

	// Define the handler function for received packets
	handler := func(info model.PacketInfo) {
		log.Printf("Received Packet: %+v", info)
	}

	// Start listening for messages
	if err := sub.Start(natsSubject, handler); err != nil {
		log.Fatalf("Subscriber failed to start: %v", err)
	}

	// Set up a channel to handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for a shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, cleaning up...")
}