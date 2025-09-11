package statistic

import (
	"sync"
	"time"
)

// Flow represents an aggregated flow of traffic with exact metrics.
type Flow struct {
	Key         string
	Fields      map[string]interface{} // Holds the actual values for the fields that make up the key.
	StartTime   time.Time
	EndTime     time.Time
	ByteCount   uint64
	PacketCount uint64
}

// Shard is a part of a sharded map, containing its own map and a mutex.
type Shard struct {
	Flows map[string]*Flow
	Mu    sync.RWMutex
}

// SnapshotData represents the full snapshot for a single exact task.
// This is the data structure returned by the Snapshot() method.
type SnapshotData struct {
	TaskName string
	Shards   []*Shard
}