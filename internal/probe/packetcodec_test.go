package probe

import (
	"Go2NetSpectra/internal/model"
	"net"
	"testing"
	"time"
)

func TestPacketInfoRoundTrip(t *testing.T) {
	timestamp := time.Unix(1700000000, 123)
	original := &model.PacketInfo{
		Timestamp: timestamp,
		Length:    128,
		FiveTuple: model.FiveTuple{
			SrcIP:    net.ParseIP("192.0.2.10"),
			DstIP:    net.ParseIP("2001:db8::1"),
			SrcPort:  443,
			DstPort:  8080,
			Protocol: 17,
		},
	}

	pbPacket, err := PacketInfoToProto(original)
	if err != nil {
		t.Fatalf("PacketInfoToProto() unexpected error: %v", err)
	}

	decoded, err := PacketInfoFromProto(pbPacket)
	if err != nil {
		t.Fatalf("PacketInfoFromProto() unexpected error: %v", err)
	}

	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Fatalf("decoded timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}
	if decoded.Length != original.Length {
		t.Fatalf("decoded length = %d, want %d", decoded.Length, original.Length)
	}
	if !decoded.FiveTuple.SrcIP.Equal(original.FiveTuple.SrcIP) {
		t.Fatalf("decoded src ip = %v, want %v", decoded.FiveTuple.SrcIP, original.FiveTuple.SrcIP)
	}
	if !decoded.FiveTuple.DstIP.Equal(original.FiveTuple.DstIP) {
		t.Fatalf("decoded dst ip = %v, want %v", decoded.FiveTuple.DstIP, original.FiveTuple.DstIP)
	}
	if decoded.FiveTuple.SrcPort != original.FiveTuple.SrcPort {
		t.Fatalf("decoded src port = %d, want %d", decoded.FiveTuple.SrcPort, original.FiveTuple.SrcPort)
	}
	if decoded.FiveTuple.DstPort != original.FiveTuple.DstPort {
		t.Fatalf("decoded dst port = %d, want %d", decoded.FiveTuple.DstPort, original.FiveTuple.DstPort)
	}
	if decoded.FiveTuple.Protocol != original.FiveTuple.Protocol {
		t.Fatalf("decoded protocol = %d, want %d", decoded.FiveTuple.Protocol, original.FiveTuple.Protocol)
	}
}

func TestPacketInfoToProtoRejectsNil(t *testing.T) {
	if _, err := PacketInfoToProto(nil); err == nil {
		t.Fatal("PacketInfoToProto(nil) error = nil, want non-nil")
	}
}

func TestMarshalPacketInfoRoundTrip(t *testing.T) {
	original := &model.PacketInfo{
		Timestamp: time.Unix(1700000010, 456),
		Length:    256,
		FiveTuple: model.FiveTuple{
			SrcIP:    net.ParseIP("198.51.100.1"),
			DstIP:    net.ParseIP("2001:db8::2"),
			SrcPort:  53000,
			DstPort:  8443,
			Protocol: 6,
		},
	}

	data, err := MarshalPacketInfo(nil, original)
	if err != nil {
		t.Fatalf("MarshalPacketInfo() unexpected error: %v", err)
	}

	decoded, err := UnmarshalPacketInfo(data)
	if err != nil {
		t.Fatalf("UnmarshalPacketInfo() unexpected error: %v", err)
	}

	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Fatalf("decoded timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}
	if !decoded.FiveTuple.SrcIP.Equal(original.FiveTuple.SrcIP) {
		t.Fatalf("decoded src ip = %v, want %v", decoded.FiveTuple.SrcIP, original.FiveTuple.SrcIP)
	}
	if !decoded.FiveTuple.DstIP.Equal(original.FiveTuple.DstIP) {
		t.Fatalf("decoded dst ip = %v, want %v", decoded.FiveTuple.DstIP, original.FiveTuple.DstIP)
	}
	if decoded.FiveTuple.SrcPort != original.FiveTuple.SrcPort {
		t.Fatalf("decoded src port = %d, want %d", decoded.FiveTuple.SrcPort, original.FiveTuple.SrcPort)
	}
	if decoded.FiveTuple.DstPort != original.FiveTuple.DstPort {
		t.Fatalf("decoded dst port = %d, want %d", decoded.FiveTuple.DstPort, original.FiveTuple.DstPort)
	}
	if decoded.FiveTuple.Protocol != original.FiveTuple.Protocol {
		t.Fatalf("decoded protocol = %d, want %d", decoded.FiveTuple.Protocol, original.FiveTuple.Protocol)
	}
	if decoded.Length != original.Length {
		t.Fatalf("decoded length = %d, want %d", decoded.Length, original.Length)
	}
}

func TestPacketInfoFromProtoRejectsMissingFiveTuple(t *testing.T) {
	if _, err := PacketInfoFromProto(nil); err == nil {
		t.Fatal("PacketInfoFromProto(nil) error = nil, want non-nil")
	}
}
