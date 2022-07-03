package codec

import "math/bits"

// minByteSize returns the minimum number of bytes required to represent v
func minByteSize(v uint64) int {
	if v == 0 {
		return 1
	}
	size := bits.Len64(v)
	if size&7 == 0 {
		return size >> 3
	}
	return size>>3 + 1
}

func minByteSizeWithMeta(v uint64, metaSize int) int {
	size := bits.Len64(v)
	if size == 0 {
		size++
	}
	size += metaSize
	if size&7 == 0 {
		return size >> 3
	}
	return size>>3 + 1
}

// toMinBytes returns the minimum number of bytes required to represent v
// and its byte representation in big endian
func toMinBytes(v uint64) (int, []byte) {
	size := minByteSize(v)
	bytes := make([]byte, size, size)
	for i, j := 0, size; j > 0; i++ {
		j--
		bytes[i] = byte(v >> (j << 3))
	}
	return size, bytes
}

// toMinBytes returns the minimum number of bytes required to represent v
func asMinBytes(v uint64) []byte {
	_, b := toMinBytes(v)
	return b
}

// toBytes returns the v's byte representation of the given size in big endian
func toBytes(v uint64, size int) []byte {
	bytes := make([]byte, size, size)
	for i := 0; size > 0; i++ {
		size--
		bytes[i] = byte(v >> (size << 3))
	}
	return bytes
}

func toUint(i int64) uint64 {
	if i >= 0 {
		return uint64(i) << 1
	}
	return uint64(^i<<1) | 1
}
