package statistic

import (
	"container/heap"
	"math/rand/v2"
)

const (
	defaultWidth = 1 << 20
	defaultDepth = 3
	defaultTopK  = 512
)

// MinHeap implements a min-heap for HeavyRecord
type MinHeap []HeavyRecord

func (h MinHeap) Len() int           { return len(h) }
func (h MinHeap) Less(i, j int) bool { return h[i].Count < h[j].Count } // Min-Heap
func (h MinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *MinHeap) Push(x any) {
	*h = append(*h, x.(HeavyRecord))
}

func (h *MinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type CountMin struct {
	w, d, k uint32
	seed    []uint32
	table   [][]uint32
	hh      MinHeap
}

func NewCountMin(width, depth, topk uint32) *CountMin {
	if width == 0 {
		width = defaultWidth
	}
	if depth == 0 {
		depth = defaultDepth
	}
	if topk == 0 {
		topk = defaultTopK
	}

	seed := make([]uint32, depth)
	for i := range seed {
		seed[i] = rand.Uint32()
	}

	table := make([][]uint32, depth)
	for i := range table {
		table[i] = make([]uint32, width)
	}

	h := make(MinHeap, 0, topk)
	heap.Init(&h)

	return &CountMin{
		w:      width,
		d:      depth,
		k:      topk,
		seed:   seed,
		table:  table,
		hh:     h,
	}
}

func (t *CountMin) Insert(flow, elem []byte) {
	val := uint32(0xFFFFFFFF)
	for i := 0; i < int(t.d); i++ {
		index := MurmurHash3(flow, t.seed[i]) % t.w
		t.table[i][index] += 1
		if t.table[i][index] < val {
			val = t.table[i][index]
		}
	}

	// Maintain the top-k heap
	if contains(t.hh, flow) {
		return
	}
	if len(t.hh) < int(t.k) {
		heap.Push(&t.hh, HeavyRecord{Flow: flow, Count: val})
	} else if val > t.hh[0].Count {
		// Replace the minimum element
		t.hh[0] = HeavyRecord{Flow: flow, Count: val}
		heap.Fix(&t.hh, 0)
	}
}

func contains(hh MinHeap, flow []byte) bool {
	for _, record := range hh {
		if string(record.Flow) == string(flow) {
			return true
		}
	}
	return false
}

// Implementation of Count-Min Sketch query
func (t *CountMin) Query(flow []byte) uint32 {
	min := uint32(0xFFFFFFFF)
	for i := 0; i < int(t.d); i++ {
		index := MurmurHash3(flow, t.seed[i]) % t.w
		if t.table[i][index] < min {
			min = t.table[i][index]
		}
	}
	return min
}

// Implementation of retrieving top-k elements
func (t *CountMin) Topk() []HeavyRecord {
	return t.hh
}