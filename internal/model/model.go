package model

import (
	"net"
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