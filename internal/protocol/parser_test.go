package protocol

import (
	"testing"

	"Go2NetSpectra/internal/model"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func TestParsePacket(t *testing.T) {
	handle, err := pcap.OpenOffline("../../test/data/test.pcap")
	if err != nil {
		t.Fatalf("Failed to open pcap file: %v", err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()

	var parsedPackets int
	for packet := range packets {
		info, err := ParsePacket(packet)
		if err != nil {
			continue
		}
		parsedPackets++

		if info.FiveTuple.SrcIP == nil {
			t.Error("Source IP should not be nil")
		}
		if info.FiveTuple.DstIP == nil {
			t.Error("Destination IP should not be nil")
		}
		if info.FiveTuple.Protocol == 0 {
			t.Error("Protocol should not be 0")
		}
	}

	if parsedPackets == 0 {
		t.Fatalf("Failed to parse any packet from the pcap file")
	}

	t.Logf("Successfully parsed %d packets", parsedPackets)
}

func TestParsePacketIntoMatchesParsePacket(t *testing.T) {
	handle, err := pcap.OpenOffline("../../test/data/test.pcap")
	if err != nil {
		t.Fatalf("OpenOffline(test.pcap) error: %v", err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packet, ok := <-packetSource.Packets()
	if !ok {
		t.Fatal("test fixture did not yield any packet")
	}

	parsed, err := ParsePacket(packet)
	if err != nil {
		t.Fatalf("ParsePacket() unexpected error: %v", err)
	}

	var reusedInfo model.PacketInfo
	if err := ParsePacketInto(packet, &reusedInfo); err != nil {
		t.Fatalf("ParsePacketInto() unexpected error: %v", err)
	}

	if !reusedInfo.Timestamp.Equal(parsed.Timestamp) {
		t.Fatalf("ParsePacketInto() timestamp = %v, want %v", reusedInfo.Timestamp, parsed.Timestamp)
	}
	if reusedInfo.Length != parsed.Length {
		t.Fatalf("ParsePacketInto() length = %d, want %d", reusedInfo.Length, parsed.Length)
	}
	if !reusedInfo.FiveTuple.SrcIP.Equal(parsed.FiveTuple.SrcIP) {
		t.Fatalf("ParsePacketInto() src ip = %v, want %v", reusedInfo.FiveTuple.SrcIP, parsed.FiveTuple.SrcIP)
	}
	if !reusedInfo.FiveTuple.DstIP.Equal(parsed.FiveTuple.DstIP) {
		t.Fatalf("ParsePacketInto() dst ip = %v, want %v", reusedInfo.FiveTuple.DstIP, parsed.FiveTuple.DstIP)
	}
	if reusedInfo.FiveTuple.SrcPort != parsed.FiveTuple.SrcPort {
		t.Fatalf("ParsePacketInto() src port = %d, want %d", reusedInfo.FiveTuple.SrcPort, parsed.FiveTuple.SrcPort)
	}
	if reusedInfo.FiveTuple.DstPort != parsed.FiveTuple.DstPort {
		t.Fatalf("ParsePacketInto() dst port = %d, want %d", reusedInfo.FiveTuple.DstPort, parsed.FiveTuple.DstPort)
	}
	if reusedInfo.FiveTuple.Protocol != parsed.FiveTuple.Protocol {
		t.Fatalf("ParsePacketInto() protocol = %d, want %d", reusedInfo.FiveTuple.Protocol, parsed.FiveTuple.Protocol)
	}
}
