package statistic

// Sketch defines the interface for a sketch data structure.
// It supports insertion of elements, querying flow metrics, and retrieving top-k elements.
type Sketch interface {
	Insert(flow, elem []byte, size uint32)
	Query(flow []byte) uint64
	HeavyHitters() HeavyRecord
	Reset()
}

// HeavySize stores a heavy-hitter flow and its estimated byte size.
type HeavySize struct {
	Flow []byte
	Size uint32
}

// HeavyCount stores a heavy-hitter flow and its estimated packet count.
type HeavyCount struct {
	Flow  []byte
	Count uint32
}

// HeavyRecord groups heavy-hitter results by size and count metrics.
type HeavyRecord struct {
	Size  []HeavySize
	Count []HeavyCount
}
