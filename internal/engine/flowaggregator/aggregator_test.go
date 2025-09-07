package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"net"
	"testing"
	"time"
)

func TestFlowAggregator(t *testing.T) {
	// 1. Load config
	cfg, err := config.LoadConfig("../../../configs/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 2. Create aggregator
	aggregator := NewFlowAggregator(cfg)

	if len(aggregator.subAggregators) != 6 {
		t.Fatalf("Expected 6 sub-aggregators based on config, but got %d", len(aggregator.subAggregators))
	}

	// 3. Create a sample packet
	packet := &model.PacketInfo{
		Timestamp: time.Now(),
		FiveTuple: model.FiveTuple{
			SrcIP:    net.ParseIP("192.168.0.1"),
			DstIP:    net.ParseIP("8.8.8.8"),
			SrcPort:  12345,
			DstPort:  53,
			Protocol: 17, // UDP
		},
		Length: 100,
	}

	// 4. Process the packet
	aggregator.ProcessPacket(packet)

	// 5. Inspect the sub-aggregators
	for _, subAgg := range aggregator.subAggregators {
		subAgg.mu.RLock()
		if len(subAgg.activeFlows) != 1 {
			t.Errorf("Aggregator '%s' should have 1 active flow, but has %d", subAgg.Name, len(subAgg.activeFlows))
		}
		// A more detailed test could check the actual key and flow values
		for key, flow := range subAgg.activeFlows {
			if flow.PacketCount != 1 {
				t.Errorf("Flow '%s' in aggregator '%s' should have PacketCount 1, got %d", key, subAgg.Name, flow.PacketCount)
			}
			if flow.ByteCount != 100 {
				t.Errorf("Flow '%s' in aggregator '%s' should have ByteCount 100, got %d", key, subAgg.Name, flow.ByteCount)
			}
		}
		subAgg.mu.RUnlock()
	}
}
