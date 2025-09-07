package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"net"
	"runtime"
	"testing"
	"time"
)

func TestFlowAggregator_Flush(t *testing.T) {
	// 1. Load config
	cfg, err := config.LoadConfig("../../../configs/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 2. Create aggregator with short intervals for testing
	numWorkers := runtime.NumCPU()
	flushInterval := 50 * time.Millisecond
	flowTimeout := 100 * time.Millisecond
	aggregator := NewFlowAggregator(cfg, numWorkers, flushInterval, flowTimeout)

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

	// 5. Wait for the flow to be flushed
	var flushedFlows []FlushedFlows
	timeoutChan := time.After(500 * time.Millisecond)
	doneChan := make(chan bool)

	go func() {
		for f := range aggregator.OutputChannel {
			flushedFlows = append(flushedFlows, f)
			if len(flushedFlows) == 6 { // We expect 6 aggregators to flush one flow each
				doneChan <- true
				return
			}
		}
	}()

	select {
	case <-timeoutChan:
		t.Fatalf("Test timed out waiting for flushed flows. Got %d of 6.", len(flushedFlows))
	case <-doneChan:
		// Continue
	}

	// 6. Stop the aggregator
	aggregator.Stop()

	// 7. Verify results
	if len(flushedFlows) != 6 {
		t.Errorf("Expected 6 sets of flushed flows, but got %d", len(flushedFlows))
	}
	for _, f := range flushedFlows {
		if len(f.Flows) != 1 {
			t.Errorf("Expected 1 flow to be flushed from aggregator '%s', but got %d", f.AggregatorName, len(f.Flows))
		}
	}
}
