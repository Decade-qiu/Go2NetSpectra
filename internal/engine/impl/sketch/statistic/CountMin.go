package statistic

import (
	"bytes"
	"math/rand/v2"
	"slices"
)

const (
	defaultWidth      = 1 << 20
	defaultDepth      = 3
	defaultThereshold = 512
)

type Bucket struct {
	FP []byte
	C  uint32
}

type CountMin struct {
	w, d, thereshold uint32
	seed             []uint32
	table            [][]Bucket
}

func NewCountMin(width, depth, thereshold uint32, FS uint32) *CountMin {
	if width == 0 {
		width = defaultWidth
	}
	if depth == 0 {
		depth = defaultDepth
	}
	if thereshold == 0 {
		thereshold = defaultThereshold
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
				FP: make([]byte, FS),
				C:  0,
			}
		}
	}

	return &CountMin{
		w:          width,
		d:          depth,
		thereshold: thereshold,
		seed:       seed,
		table:      table,
	}
}

func (t *CountMin) Insert(flow, elem []byte) {
	for i := 0; i < int(t.d); i++ {
		index := MurmurHash3(flow, t.seed[i]) % t.w
		if t.table[i][index].C == 0 {
			copy(t.table[i][index].FP, flow)
			t.table[i][index].C = 1
		} else {
			if bytes.Equal(t.table[i][index].FP, flow) {
				t.table[i][index].C++
			} else {
				t.table[i][index].C--
				if t.table[i][index].C == 0 {
					copy(t.table[i][index].FP, flow)
					t.table[i][index].C = 1
				}
			}
		}
	}

}

// Implementation of Count-Min Sketch query
func (t *CountMin) Query(flow []byte) uint32 {
	sz := uint32(0)
	for i := 0; i < int(t.d); i++ {
		index := MurmurHash3(flow, t.seed[i]) % t.w
		if bytes.Equal(t.table[i][index].FP, flow) {
			sz = max(sz, t.table[i][index].C)
		}
	}
	return sz
}

// Implementation of retrieving top-k elements
func (t *CountMin) HeavyHitters() []HeavyRecord {
	hh := make(map[string]uint32)
	for i := 0; i < int(t.d); i++ {
		for j := 0; j < int(t.w); j++ {
			bucket := t.table[i][j]
			if bucket.C > 0 {
				key := string(bucket.FP)
				if count, exists := hh[key]; exists {
					hh[key] = max(count, bucket.C)
				} else {
					hh[key] = bucket.C
				}
			}
		}
	}
	heavyHitters := make([]HeavyRecord, 0)
	for k, v := range hh {
		if v < t.thereshold {
			continue
		}
		heavyHitters = append(heavyHitters, HeavyRecord{
			Flow:  []byte(k),
			Count: v,
		})
	}
	// Sort heavy hitters by count in descending order
	slices.SortFunc(heavyHitters, func(a, b HeavyRecord) int {
		return int(b.Count) - int(a.Count)
	})
	return heavyHitters
}
