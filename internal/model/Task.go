package model

// Task defines a single, self-contained aggregation task (e.g., exact count, sketch, etc.).
// This is the interface for the "execution layer".
type Task interface {
	ProcessPacket(packet *PacketInfo)
	Snapshot() interface{}
	Reset()
	Name() string
}