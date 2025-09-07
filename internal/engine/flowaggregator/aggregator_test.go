package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"net"
	"runtime"
	"testing"
	"time"
)

func TestFlowAggregator_Concurrent(t *testing.T) {
	// 1. Load config
	cfg, err := config.LoadConfig("../../../configs/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 2. Create aggregator with a number of workers equal to CPU cores
	numWorkers := runtime.NumCPU()
	aggregator := NewFlowAggregator(cfg, numWorkers)

	if len(aggregator.subAggregators) != 6 {
		t.Fatalf("Expected 6 sub-aggregators based on config, but got %d", len(aggregator.subAggregators))
	}

	// 3. Start the aggregator's worker pool
	aggregator.Start()

	// 4. Create a sample packet and send it
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

	aggregator.InputChannel <- packet

	// 5. Stop the aggregator, which waits for workers to finish processing
	aggregator.Stop()

	// 6. Inspect the sub-aggregators to verify the packet was processed
	for _, subAgg := range aggregator.subAggregators {
		if subAgg.GetFlowCount() != 1 {
			t.Errorf("Aggregator '%s' should have 1 active flow, but has %d", subAgg.Name, subAgg.GetFlowCount())
		}

		// Generate the expected key to verify the flow
		expectedKey, _ := subAgg.generateKey(packet.FiveTuple)
		if flow, ok := subAgg.GetFlow(expectedKey); ok {
			if flow.PacketCount != 1 {
				t.Errorf("Flow '%s' in aggregator '%s' should have PacketCount 1, got %d", expectedKey, subAgg.Name, flow.PacketCount)
			}
			if flow.ByteCount != 100 {
				t.Errorf("Flow '%s' in aggregator '%s' should have ByteCount 100, got %d", expectedKey, subAgg.Name, flow.ByteCount)
			}
		} else {
			t.Errorf("Aggregator '%s' did not contain expected flow with key '%s'", subAgg.Name, expectedKey)
		}
	}
}
