package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"fmt"
	"sync"
	"time"
)

// SnapshotData represents the full snapshot for a single sub-aggregator.
type SnapshotData struct {
	AggregatorName string
	Shards         []*Shard
}

// FlowAggregator is the main engine that manages multiple aggregation tasks.
type FlowAggregator struct {
	subAggregators   []*KeyedAggregator
	InputChannel     chan *model.PacketInfo
	OutputChannel    chan SnapshotData // Channel for exporting snapshots
	wg               sync.WaitGroup
	numWorkers       int
	snapshotInterval time.Duration
}

// NewFlowAggregator creates a new FlowAggregator based on the provided configuration.
func NewFlowAggregator(cfg *config.Config, numWorkers int, inputChanSize, outputChanSize int) (*FlowAggregator, error) {
	snapshotInterval, err := time.ParseDuration(cfg.Aggregator.SnapshotInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot_interval: %w", err)
	}

	var subAggregators []*KeyedAggregator
	for _, task := range cfg.Aggregator.Tasks {
		agg := NewKeyedAggregator(task.Name, task.KeyFields)
		subAggregators = append(subAggregators, agg)
	}

	return &FlowAggregator{
		subAggregators:   subAggregators,
		InputChannel:     make(chan *model.PacketInfo, inputChanSize),
		OutputChannel:    make(chan SnapshotData, outputChanSize),
		numWorkers:       numWorkers,
		snapshotInterval: snapshotInterval,
	}, nil
}

// Start launches the aggregator worker pool and the snapshotting ticker.
func (fa *FlowAggregator) Start() {
	// Start packet processing workers
	fa.wg.Add(fa.numWorkers)
	for i := 0; i < fa.numWorkers; i++ {
		go fa.worker()
	}

	// Start the snapshotting ticker
	fa.wg.Add(1)
	go fa.snapshotter()
}

// Stop waits for all workers and the snapshotter to finish.
func (fa *FlowAggregator) Stop() {
	close(fa.InputChannel)
	fa.wg.Wait()
	close(fa.OutputChannel)
}

// worker is a single worker that reads from the input channel and processes packets.
func (fa *FlowAggregator) worker() {
	defer fa.wg.Done()
	for packetInfo := range fa.InputChannel {
		for _, subAgg := range fa.subAggregators {
			subAgg.ProcessPacket(packetInfo)
		}
	}
}

// snapshotter periodically triggers a snapshot of all sub-aggregators.
func (fa *FlowAggregator) snapshotter() {
	defer fa.wg.Done()
	ticker := time.NewTicker(fa.snapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, subAgg := range fa.subAggregators {
				snapshot := subAgg.Snapshot()
				// We should check if the snapshot is empty
				hasFlows := false
				for _, shard := range snapshot {
					if len(shard.Flows) > 0 {
						hasFlows = true
						break
					}
				}
				if hasFlows {
					fa.OutputChannel <- SnapshotData{
						AggregatorName: subAgg.Name,
						Shards:         snapshot,
					}
				}
			}
		case _, ok := <-fa.InputChannel:
			if !ok {
				// Input channel is closed, do a final snapshot and exit.
				for _, subAgg := range fa.subAggregators {
					snapshot := subAgg.Snapshot()
					hasFlows := false
					for _, shard := range snapshot {
						if len(shard.Flows) > 0 {
							hasFlows = true
							break
						}
					}
					if hasFlows {
						fa.OutputChannel <- SnapshotData{
							AggregatorName: subAgg.Name,
							Shards:         snapshot,
						}
					}
				}
				return
			}
		}
	}
}
