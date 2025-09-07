package model

import (
	"net"
	"time"
)

// FiveTuple represents the 5-tuple of a network flow.
type FiveTuple struct {
	SrcIP    net.IP
	DstIP    net.IP
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8 // e.g., TCP, UDP
}

// PacketInfo holds the metadata extracted from a single packet.
type PacketInfo struct {
	Timestamp time.Time
	FiveTuple FiveTuple
	Length    int
}

// Flow represents a network flow.
type Flow struct {
	ID        string // A unique identifier for the flow, e.g., hash of 5-tuple
	FiveTuple FiveTuple
	Packets   []*PacketInfo
	StartTime time.Time
	EndTime   time.Time
	ByteCount int
	PacketCount int
}
