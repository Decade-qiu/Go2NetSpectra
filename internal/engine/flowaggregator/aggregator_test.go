package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"net"
	"runtime"
	"testing"
	"time"
)

func TestFlowAggregator_Snapshot(t *testing.T) {
	// 1. Load config
	cfg, err := config.LoadConfig("../../../configs/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	// Use a short interval for testing
	cfg.Aggregator.SnapshotInterval = "50ms"

	// 2. Create aggregator
	numWorkers := runtime.NumCPU()
	aggregator, err := NewFlowAggregator(cfg, numWorkers)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	// 3. Start the aggregator
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

	// 5. Wait for the snapshot to be received on the output channel
	var snapshots []SnapshotData
	timeoutChan := time.After(500 * time.Millisecond)
	doneChan := make(chan bool)

	go func() {
		for s := range aggregator.OutputChannel {
			snapshots = append(snapshots, s)
			if len(snapshots) == 6 { // We expect 6 aggregators to produce a snapshot
				doneChan <- true
				return
			}
		}
	}()

	select {
	case <-timeoutChan:
		t.Fatalf("Test timed out waiting for snapshots. Got %d of 6.", len(snapshots))
	case <-doneChan:
		// Continue
	}

	// 6. Stop the aggregator
	aggregator.Stop()

	// 7. Verify results
	if len(snapshots) != 6 {
		t.Errorf("Expected 6 snapshots, but got %d", len(snapshots))
	}
	for _, s := range snapshots {
        totalFlows := 0
        for _, shard := range s.Shards {
            totalFlows += len(shard.Flows)
        }
		if totalFlows != 1 {
			t.Errorf("Expected 1 flow in snapshot from aggregator '%s', but got %d", s.AggregatorName, totalFlows)
		}
	}
}
