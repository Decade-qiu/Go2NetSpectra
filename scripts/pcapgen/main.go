package main

import (
	"flag"

	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func main() {
	outputFile := flag.String("o", "test.pcap", "Output pcap file path")
	packetCount := flag.Int("c", 1000, "Number of packets to generate")
	flag.Parse()
 
	f, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	pcapWriter := pcapgo.NewWriter(f)
	if err := pcapWriter.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
		log.Fatalf("Failed to write pcap header: %v", err)
	}

	rand.Seed(time.Now().UnixNano())

	log.Printf("Generating %d packets into %s...", *packetCount, *outputFile)

	for i := 0; i < *packetCount; i++ {
		if (i+1)%100000 == 0 {
			log.Printf("Generated %d packets...", i+1)
		}
		
		// Generate random packet properties
		srcIP := net.IP{byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256))}
		dstIP := net.IP{byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256))}
		srcPort := layers.TCPPort(rand.Intn(65535-1024) + 1024)
		dstPort := layers.TCPPort(rand.Intn(65535-1024) + 1024)
		payloadSize := rand.Intn(1400) + 50 // Random payload size between 50 and 1450

		// Create layers
		ethLayer := &layers.Ethernet{
			SrcMAC: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			DstMAC: net.HardwareAddr{0x00, 0x66, 0x77, 0x88, 0x99, 0xAA},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ipLayer := &layers.IPv4{
			SrcIP:    srcIP,
			DstIP:    dstIP,
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolTCP,
		}
		tcpLayer := &layers.TCP{
			SrcPort: srcPort,
			DstPort: dstPort,
			Seq:     rand.Uint32(),
			Ack:     rand.Uint32(),
			SYN:     true,
			Window:  14600,
		}
		tcpLayer.SetNetworkLayerForChecksum(ipLayer)

		payload := make([]byte, payloadSize)
		rand.Read(payload)

		// Serialize the packet
		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true,
		}
		if err := gopacket.SerializeLayers(buf, opts, ethLayer, ipLayer, tcpLayer, gopacket.Payload(payload)); err != nil {
			log.Fatalf("Failed to serialize layers: %v", err)
		}

		// Write packet to file
		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(buf.Bytes()),
			Length:        len(buf.Bytes()),
		}
		if err := pcapWriter.WritePacket(ci, buf.Bytes()); err != nil {
			log.Fatalf("Failed to write packet: %v", err)
		}
	}

	log.Printf("Successfully generated %d packets into %s.", *packetCount, *outputFile)
}
