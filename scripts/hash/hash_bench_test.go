package main

import (
	"encoding/binary"
	"hash/crc32"
	"math/bits"
	"math/rand"
	"testing"
	"time"
)

//////////////////////
// 1. BobHash
//////////////////////

func BobHash(data []byte) uint32 {
	var (
		length = len(data)
		a      = uint32(0xdeadbeef + length)
		b      = a
		c      = a
	)

	rot := func(x uint32, k uint) uint32 {
		return (x << k) | (x >> (32 - k))
	}

	mix := func() {
		a -= c; a ^= rot(c, 4); c += b
		b -= a; b ^= rot(a, 6); a += c
		c -= b; c ^= rot(b, 8); b += a
		a -= c; a ^= rot(c, 16); c += b
		b -= a; b ^= rot(a, 19); a += c
		c -= b; c ^= rot(b, 4); b += a
	}

	final := func() {
		c ^= b; c -= rot(b, 14)
		a ^= c; a -= rot(c, 11)
		b ^= a; b -= rot(a, 25)
		c ^= b; c -= rot(b, 16)
		a ^= c; a -= rot(c, 4)
		b ^= a; b -= rot(a, 14)
		c ^= b; c -= rot(b, 24)
	}

	i := 0
	for length >= 12 {
		a += binary.LittleEndian.Uint32(data[i+0:])
		b += binary.LittleEndian.Uint32(data[i+4:])
		c += binary.LittleEndian.Uint32(data[i+8:])
		mix()
		i += 12
		length -= 12
	}

	switch length {
	case 11:
		c += uint32(data[i+10]) << 16
		fallthrough
	case 10:
		c += uint32(data[i+9]) << 8
		fallthrough
	case 9:
		c += uint32(data[i+8])
		fallthrough
	case 8:
		b += binary.LittleEndian.Uint32(data[i+4:])
		a += binary.LittleEndian.Uint32(data[i+0:])
	case 7:
		b += uint32(data[i+6]) << 16
		fallthrough
	case 6:
		b += uint32(data[i+5]) << 8
		fallthrough
	case 5:
		b += uint32(data[i+4])
		fallthrough
	case 4:
		a += binary.LittleEndian.Uint32(data[i+0:])
	case 3:
		a += uint32(data[i+2]) << 16
		fallthrough
	case 2:
		a += uint32(data[i+1]) << 8
		fallthrough
	case 1:
		a += uint32(data[i+0])
	}
	final()
	return c
}

//////////////////////
// 2. MurmurHash3 (32-bit)
//////////////////////

const (
	c1_32 uint32 = 0xcc9e2d51
	c2_32 uint32 = 0x1b873593
)

func MurmurHash3(data []byte, seed uint32) (h1 uint32) {
	h1 = seed
	clen := uint32(len(data))
	for len(data) >= 4 {
		k1 := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
		data = data[4:]

		k1 *= c1_32
		k1 = bits.RotateLeft32(k1, 15)
		k1 *= c2_32

		h1 ^= k1
		h1 = bits.RotateLeft32(h1, 13)
		h1 = h1*5 + 0xe6546b64
	}
	var k1 uint32
	switch len(data) {
	case 3:
		k1 ^= uint32(data[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(data[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(data[0])
		k1 *= c1_32
		k1 = bits.RotateLeft32(k1, 15)
		k1 *= c2_32
		h1 ^= k1
	}

	h1 ^= uint32(clen)

	h1 ^= h1 >> 16
	h1 *= 0x85ebca6b
	h1 ^= h1 >> 13
	h1 *= 0xc2b2ae35
	h1 ^= h1 >> 16

	return h1
}

//////////////////////
// 3. xxHash32 (更快)
//////////////////////

func xxHash32(data []byte, seed uint32) uint32 {
	const (
		prime1 = 2654435761
		prime2 = 2246822519
		prime3 = 3266489917
		prime4 = 668265263
		prime5 = 374761393
	)
	n := len(data)
	h := seed + prime5 + uint32(n)

	i := 0
	for n >= 4 {
		k := binary.LittleEndian.Uint32(data[i:])
		k *= prime3
		k = (k << 17) | (k >> 15)
		k *= prime4

		h ^= k
		h = (h << 17) | (h >> 15)
		h = h*prime1 + prime4

		i += 4
		n -= 4
	}

	for n > 0 {
		h ^= uint32(data[i]) * prime5
		h = (h << 11) | (h >> 21)
		h *= prime1
		i++
		n--
	}

	h ^= h >> 15
	h *= prime2
	h ^= h >> 13
	h *= prime3
	h ^= h >> 16

	return h
}

//////////////////////
// Benchmark
//////////////////////


var (
	data1MB  []byte
	data10MB []byte
	data100MB []byte
)

func init() {
	rand.Seed(time.Now().UnixNano())
	data1MB = make([]byte, 1024*1024)
	data10MB = make([]byte, 10*1024*1024)
	data100MB = make([]byte, 100*1024*1024)
	rand.Read(data1MB)
	rand.Read(data10MB)
	rand.Read(data100MB)
}

//////////////////////
// Benchmark 1KB
//////////////////////

func BenchmarkBobHash1KB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = BobHash(data1MB)
	}
}

func BenchmarkMurmurHash31KB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = MurmurHash3(data1MB, 0)
	}
}

func BenchmarkXXHash321KB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = xxHash32(data1MB, 0)
	}
}

func BenchmarkCRC321KB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = crc32.ChecksumIEEE(data1MB)
	}
}

//////////////////////
// Benchmark 1MB
//////////////////////

func BenchmarkBobHash1MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = BobHash(data10MB)
	}
}

func BenchmarkMurmurHash31MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = MurmurHash3(data10MB, 0)
	}
}

func BenchmarkXXHash321MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = xxHash32(data10MB, 0)
	}
}

func BenchmarkCRC321MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = crc32.ChecksumIEEE(data10MB)
	}
}

//

func BenchmarkBobHash100MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = BobHash(data100MB)
	}
}

func BenchmarkMurmurHash3100MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = MurmurHash3(data100MB, 0)
	}
}

func BenchmarkXXHash32100MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = xxHash32(data100MB, 0)
	}
}

func BenchmarkCRC32100MB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = crc32.ChecksumIEEE(data100MB)
	}
}