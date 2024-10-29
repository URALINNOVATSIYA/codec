package codec

import (
	"math/bits"
)

func id(v int) byte {
	return c2b(v)[0]
}

func c2b(v int) []byte {
	return u2bs(uint64(v), 4)
}

// u2bs (uint64 to bytes with size) returns the minimum byte representation of 
// v with byte size info in big endian
func u2bs(v uint64, sizeBits int) []byte {
	valueByteCount, totalByteCount := byteCount(v, sizeBits)
	if totalByteCount > valueByteCount {
		return append([]byte{byte(totalByteCount << (8 - sizeBits))}, u2b(v, valueByteCount)...)
	}
	return u2b(v|uint64(totalByteCount<<(8*totalByteCount-sizeBits)), totalByteCount)
}

func byteCount(v uint64, metaBitCount int) (valueByteCount int, totalByteCount int) {
	bitCount := bits.Len64(v)
	if bitCount == 0 {
		bitCount++
	}
	if bitCount&7 == 0 {
		valueByteCount = bitCount >> 3
	} else {
		valueByteCount = bitCount>>3 + 1
	}
	bitCount += metaBitCount
	if bitCount&7 == 0 {
		totalByteCount = bitCount >> 3
	} else {
		totalByteCount = bitCount>>3 + 1
	}
	return
}

// u2b (uint64 to bytes) returns the v's byte representation of the given size in big endian
func u2b(v uint64, size int) []byte {
	bytes := make([]byte, size)
	for i := 0; size > 0; i++ {
		size--
		bytes[i] = byte(v >> (size << 3))
	}
	return bytes
}

func b2u(bytes []byte) uint64 {
	var v uint64
	for i, size := 0, len(bytes); i < size; i++ {
		v = v<<8 | uint64(bytes[i])
	}
	return v
}

// i2u adjusts int64 value to store it as uint64 so that the sign bit becomes the first one
// and other sign bits are cleared
func i2u(i int64) uint64 {
	if i >= 0 {
		return uint64(i) << 1
	}
	return uint64(^i<<1) | 1
}

func u2i(i uint64) int64 {
	if i&1 != 0 { // negative
		return ^int64(i >> 1)
	}
	return int64(i >> 1)
}