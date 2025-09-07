package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
)

// FlowAggregator is the main engine that manages multiple aggregation tasks.
type FlowAggregator struct {
	subAggregators []*KeyedAggregator
}

// NewFlowAggregator creates a new FlowAggregator based on the provided configuration.
func NewFlowAggregator(cfg *config.Config) *FlowAggregator {
	var subAggregators []*KeyedAggregator
	for _, task := range cfg.Aggregator.Tasks {
		agg := NewKeyedAggregator(task.Name, task.KeyFields)
		subAggregators = append(subAggregators, agg)
	}

	return &FlowAggregator{
		subAggregators: subAggregators,
	}
}

// ProcessPacket dispatches a single packet to all configured sub-aggregators.
func (fa *FlowAggregator) ProcessPacket(packetInfo *model.PacketInfo) {
	for _, subAgg := range fa.subAggregators {
		subAgg.ProcessPacket(packetInfo)
	}
}
