package statistic

// Sketch defines the interface for a sketch data structure.
// It supports insertion of elements, querying flow metrics, and retrieving top-k elements.
type Sketch interface {
	Insert(flow, elem []byte)
	Query(flow []byte) uint32
	HeavyHitters() []HeavyRecord
}

type HeavyRecord struct {
	Flow  []byte
	Count uint32
}