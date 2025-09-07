package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"sync"
	"time"
)

// FlushedFlows represents the set of flows flushed from a single sub-aggregator.
type FlushedFlows struct {
	AggregatorName string
	Flows          []*model.Flow
}

// FlowAggregator is the main engine that manages multiple aggregation tasks.
type FlowAggregator struct {
	subAggregators []*KeyedAggregator
	InputChannel   chan *model.PacketInfo
	OutputChannel  chan FlushedFlows
	wg             sync.WaitGroup
	numWorkers     int
	flushInterval  time.Duration
	flowTimeout    time.Duration
}

// NewFlowAggregator creates a new FlowAggregator based on the provided configuration.
func NewFlowAggregator(cfg *config.Config, numWorkers int, flushInterval, flowTimeout time.Duration) *FlowAggregator {
	var subAggregators []*KeyedAggregator
	for _, task := range cfg.Aggregator.Tasks {
		agg := NewKeyedAggregator(task.Name, task.KeyFields)
		subAggregators = append(subAggregators, agg)
	}

	return &FlowAggregator{
		subAggregators: subAggregators,
		InputChannel:   make(chan *model.PacketInfo, 1000),
		OutputChannel:  make(chan FlushedFlows, 100), // Channel for exporting flushed flows
		numWorkers:     numWorkers,
		flushInterval:  flushInterval,
		flowTimeout:    flowTimeout,
	}
}

// Start launches the aggregator worker pool and the flushing ticker.
func (fa *FlowAggregator) Start() {
	// Start packet processing workers
	fa.wg.Add(fa.numWorkers)
	for i := 0; i < fa.numWorkers; i++ {
		go fa.worker()
	}

	// Start the flushing ticker
	fa.wg.Add(1)
	go fa.flusher()
}

// Stop waits for all workers and the flusher to finish.
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

// flusher periodically triggers a flush of inactive flows for all sub-aggregators.
func (fa *FlowAggregator) flusher() {
	defer fa.wg.Done()
	ticker := time.NewTicker(fa.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, subAgg := range fa.subAggregators {
				flushed := subAgg.FlushInactiveFlows(fa.flowTimeout)
				if len(flushed) > 0 {
					fa.OutputChannel <- FlushedFlows{
						AggregatorName: subAgg.Name,
						Flows:          flushed,
					}
				}
			}
		case _, ok := <-fa.InputChannel:
			if !ok {
				// Input channel is closed, flush one last time and exit.
				for _, subAgg := range fa.subAggregators {
					flushed := subAgg.FlushInactiveFlows(0) // Flush all remaining flows
					if len(flushed) > 0 {
						fa.OutputChannel <- FlushedFlows{
							AggregatorName: subAgg.Name,
							Flows:          flushed,
						}
					}
				}
				return
			}
		}
	}
}
