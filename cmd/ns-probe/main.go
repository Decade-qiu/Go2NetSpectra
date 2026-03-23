package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/probe"
	"Go2NetSpectra/internal/protocol"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const (
	snapshotLen   int32 = 1600
	promiscuous         = true
	timeout             = pcap.BlockForever
	modePublish         = "pub"
	modeSubscribe       = "sub"
)

func main() {
	// --- Command-Line Flag Parsing ---
	mode := flag.String("mode", modeSubscribe, "Operating mode: 'pub' captures and publishes packets, 'sub' subscribes and prints.")
	iface := flag.String("iface", "", "Interface to capture packets from. Required in pub mode.")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// --- Mode Dispatch ---
	switch *mode {
	case modePublish:
		runProbe(cfg.Probe, *iface)
	case modeSubscribe:
		runSubscriber(cfg.Probe)
	default:
		fmt.Fprintf(os.Stderr, "invalid mode %q\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

// runProbe contains the logic for capturing packets and publishing them to NATS.
func runProbe(cfg config.ProbeConfig, interfaceName string) {
	if interfaceName == "" {
		log.Println("error: -iface flag is required in pub mode")
		flag.Usage()
		os.Exit(1)
	}
	log.Printf("Starting ns-probe in publisher mode on interface %s", interfaceName)

	// Initialize NATS Publisher
	pub, err := probe.NewPublisher(cfg)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer pub.Close()

	// Open device for live capture
	handle, err := pcap.OpenLive(interfaceName, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatalf("failed to open device %s: %v", interfaceName, err)
	}
	defer handle.Close()

	log.Println("Capture started successfully. Publishing packets to NATS...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start processing packets in a separate goroutine
	go func() {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		packetsPublished := 0
		for packet := range packetSource.Packets() {
			info, err := protocol.ParsePacket(packet)
			if err != nil {
				continue
			}
			if err := pub.Publish(packet, info); err != nil {
				log.Printf("failed to publish packet: %v", err)
			}
			packetsPublished++
			if packetsPublished%1000 == 0 {
				log.Printf("%d packets published...", packetsPublished)
			}
		}
	}()

	// Wait for a shutdown signal
	<-ctx.Done()
	log.Println("Shutdown signal received, cleaning up...")
}

// runSubscriber contains the logic for subscribing to NATS and printing messages.
func runSubscriber(cfg config.ProbeConfig) {
	log.Println("Starting ns-probe in subscriber mode...")

	// Create a new subscriber
	sub, err := probe.NewSubscriber(cfg)
	if err != nil {
		log.Fatalf("failed to create subscriber: %v", err)
	}
	defer sub.Close()

	// Define the handler function for received packets
	handler := func(info model.PacketInfo) {
		log.Printf("Received Packet: %+v", info)
	}

	// Start listening for messages
	if err := sub.Start(handler); err != nil {
		log.Fatalf("subscriber failed to start: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Wait for a shutdown signal
	<-ctx.Done()
	log.Println("Shutdown signal received, cleaning up...")
}
