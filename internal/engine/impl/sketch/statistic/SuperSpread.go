package statistic

import (
	"bytes"
	"math"
	"math/rand"
	"sort"
)

const (
	hllDefaultM = 128
	hllDefaultSize = 5
	hllDefaultBase = 0.5
	ssDefaultB = 1.08
	ssDefaultThreshold = 4096
	ssDefaultWidth = 1 << 20
	ssDefaultDepth = 3
)

// GeneralHLL HyperLogLog Sampler
type GeneralHLL struct {
	m         uint32
	size      uint32    
	base      float64
	maxValue  uint8
	hll       []uint8  
	seeds     []uint32
	p         float64
}

// NewGeneralHLL
func NewGeneralHLL(m, size uint32, base float64) *GeneralHLL {
	hll := &GeneralHLL{
		m:        m,
		size:     size,
		base:     base,
		maxValue: (1 << size) - 1,
		hll:      make([]uint8, m),
		seeds:    make([]uint32, m+1),
		p:        1.0,
	}

	for i := range hll.seeds {
		hll.seeds[i] = rand.Uint32()
	}

	return hll
}

func leadingZeros32(x uint32) uint8 {
	if x == 0 {
		return 32
	}
	n := uint8(0)
	for (x & 0x80000000) == 0 {
		n++
		x <<= 1
	}
	return n
}

func (g *GeneralHLL) geometricHash(element []byte) uint8 {
	hash := MurmurHash3(element, g.seeds[0])
	v := min(leadingZeros32(hash) + 1, g.maxValue)
	return v
}

func (g *GeneralHLL) encode(element []byte) float64 {
    leadingZeros := g.geometricHash(element)

    hashVal := MurmurHash3(element, g.seeds[1])
	idx := hashVal % g.m
    original := g.hll[idx]

    if leadingZeros <= original {
        return -1.0
    }

	result := g.p
    g.p -= math.Pow(g.base, float64(original)) / float64(g.m)
    g.hll[idx] = leadingZeros
    if leadingZeros < g.maxValue {
        g.p += math.Pow(g.base, float64(leadingZeros)) / float64(g.m)
    }
    return result
}

// SuperSpread
type SuperSpread struct {
	d         uint32
	w         uint32
	threshold uint32
	cm        [][]*GeneralHLL
	keys      [][][]byte
	values    [][]uint32
	seeds     []uint32
	b         float64
}

// NewSuperSpread
func NewSuperSpread(width, depth, threshold, m, size uint32, base, b float64, FS uint32) *SuperSpread {

	if width == 0 {
		width = ssDefaultWidth
	}
	if depth == 0 {
		depth = ssDefaultDepth
	}
	if threshold == 0 {
		threshold = ssDefaultThreshold
	}
	if m == 0 {
		m = hllDefaultM
	}
	if size == 0 {
		size = hllDefaultSize
	}
	if base == 0 {
		base = hllDefaultBase
	}
	if b == 0 {
		b = ssDefaultB
	}

	ss := &SuperSpread{
		d:         depth,
		w:         width,
		threshold: threshold,
		cm:        make([][]*GeneralHLL, depth),
		keys:      make([][][]byte, depth),
		values:    make([][]uint32, depth),
		seeds:     make([]uint32, depth),
		b:         b,
	}

	for i := 0; i < int(depth); i++ {
		ss.cm[i] = make([]*GeneralHLL, width)
		ss.keys[i] = make([][]byte, width)
		ss.values[i] = make([]uint32, width)
		
		for j := 0; j < int(width); j++ {
			ss.cm[i][j] = NewGeneralHLL(m, size, base)
			ss.keys[i][j] = make([]byte, FS)
			ss.values[i][j] = 0
		}

		ss.seeds[i] = rand.Uint32()
	}

	return ss
}

// Implementation of SuperSpread insertion
func (ss *SuperSpread) Insert(flow, elem []byte, size uint32) {
	merged := append(flow, elem...)
	for i := 0; i < int(ss.d); i++ {
		j := MurmurHash3(flow, ss.seeds[i]) % ss.w

		tempP := ss.cm[i][j].encode(merged)
		if tempP == -1.0 {
			continue
		}

		pCU := 1.0 / tempP / math.Ceil(1.0/tempP)
		randFloatValue := rand.Float64()
		if randFloatValue < pCU {
			tempVV := int(math.Ceil(1.0 / tempP))
			for tempVV > 0 {
				tempVV--
				if ss.values[i][j] == 0 {
					copy(ss.keys[i][j], flow)
					ss.values[i][j] += 1
				} else if bytes.Equal(ss.keys[i][j], flow) {
					ss.values[i][j] += 1
				} else {
					ppp := math.Pow(ss.b, -float64(ss.values[i][j]))
					randFloatValue = rand.Float64()
					if randFloatValue < ppp {
						ss.values[i][j] -= 1
					}
				}
			}
		}
	}
}

// Implementation of SuperSpread query
func (ss *SuperSpread) Query(flow []byte) uint64 {
	estimate := uint32(0)
	for i := 0; i < int(ss.d); i++ {
		j := MurmurHash3(flow, ss.seeds[i]) % ss.w
		if bytes.Equal(ss.keys[i][j], flow) {
			if ss.values[i][j] > estimate {
				estimate = ss.values[i][j]
			}
		}
	}
	return uint64(math.Max(1, float64(estimate)))
}

// Implementation of SuperSpread HeavyHitters
// result reuse HeavyRecord.Count.Flow as the flow ID 
// and HeavyRecord.Count.Count as the estimated spread
func (ss *SuperSpread) HeavyHitters() HeavyRecord {
	flowSet := make(map[string]bool)
	results := make([]HeavyCount, 0)
	// record all unique flows
	for i := 0; i < int(ss.d); i++ {
		for j := 0; j < int(ss.w); j++ {
			if ss.values[i][j] > 0 {
				flowSet[string(ss.keys[i][j])] = true
			}
		}
	}
	// estimate each unique flow
	for flowID := range flowSet {
		estimate := uint32(0)
		flow := []byte(flowID)
		for i := 0; i < int(ss.d); i++ {
			j := MurmurHash3(flow, ss.seeds[i]) % ss.w
			if bytes.Equal(ss.keys[i][j], flow) {
				if ss.values[i][j] > estimate {
					estimate = ss.values[i][j]
				}
			}
		}
		if estimate >= ss.threshold {
			results = append(results, HeavyCount{
				Flow: flow,
				Count: estimate,
			})
		}
	}
	
	// sort by estimated spread in descending order
	sort.Slice(results, func(i, j int) bool {
		return results[i].Count > results[j].Count
	})
	
	return HeavyRecord{
		Count: results,
		Size:  nil,
	}
}

// Reset
func (ss *SuperSpread) Reset() {
	for i := 0; i < int(ss.d); i++ {
		for j := 0; j < int(ss.w); j++ {
			for k := range ss.cm[i][j].hll {
				ss.cm[i][j].hll[k] = 0
			}
			for k := range ss.keys[i][j] {
				ss.keys[i][j][k] = 0
			}
			ss.values[i][j] = 0
		}
	}
}