package exactaggregator

import (
	"sync"
	"time"
)

// Flow represents an aggregated flow of traffic with exact metrics.
type Flow struct {
	Key         string
	StartTime   time.Time
	EndTime     time.Time
	ByteCount   uint64
	PacketCount uint64
}

// Shard is a part of a sharded map, containing its own map and a mutex.
// It is used in the snapshot data structure for the ExactAggregator.
type Shard struct {
	Flows map[string]*Flow
	Mu    sync.RWMutex
}

// SnapshotData represents the full snapshot for a single sub-aggregator.
type SnapshotData struct {
	AggregatorName string
	Shards         []*Shard
}
