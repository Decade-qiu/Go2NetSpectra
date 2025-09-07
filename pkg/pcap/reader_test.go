package pcap

import (
	"Go2NetSpectra/internal/core/model"
	"testing"
)

func TestReader_ReadPackets(t *testing.T) {
	// Our test pcap file contains a single TCP packet that should be parsed successfully.
	reader, err := NewReader("../../test/data/test.pcap")
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	out := make(chan *model.PacketInfo)

	go reader.ReadPackets(out)

	count := 0
	for range out {
		count++
	}

	expectedCount := 1
	if count != expectedCount {
		t.Errorf("Expected to read %d packet, but got %d", expectedCount, count)
	}
}
