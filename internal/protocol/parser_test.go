package protocol

import (
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func TestParsePacket(t *testing.T) {
	handle, err := pcap.OpenOffline("../../../test/data/test.pcap")
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
