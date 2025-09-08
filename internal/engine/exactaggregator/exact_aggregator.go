package exactaggregator

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"fmt"
	"log"
	"sync"
	"time"
)

// ExactAggregator is the main engine that manages multiple aggregation tasks.
type ExactAggregator struct {
	subAggregators   []*KeyedAggregator
	inputChannel     chan *model.PacketInfo // private
	wg               sync.WaitGroup
	numWorkers       int
	snapshotInterval time.Duration
	storageRootPath  string
	writer           model.Writer // Changed to interface type
}

// NewExactAggregator creates a new ExactAggregator that manages all "exact" aggregation tasks.
func NewExactAggregator(appCfg *config.Config) (*ExactAggregator, error) {
	snapshotInterval, err := time.ParseDuration(appCfg.Aggregator.SnapshotInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot_interval: %w", err)
	}

	var subAggregators []*KeyedAggregator
	for _, task := range appCfg.Aggregator.ExactTasks {
		agg := NewKeyedAggregator(task.Name, task.KeyFields, task.NumShards)
		subAggregators = append(subAggregators, agg)
	}

	return &ExactAggregator{
		subAggregators:   subAggregators,
		inputChannel:     make(chan *model.PacketInfo, appCfg.Aggregator.SizeOfPacketChannel),
		numWorkers:       appCfg.Aggregator.NumWorkers,
		snapshotInterval: snapshotInterval,
		storageRootPath:  appCfg.Aggregator.StorageRootPath,
		writer:           NewExactWriter(),
	}, nil
}

// Input returns the channel to which packets should be sent for processing.
func (fa *ExactAggregator) Input() chan<- *model.PacketInfo {
	return fa.inputChannel
}

// Start launches the aggregator worker pool and the snapshotting ticker.
func (fa *ExactAggregator) Start() {
	fa.wg.Add(fa.numWorkers)
	for i := 0; i < fa.numWorkers; i++ {
		go fa.worker()
	}
	fa.wg.Add(1)
	go fa.snapshotter()
}

// Stop waits for all workers and the snapshotter to finish.
func (fa *ExactAggregator) Stop() {
	close(fa.inputChannel)
	fa.wg.Wait()
}

// worker processes packets from the input channel.
func (fa *ExactAggregator) worker() {
	defer fa.wg.Done()
	for packetInfo := range fa.inputChannel {
		for _, subAgg := range fa.subAggregators {
			subAgg.ProcessPacket(packetInfo)
		}
	}
}

// snapshotter periodically triggers a snapshot of all sub-aggregators.
func (fa *ExactAggregator) snapshotter() {
	defer fa.wg.Done()
	ticker := time.NewTicker(fa.snapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fa.takeSnapshot()
			log.Println("Snapshot taken at ", time.Now())
		case _, ok := <-fa.inputChannel:
			if !ok {
				log.Println("Input channel closed, taking final snapshot...")
				fa.takeSnapshot()
				return
			}
		}
	}
}

// takeSnapshot orchestrates the process of taking and writing a snapshot.
func (fa *ExactAggregator) takeSnapshot() {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	log.Printf("Starting snapshot for timestamp %s... for %d sub-aggregators", timestamp, len(fa.subAggregators))

	wg := sync.WaitGroup{}
	wg.Add(len(fa.subAggregators))
	for _, subAgg := range fa.subAggregators {
		go func(subAgg *KeyedAggregator) { // Fixed loop variable capture
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
				aggSnapshot := SnapshotData{
					AggregatorName: subAgg.Name,
					Shards:         snapshotData,
				}
				if err := fa.writer.Write(aggSnapshot, fa.storageRootPath, timestamp); err != nil {
					log.Printf("Error writing snapshot for %s: %v", subAgg.Name, err)
				}
			}
		}(subAgg)
	}
	wg.Wait()
	log.Printf("Snapshot stored in path %s/%s", fa.storageRootPath, timestamp)
}
