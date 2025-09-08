package flowaggregator

import (
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/snapshot"
	"fmt"
	"log"
	"sync"
	"time"
)

// FlowAggregator is the main engine that manages multiple aggregation tasks.
type FlowAggregator struct {
	subAggregators   []*KeyedAggregator
	InputChannel     chan *model.PacketInfo
	wg               sync.WaitGroup
	numWorkers       int
	snapshotInterval time.Duration
	storageRootPath  string
	writer           *snapshot.Writer
}

// NewFlowAggregator creates a new FlowAggregator based on the provided configuration.
func NewFlowAggregator(cfg *config.Config) (*FlowAggregator, error) {
	snapshotInterval, err := time.ParseDuration(cfg.Aggregator.SnapshotInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot_interval: %w", err)
	}

	var subAggregators []*KeyedAggregator
	for _, task := range cfg.Aggregator.Tasks {
		agg := NewKeyedAggregator(task.Name, task.KeyFields, task.NumShards)
		subAggregators = append(subAggregators, agg)
	}

	return &FlowAggregator{
		subAggregators:   subAggregators,
		InputChannel:     make(chan *model.PacketInfo, cfg.Aggregator.SizeOfPacketChannel),
		numWorkers:       cfg.Aggregator.NumWorkers,
		snapshotInterval: snapshotInterval,
		storageRootPath:  cfg.Aggregator.StorageRootPath,
		writer:           snapshot.NewWriter(),
	}, nil
}

// Start launches the aggregator worker pool and the snapshotting ticker.
func (fa *FlowAggregator) Start() {
	fa.wg.Add(fa.numWorkers)
	for i := 0; i < fa.numWorkers; i++ {
		go fa.worker()
	}
	fa.wg.Add(1)
	go fa.snapshotter()
}

// Stop waits for all workers and the snapshotter to finish.
func (fa *FlowAggregator) Stop() {
	close(fa.InputChannel)
	fa.wg.Wait()
}

// worker processes packets from the input channel.
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
			fa.takeSnapshot()
			log.Println("Snapshot taken at ", time.Now())
		case _, ok := <-fa.InputChannel:
			if !ok {
				log.Println("Input channel closed, taking final snapshot...")
				fa.takeSnapshot()
				return
			}
		}
	}
}

// takeSnapshot orchestrates the process of taking and writing a snapshot.
func (fa *FlowAggregator) takeSnapshot() {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	log.Printf("Starting snapshot for timestamp %s...", timestamp)

	wg := sync.WaitGroup{}
	wg.Add(len(fa.subAggregators))
	for _, subAgg := range fa.subAggregators {
		go func ()  {
			defer wg.Done()
			snapshotData := subAgg.Snapshot()
			hasFlows := false
			for _, shard := range snapshotData {
				if len(shard.Flows) > 0 {
					hasFlows = true
					break
				}
			}

			if hasFlows {
				aggSnapshot := model.SnapshotData{
					AggregatorName: subAgg.Name,
					Shards:         snapshotData,
				}
				if err := fa.writer.Write(aggSnapshot, fa.storageRootPath, timestamp); err != nil {
					log.Printf("Error writing snapshot for %s: %v", subAgg.Name, err)
				}
			}
		}()
	}
	wg.Wait()
	log.Printf("Snapshot stored in path %s/%s", fa.storageRootPath, timestamp)
}
