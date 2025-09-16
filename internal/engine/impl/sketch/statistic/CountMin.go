package statistic

import (
	"bytes"
	"math/rand/v2"
	"slices"
)

const (
	defaultWidth           = 1 << 20
	defaultDepth           = 3
	defaultSizeThereshold  = 512 * 1024
	defaultCountThereshold = 512
)

type PacketCount struct {
	FP []byte
	C  uint32
}

type PacketSize struct {
	FP []byte
	S  uint32
}

type Bucket struct {
	Count PacketCount
	Size  PacketSize
}

type CountMin struct {
	w, d            uint32
	sizeThereshold  uint32
	countThereshold uint32
	seed            []uint32
	table           [][]Bucket
}

func NewCountMin(width, depth, st, ct uint32, FS uint32) *CountMin {
	if width == 0 {
		width = defaultWidth
	}
	if depth == 0 {
		depth = defaultDepth
	}
	if st == 0 {
		st = defaultSizeThereshold
	}
	if ct == 0 {
		ct = defaultCountThereshold
	}

	seed := make([]uint32, depth)
	for i := range seed {
		seed[i] = rand.Uint32()
	}

	table := make([][]Bucket, depth)
	for i := range table {
		table[i] = make([]Bucket, width)
		for j := range table[i] {
			table[i][j] = Bucket{
				Count: PacketCount{
					FP: make([]byte, FS),
					C:  0,
				},
				Size: PacketSize{
					FP: make([]byte, FS),
					S:  0,
				},
			}
		}
	}

	return &CountMin{
		w:               width,
		d:               depth,
		sizeThereshold:  st,
		countThereshold: ct,
		seed:            seed,
		table:           table,
	}
}

// Implementation of Count-Min Sketch insertion
func (t *CountMin) Insert(flow, elem []byte, size uint32) {
	for i := 0; i < int(t.d); i++ {
		index := MurmurHash3(flow, t.seed[i]) % t.w
		bucket := &t.table[i][index]
		// Update Packet Size
		bsize := &bucket.Size
		if bsize.S == 0 {
			copy(bsize.FP, flow)
			bsize.S = size
		} else {
			if bytes.Equal(bsize.FP, flow) {
				bsize.S += size
			} else {
				if size > bsize.S {
					copy(bsize.FP, flow)
					bsize.S = size
				} else {
					bsize.S -= size
				}
			}
		}
		// Update Packet Count
		bcount := &bucket.Count
		if bcount.C == 0 {
			copy(bcount.FP, flow)
			bcount.C = 1
		} else {
			if bytes.Equal(bcount.FP, flow) {
				bcount.C++
			} else {
				bcount.C--
				if bcount.C == 0 {
					copy(bcount.FP, flow)
					bcount.C = 1
				}
			}
		}
	}

}

// Implementation of Count-Min Sketch query
func (t *CountMin) Query(flow []byte) uint64 {
	sz, ct := uint32(0), uint32(0)
	for i := 0; i < int(t.d); i++ {
		index := MurmurHash3(flow, t.seed[i]) % t.w
		bucket := &t.table[i][index]
		if bytes.Equal(bucket.Size.FP, flow) {
			sz = max(sz, bucket.Size.S)
		}
		if bytes.Equal(bucket.Count.FP, flow) {
			ct = max(ct, bucket.Count.C)
		}
	}
	return uint64(ct)<<32 | uint64(sz)
}

// HeavyHitters returns the heavy hitters for both Size and Count
// It scans the CountMin table, finds flows exceeding the threshold,
// and returns them sorted in descending order.
func (t *CountMin) HeavyHitters() HeavyRecord {
	// Temporary maps to track the maximum Size and Count per flow
	sizeMap := make(map[string]uint32)
	countMap := make(map[string]uint32)

	for i := 0; i < int(t.d); i++ {
		for j := 0; j < int(t.w); j++ {
			bucket := t.table[i][j]

			// Update Size map if the bucket has a positive size
			if bucket.Size.S > 0 {
				key := string(bucket.Size.FP)
				if cur, exists := sizeMap[key]; exists {
					sizeMap[key] = max(cur, bucket.Size.S)
				} else {
					sizeMap[key] = bucket.Size.S
				}
			}

			// Update Count map if the bucket has a positive count
			if bucket.Count.C > 0 {
				key := string(bucket.Count.FP)
				if cur, exists := countMap[key]; exists {
					countMap[key] = max(cur, bucket.Count.C)
				} else {
					countMap[key] = bucket.Count.C
				}
			}
		}
	}

	// Construct HeavySize list for flows whose Size >= threshold
	heavySizes := make([]HeavySize, 0)
	for k, sz := range sizeMap {
		if sz >= t.sizeThereshold {
			heavySizes = append(heavySizes, HeavySize{
				Flow: []byte(k),
				Size: sz,
			})
		}
	}

	// Construct HeavyCount list for flows whose Count >= threshold
	heavyCounts := make([]HeavyCount, 0)
	for k, ct := range countMap {
		if ct >= t.countThereshold {
			heavyCounts = append(heavyCounts, HeavyCount{
				Flow:  []byte(k),
				Count: ct,
			})
		}
	}

	// Sort HeavySize list in descending order by Size
	slices.SortFunc(heavySizes, func(a, b HeavySize) int {
		return int(b.Size) - int(a.Size)
	})

	// Sort HeavyCount list in descending order by Count
	slices.SortFunc(heavyCounts, func(a, b HeavyCount) int {
		return int(b.Count) - int(a.Count)
	})

	// Return combined HeavyRecord
	return HeavyRecord{
		Size:  heavySizes,
		Count: heavyCounts,
	}
}
