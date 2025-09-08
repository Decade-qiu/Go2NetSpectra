package model

import (
	"net"
	"sync"
	"time"
)

// FiveTuple represents the 5-tuple of a network packet.
type FiveTuple struct {
	SrcIP    net.IP
	DstIP    net.IP
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8
}

// PacketInfo holds the metadata extracted from a single packet.
type PacketInfo struct {
	Timestamp time.Time
	FiveTuple FiveTuple
	Length    int
}

// Flow represents an aggregated flow of traffic. The definition of the flow
// is determined by the aggregator that produces it.
type Flow struct {
	// The value of the key(s) that defined this flow.
	// e.g., "1.2.3.4" if aggregated by SrcIP, or "1.2.3.4:80->2.3.4.5:443" for 5-tuple.
	Key         string
	StartTime   time.Time
	EndTime     time.Time
	ByteCount   uint64
	PacketCount uint64
}

// Shard is a part of a sharded map, containing its own map and a mutex.
// It is used in the snapshot data structure.
type Shard struct {
	Flows map[string]*Flow
	Mu    sync.RWMutex
}

// SnapshotData represents the full snapshot for a single sub-aggregator.
type SnapshotData struct {
	AggregatorName string
	Shards         []*Shard
}