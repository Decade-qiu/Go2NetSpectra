package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"sync"
)

// FlowAggregator is the main engine that manages multiple aggregation tasks.
type FlowAggregator struct {
	subAggregators []*KeyedAggregator
	InputChannel   chan *model.PacketInfo
	wg             sync.WaitGroup
	numWorkers     int
}

// NewFlowAggregator creates a new FlowAggregator based on the provided configuration.
func NewFlowAggregator(cfg *config.Config, numWorkers int) *FlowAggregator {
	var subAggregators []*KeyedAggregator
	for _, task := range cfg.Aggregator.Tasks {
		agg := NewKeyedAggregator(task.Name, task.KeyFields)
		subAggregators = append(subAggregators, agg)
	}

	return &FlowAggregator{
		subAggregators: subAggregators,
		InputChannel:   make(chan *model.PacketInfo, 1000), // Buffered channel
		numWorkers:     numWorkers,
	}
}

// Start launches the aggregator worker pool.
func (fa *FlowAggregator) Start() {
	fa.wg.Add(fa.numWorkers)
	for i := 0; i < fa.numWorkers; i++ {
		go fa.worker()
	}
}

// Stop waits for all workers to finish.
func (fa *FlowAggregator) Stop() {
	close(fa.InputChannel)
	fa.wg.Wait()
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
