package statistic

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestMurmurHash3Uniformity(t *testing.T) {
	const (
		numKeys    = 100_000_000
		numBuckets = 1 << 10
		seed       = 17371
	)

	buckets := make([]int, numBuckets)

	for i := 0; i < numKeys; i++ {
		key := make([]byte, 4)
		binary.LittleEndian.PutUint32(key, rand.Uint32()) // 随机整数
		hash := MurmurHash3(key, seed)
		idx := hash % numBuckets
		buckets[idx]++
	}	

	// 统计平均值
	sum := 0
	for _, cnt := range buckets {
		sum += cnt
	}
	avg := float64(sum) / float64(numBuckets)

	// 计算标准差
	var variance float64
	for _, cnt := range buckets {
		diff := float64(cnt) - avg
		variance += diff * diff
	}
	std := (variance / float64(numBuckets))
	cv := std / avg // coefficient of variation

	fmt.Printf("avg = %.2f, std = %.2f, CV = %.4f\n", avg, std, cv)
}

