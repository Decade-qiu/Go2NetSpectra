package statistic

// Sketch defines the interface for a sketch data structure.
// It supports insertion of elements, querying flow metrics, and retrieving top-k elements.
type Sketch interface {
	Insert(flow, elem []byte, size uint32)
	Query(flow []byte) uint64
	HeavyHitters() HeavyRecord
}

type HeavySize struct {
	Flow  []byte
	Size  uint32
}

type HeavyCount struct {
	Flow  []byte
	Count uint32
}

type HeavyRecord struct {
	Size  []HeavySize
	Count []HeavyCount
}