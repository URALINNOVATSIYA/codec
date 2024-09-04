package codec

import (
	"math/bits"
)

func minByteCount(v uint64, metaBitCount int) (valueByteCount int, totalByteCount int) {
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

// asBytesWithMeta returns the minimum byte representation of v with byte size info in big endian
func asBytesWithSize(v uint64, sizeBits int) []byte {
	valueByteCount, totalByteCount := minByteCount(v, sizeBits)
	if totalByteCount > valueByteCount {
		return append([]byte{byte(totalByteCount << (8 - sizeBits))}, asBytes(v, valueByteCount)...)
	}
	return asBytes(v | uint64(totalByteCount << (8 * totalByteCount - sizeBits)), totalByteCount)
}

func asMinBytes(v uint64) []byte {
	_, byteCount := minByteCount(v, 0)
	return asBytes(v, byteCount)
}

// asBytes returns the v's byte representation of the given size in big endian
func asBytes(v uint64, size int) []byte {
	bytes := make([]byte, size)
	for i := 0; size > 0; i++ {
		size--
		bytes[i] = byte(v >> (size << 3))
	}
	return bytes
}

// toUint adjusts int64 value to store it as uint64 so that the sign bit becomes the first one
// and other sign bits are cleared
func toUint(i int64) uint64 {
	if i >= 0 {
		return uint64(i) << 1
	}
	return uint64(^i<<1) | 1
}