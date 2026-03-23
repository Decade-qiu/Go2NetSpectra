package streamaggregator

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/probe"

	"github.com/nats-io/nats.go"
)

func TestHandlePacketRoutesDecodedPacket(t *testing.T) {
	input := make(chan *model.PacketInfo, 1)
	aggregator := &StreamAggregator{inputChannel: input}

	expected := &model.PacketInfo{
		Timestamp: time.Unix(1700000020, 321),
		Length:    256,
		FiveTuple: model.FiveTuple{
			SrcIP:    net.ParseIP("192.0.2.10"),
			DstIP:    net.ParseIP("198.51.100.20"),
			SrcPort:  443,
			DstPort:  8443,
			Protocol: 6,
		},
	}

	payload, err := probe.MarshalPacketInfo(nil, expected)
	if err != nil {
		t.Fatalf("MarshalPacketInfo() unexpected error: %v", err)
	}

	aggregator.handlePacket(&nats.Msg{Data: payload})

	select {
	case got := <-input:
		if got == nil {
			t.Fatal("decoded packet = nil, want non-nil")
		}
		if !got.Timestamp.Equal(expected.Timestamp) {
			t.Fatalf("decoded timestamp = %v, want %v", got.Timestamp, expected.Timestamp)
		}
		if got.Length != expected.Length {
			t.Fatalf("decoded length = %d, want %d", got.Length, expected.Length)
		}
		if !got.FiveTuple.SrcIP.Equal(expected.FiveTuple.SrcIP) {
			t.Fatalf("decoded src ip = %v, want %v", got.FiveTuple.SrcIP, expected.FiveTuple.SrcIP)
		}
	case <-time.After(time.Second):
		t.Fatal("handlePacket() did not forward decoded packet")
	}
}

func TestHandlePacketRejectsLegacyProtobufPayload(t *testing.T) {
	input := make(chan *model.PacketInfo, 1)
	aggregator := &StreamAggregator{inputChannel: input}

	payload, err := hex.DecodeString("0a060880e2cfaa0612140a04c000020a1204c633641418bb0320fb412806188001")
	if err != nil {
		t.Fatalf("hex.DecodeString() unexpected error: %v", err)
	}

	aggregator.handlePacket(&nats.Msg{Data: payload})

	select {
	case got := <-input:
		t.Fatalf("forwarded packet = %v, want no packet", got)
	case <-time.After(100 * time.Millisecond):
	}
}
