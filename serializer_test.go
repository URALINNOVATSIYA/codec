package codec

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

type serializerTestArgs struct {
	value any
	data  []byte
}

func TestBasicSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		// Nil
		{
			nil,
			[]byte{version, tNil},
		},
		// Bool
		{
			false,
			[]byte{version, tBool},
		},
		{
			true,
			[]byte{version, tBool | tru},
		},
		// Uint
		{
			uint(0),
			[]byte{version, tInt, 0},
		},
		{
			uint(1),
			[]byte{version, tInt, 1},
		},
		{
			uint(255),
			[]byte{version, tInt, 255},
		},
		{
			uint(256),
			[]byte{version, tInt | 0b0001, 1, 0},
		},
		{
			uint(256<<8 - 1),
			[]byte{version, tInt | 0b0001, 255, 255},
		},
		{
			uint(256 << 8),
			[]byte{version, tInt | 0b0010, 1, 0, 0},
		},
		{
			uint(256<<16 - 1),
			[]byte{version, tInt | 0b0010, 255, 255, 255},
		},
		{
			uint(256 << 16),
			[]byte{version, tInt | 0b0011, 1, 0, 0, 0},
		},
		{
			uint(256<<24 - 1),
			[]byte{version, tInt | 0b0011, 255, 255, 255, 255},
		},
		{
			uint(256 << 24),
			[]byte{version, tInt | 0b0100, 1, 0, 0, 0, 0},
		},
		{
			uint(256<<32 - 1),
			[]byte{version, tInt | 0b0100, 255, 255, 255, 255, 255},
		},
		{
			uint(256 << 32),
			[]byte{version, tInt | 0b0101, 1, 0, 0, 0, 0, 0},
		},
		{
			uint(256<<40 - 1),
			[]byte{version, tInt | 0b0101, 255, 255, 255, 255, 255, 255},
		},
		{
			uint(256 << 40),
			[]byte{version, tInt | 0b0110, 1, 0, 0, 0, 0, 0, 0},
		},
		{
			uint(256<<48 - 1),
			[]byte{version, tInt | 0b0110, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			uint(256 << 48),
			[]byte{version, tInt | 0b0111, 1, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			uint(math.MaxUint),
			[]byte{version, tInt | 0b0111, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		// Int
		{
			0,
			[]byte{version, tInt | signed, 0},
		},
		{
			127,
			[]byte{version, tInt | signed, 254},
		},
		{
			-128,
			[]byte{version, tInt | signed, 255},
		},
		{
			128,
			[]byte{version, tInt | signed | 0b0001, 1, 0},
		},
		{
			-129,
			[]byte{version, tInt | signed | 0b0001, 1, 1},
		},
		{
			128<<8 - 1,
			[]byte{version, tInt | signed | 0b0001, 255, 254},
		},
		{
			-128 << 8,
			[]byte{version, tInt | signed | 0b0001, 255, 255},
		},
		{
			128 << 8,
			[]byte{version, tInt | signed | 0b0010, 1, 0, 0},
		},
		{
			-128<<8 - 1,
			[]byte{version, tInt | signed | 0b0010, 1, 0, 1},
		},
		{
			128<<16 - 1,
			[]byte{version, tInt | signed | 0b0010, 255, 255, 254},
		},
		{
			-128 << 16,
			[]byte{version, tInt | signed | 0b0010, 255, 255, 255},
		},
		{
			128 << 16,
			[]byte{version, tInt | signed | 0b0011, 1, 0, 0, 0},
		},
		{
			-128<<16 - 1,
			[]byte{version, tInt | signed | 0b0011, 1, 0, 0, 1},
		},
		{
			128<<24 - 1,
			[]byte{version, tInt | signed | 0b0011, 255, 255, 255, 254},
		},
		{
			-128 << 24,
			[]byte{version, tInt | signed | 0b0011, 255, 255, 255, 255},
		},
		{
			128 << 24,
			[]byte{version, tInt | signed | 0b0100, 1, 0, 0, 0, 0},
		},
		{
			-128<<24 - 1,
			[]byte{version, tInt | signed | 0b0100, 1, 0, 0, 0, 1},
		},
		{
			128<<32 - 1,
			[]byte{version, tInt | signed | 0b0100, 255, 255, 255, 255, 254},
		},
		{
			-128 << 32,
			[]byte{version, tInt | signed | 0b0100, 255, 255, 255, 255, 255},
		},
		{
			128 << 32,
			[]byte{version, tInt | signed | 0b0101, 1, 0, 0, 0, 0, 0},
		},
		{
			-128<<32 - 1,
			[]byte{version, tInt | signed | 0b0101, 1, 0, 0, 0, 0, 1},
		},
		{
			128<<40 - 1,
			[]byte{version, tInt | signed | 0b0101, 255, 255, 255, 255, 255, 254},
		},
		{
			-128 << 40,
			[]byte{version, tInt | signed | 0b0101, 255, 255, 255, 255, 255, 255},
		},
		{
			128 << 40,
			[]byte{version, tInt | signed | 0b0110, 1, 0, 0, 0, 0, 0, 0},
		},
		{
			-128<<40 - 1,
			[]byte{version, tInt | signed | 0b0110, 1, 0, 0, 0, 0, 0, 1},
		},
		{
			128<<48 - 1,
			[]byte{version, tInt | signed | 0b0110, 255, 255, 255, 255, 255, 255, 254},
		},
		{
			-128 << 48,
			[]byte{version, tInt | signed | 0b0110, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			128 << 48,
			[]byte{version, tInt | signed | 0b0111, 1, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			-128<<48 - 1,
			[]byte{version, tInt | signed | 0b0111, 1, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			math.MaxInt,
			[]byte{version, tInt | signed | 0b0111, 255, 255, 255, 255, 255, 255, 255, 254},
		},
		{
			math.MinInt,
			[]byte{version, tInt | signed | 0b0111, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		// Uint8, byte
		{
			uint8(0),
			[]byte{version, tInt8, 0},
		},
		{
			uint8(1),
			[]byte{version, tInt8, 1},
		},
		{
			uint8(255),
			[]byte{version, tInt8, 255},
		},
		// Int8
		{
			int8(0),
			[]byte{version, tInt8 | signed, 0},
		},
		{
			int8(1),
			[]byte{version, tInt8 | signed, 2},
		},
		{
			int8(-1),
			[]byte{version, tInt8 | signed, 1},
		},
		{
			int8(127),
			[]byte{version, tInt8 | signed, 254},
		},
		{
			int8(-128),
			[]byte{version, tInt8 | signed, 255},
		},
		// Uint16
		{
			uint16(0),
			[]byte{version, tInt16 | meta, 0},
		},
		{
			uint16(1),
			[]byte{version, tInt16 | meta, 1},
		},
		{
			uint16(256),
			[]byte{version, tInt16, 1, 0},
		},
		{
			uint16(65535),
			[]byte{version, tInt16, 255, 255},
		},
		// Int16
		{
			int16(0),
			[]byte{version, tInt16 | signed | meta, 0},
		},
		{
			int16(1),
			[]byte{version, tInt16 | signed | meta, 2},
		},
		{
			int16(-1),
			[]byte{version, tInt16 | signed | meta, 1},
		},
		{
			int16(256),
			[]byte{version, tInt16 | signed, 2, 0},
		},
		{
			int16(-256),
			[]byte{version, tInt16 | signed, 1, 255},
		},
		{
			int16(32767),
			[]byte{version, tInt16 | signed, 255, 254},
		},
		{
			int16(-32768),
			[]byte{version, tInt16 | signed, 255, 255},
		},
		// Uint32
		{
			uint32(0),
			[]byte{version, tInt32 | meta, 0 | 0b01_000000},
		},
		{
			uint32(1),
			[]byte{version, tInt32 | meta, 1 | 0b01_000000},
		},
		{
			uint32(256),
			[]byte{version, tInt32 | meta, 1 | 0b10_000000, 0},
		},
		{
			uint32(123456),
			[]byte{version, tInt32 | meta, 1 | 0b11_000000, 226, 64},
		},
		{
			uint32(math.MaxUint32),
			[]byte{version, tInt32, 255, 255, 255, 255},
		},
		// Int32
		{
			int32(0),
			[]byte{version, tInt32 | signed | meta, 0 | 0b01_000000},
		},
		{
			int32(1),
			[]byte{version, tInt32 | signed | meta, 2 | 0b01_000000},
		},
		{
			int32(-1),
			[]byte{version, tInt32 | signed | meta, 1 | 0b01_000000},
		},
		{
			int32(256),
			[]byte{version, tInt32 | signed | meta, 2 | 0b10_000000, 0},
		},
		{
			int32(-256),
			[]byte{version, tInt32 | signed | meta, 1 | 0b10_000000, 255},
		},
		{
			int32(123456),
			[]byte{version, tInt32 | signed | meta, 3 | 0b11_000000, 196, 128},
		},
		{
			int32(-123456),
			[]byte{version, tInt32 | signed | meta, 3 | 0b11_000000, 196, 127},
		},
		{
			int32(math.MaxInt32),
			[]byte{version, tInt32 | signed, 255, 255, 255, 254},
		},
		{
			int32(math.MinInt32),
			[]byte{version, tInt32 | signed, 255, 255, 255, 255},
		},
		// Uint64
		{
			uint64(0),
			[]byte{version, tInt64 | meta, 0 | 0b001_00000},
		},
		{
			uint64(1),
			[]byte{version, tInt64 | meta, 1 | 0b001_00000},
		},
		{
			uint64(256),
			[]byte{version, tInt64 | meta, 1 | 0b010_00000, 0},
		},
		{
			uint64(256 << 8),
			[]byte{version, tInt64 | meta, 1 | 0b011_00000, 0, 0},
		},
		{
			uint64(256 << 16),
			[]byte{version, tInt64 | meta, 1 | 0b100_00000, 0, 0, 0},
		},
		{
			uint64(256 << 24),
			[]byte{version, tInt64 | meta, 1 | 0b101_00000, 0, 0, 0, 0},
		},
		{
			uint64(256 << 32),
			[]byte{version, tInt64 | meta, 1 | 0b110_00000, 0, 0, 0, 0, 0},
		},
		{
			uint64(256 << 40),
			[]byte{version, tInt64 | meta, 1 | 0b111_00000, 0, 0, 0, 0, 0, 0},
		},
		{
			uint64(math.MaxUint64),
			[]byte{version, tInt64, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		// Int64
		{
			int64(0),
			[]byte{version, tInt64 | signed | meta, 0 | 0b001_00000},
		},
		{
			int64(1),
			[]byte{version, tInt64 | signed | meta, 2 | 0b001_00000},
		},
		{
			int64(-1),
			[]byte{version, tInt64 | signed | meta, 1 | 0b001_00000},
		},
		{
			int64(256),
			[]byte{version, tInt64 | signed | meta, 2 | 0b010_00000, 0},
		},
		{
			int64(-256),
			[]byte{version, tInt64 | signed | meta, 1 | 0b010_00000, 255},
		},
		{
			int64(256 << 8),
			[]byte{version, tInt64 | signed | meta, 2 | 0b011_00000, 0, 0},
		},
		{
			int64(-(256 << 8)),
			[]byte{version, tInt64 | signed | meta, 1 | 0b011_00000, 255, 255},
		},
		{
			int64(256 << 16),
			[]byte{version, tInt64 | signed | meta, 2 | 0b100_00000, 0, 0, 0},
		},
		{
			int64(-(256 << 16)),
			[]byte{version, tInt64 | signed | meta, 1 | 0b100_00000, 255, 255, 255},
		},
		{
			int64(256 << 24),
			[]byte{version, tInt64 | signed | meta, 2 | 0b101_00000, 0, 0, 0, 0},
		},
		{
			int64(-(256 << 24)),
			[]byte{version, tInt64 | signed | meta, 1 | 0b101_00000, 255, 255, 255, 255},
		},
		{
			int64(256 << 32),
			[]byte{version, tInt64 | signed | meta, 2 | 0b110_00000, 0, 0, 0, 0, 0},
		},
		{
			int64(-(256 << 32)),
			[]byte{version, tInt64 | signed | meta, 1 | 0b110_00000, 255, 255, 255, 255, 255},
		},
		{
			int64(256 << 40),
			[]byte{version, tInt64 | signed | meta, 2 | 0b111_00000, 0, 0, 0, 0, 0, 0},
		},
		{
			int64(-(256 << 40)),
			[]byte{version, tInt64 | signed | meta, 1 | 0b111_00000, 255, 255, 255, 255, 255, 255},
		},
		{
			int64(math.MaxInt64),
			[]byte{version, tInt64 | signed, 255, 255, 255, 255, 255, 255, 255, 254},
		},
		{
			int64(math.MinInt64),
			[]byte{version, tInt64 | signed, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		// Float32
		{
			float32(0),
			[]byte{version, tFloat, 0},
		},
		{
			float32(1),
			[]byte{version, tFloat | 0b0001, 128, 63},
		},
		{
			float32(10),
			[]byte{version, tFloat | 0b0001, 32, 65},
		},
		{
			float32(-1),
			[]byte{version, tFloat | 0b0001, 128, 191},
		},
		{
			float32(-10),
			[]byte{version, tFloat | 0b0001, 32, 193},
		},
		{
			float32(1.23),
			[]byte{version, tFloat | 0b0011, 164, 112, 157, 63},
		},
		{
			float32(-1.23),
			[]byte{version, tFloat | 0b0011, 164, 112, 157, 191},
		},
		// Float64
		{
			float64(0),
			[]byte{version, tFloat | wide, 0},
		},
		{
			float64(1),
			[]byte{version, tFloat | wide | 0b0001, 240, 63},
		},
		{
			float64(10),
			[]byte{version, tFloat | wide | 0b0001, 36, 64},
		},
		{
			float64(-1),
			[]byte{version, tFloat | wide | 0b0001, 240, 191},
		},
		{
			float64(-10),
			[]byte{version, tFloat | wide | 0b0001, 36, 192},
		},
		{
			1.23,
			[]byte{version, tFloat | wide | 0b0111, 174, 71, 225, 122, 20, 174, 243, 63},
		},
		{
			-1.23,
			[]byte{version, tFloat | wide | 0b0111, 174, 71, 225, 122, 20, 174, 243, 191},
		},
		// Complex64
		{
			complex(float32(0), float32(0)),
			[]byte{version, tComplex, tFloat, 0, tFloat, 0},
		},
		{
			complex(float32(1), float32(0)),
			[]byte{version, tComplex, tFloat | 0b0001, 128, 63, tFloat, 0},
		},
		{
			complex(float32(0), float32(1)),
			[]byte{version, tComplex, tFloat, 0, tFloat | 0b0001, 128, 63},
		},
		{
			complex(float32(1.23), float32(-1.23)),
			[]byte{version, tComplex, tFloat | 0b0011, 164, 112, 157, 63, tFloat | 0b0011, 164, 112, 157, 191},
		},
		// Complex128
		{
			complex(float64(0), float64(0)),
			[]byte{version, tComplex | wide, tFloat | wide, 0, tFloat | wide, 0},
		},
		{
			complex(float64(1), float64(0)),
			[]byte{version, tComplex | wide, tFloat | wide | 0b0001, 240, 63, tFloat | wide, 0},
		},
		{
			complex(float64(0), float64(1)),
			[]byte{version, tComplex | wide, tFloat | wide, 0, tFloat | wide | 0b0001, 240, 63},
		},
		{
			complex(1.23, -1.23),
			[]byte{version, tComplex | wide,
				tFloat | wide | 0b0111, 174, 71, 225, 122, 20, 174, 243, 63,
				tFloat | wide | 0b0111, 174, 71, 225, 122, 20, 174, 243, 191},
		},
		// String
		{
			"",
			[]byte{version, tString, 0},
		},
		{
			"a",
			[]byte{version, tString, 1, 97},
		},
		{
			"ab",
			[]byte{version, tString, 2, 97, 98},
		},
		{
			strings.Repeat("a", 256),
			append([]byte{version, tString | 0b0001, 1, 0}, []byte(strings.Repeat("a", 256))...),
		},
		{
			strings.Repeat("a", 65536),
			append([]byte{version, tString | 0b0010, 1, 0, 0}, []byte(strings.Repeat("a", 65536))...),
		},
		// Uintptr
		{
			uintptr(123456),
			[]byte{version, tUintptr | 0b0010, 1, 226, 64},
		},
		// Unsafe pointer
		{
			unsafe.Pointer(uintptr(123456)),
			[]byte{version, tUintptr | raw | 0b0010, 1, 226, 64},
		},
	}
	checkSerializer(args, t)
}

func TestBasicAliases(t *testing.T) {
	var args = []serializerTestArgs{
		{
			testBool(true),
			[]byte{
				version, tType, id(testBool(false)), tBool | tru,
			},
		},
		{
			testString("abcd"),
			[]byte{version, tType, id(testString("")), tString, 4, 97, 98, 99, 100},
		},
		{
			testInt8(123),
			[]byte{version, tType, id(testInt8(0)), tInt8 | signed, 246},
		},
		{
			testUint8(123),
			[]byte{version, tType, id(testUint8(0)), tInt8, 123},
		},
		{
			testInt16(12345),
			[]byte{version, tType, id(testInt16(0)), tInt16 | signed, 96, 114},
		},
		{
			testUint16(12345),
			[]byte{version, tType, id(testUint16(0)), tInt16, 48, 57},
		},
		{
			testInt32(1234567),
			[]byte{version, tType, id(testInt32(0)), tInt32 | signed | meta, 229, 173, 14},
		},
		{
			testUint32(1234567),
			[]byte{version, tType, id(testUint32(0)), tInt32 | meta, 210, 214, 135},
		},
		{
			testInt64(1234567890),
			[]byte{version, tType, id(testInt64(0)), tInt64 | signed | meta, 160, 147, 44, 5, 164},
		},
		{
			testUint64(1234567890),
			[]byte{version, tType, id(testUint64(0)), tInt64 | meta, 160, 73, 150, 2, 210},
		},
		{
			testInt(12345),
			[]byte{version, tType, id(testInt(0)), tInt | signed | 0b001, 96, 114},
		},
		{
			testUint(12345),
			[]byte{version, tType, id(testUint(0)), tInt | 0b001, 48, 57},
		},
		{
			testFloat32(123),
			[]byte{version, tType, id(testFloat32(0)), tFloat | 0b001, 246, 66},
		},
		{
			testFloat64(123),
			[]byte{version, tType, id(testFloat64(0)), tFloat | wide | 0b010, 192, 94, 64},
		},
		{
			testComplex64(1 + 2i),
			[]byte{
				version, tType, id(testComplex64(0)), tComplex,
				tFloat | 0b001, 128, 63,
				tFloat, 64,
			},
		},
		{
			testComplex128(1 + 2i),
			[]byte{
				version, tType, id(testComplex128(0)), tComplex | wide,
				tFloat | wide | 0b001, 240, 63,
				tFloat | wide, 64,
			},
		},
		{
			testUintptr(12345),
			[]byte{version, tType, id(testUintptr(0)), tUintptr | 0b001, 48, 57},
		},
		{
			testRawPtr(uintptr(12345)),
			[]byte{version, tType, id(testRawPtr(nil)), tUintptr | raw | 0b001, 48, 57},
		},
	}
	checkSerializer(args, t)
}

func TestListSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		// Slices
		{
			([]string)(nil),
			[]byte{version, tType | null, id([]string{}), tList},
		},
		{
			[]string{},
			[]byte{version, tType, id([]string{}), tList, 0},
		},
		{
			[]byte{1, 2, 3},
			[]byte{version, tType, id([]byte{}), tList, 3, tByte, 1, tByte, 2, tByte, 3},
		},
		{
			bytes.Repeat([]byte{1}, 256),
			append([]byte{version, tType, id([]byte{}), tList | 0b0001, 1, 0}, bytes.Repeat([]byte{tByte, 1}, 256)...),
		},
		{
			bytes.Repeat([]byte{1}, 65536),
			append([]byte{version, tType, id([]byte{}), tList | 0b0010, 1, 0, 0}, bytes.Repeat([]byte{tByte, 1}, 65536)...),
		},
		{
			[]int{},
			[]byte{version, tType, id([]int{}), tList, 0},
		},
		{
			[]int{1, -1, 0, 1234, -1234},
			[]byte{
				version, tType, id([]int{}), tList, 5,
				tInt | signed, 2,
				tInt | signed, 1,
				tInt | signed, 0,
				tInt | signed | 0b0001, 9, 164,
				tInt | signed | 0b0001, 9, 163,
			},
		},
		{
			[]any{uint16(1), false, 1.23, "abc"},
			[]byte{
				version, tType, id([]any{}), tList, 4,
				tInterface, tInt16 | meta, 1,
				tInterface, tBool,
				tInterface, tFloat | wide | 0b0111, 174, 71, 225, 122, 20, 174, 243, 63,
				tInterface, tString, 3, 97, 98, 99,
			},
		},
		{
			testSlice{"a", "b", "c"},
			[]byte{
				version, tType, id(testSlice{}), tList, 3,
				tString, 1, 97,
				tString, 1, 98,
				tString, 1, 99,
			},
		},
		{
			testGenericSlice[byte]{1, 2, 3},
			[]byte{
				version, tType, id(testGenericSlice[byte]{}), tList, 3,
				tByte, 1,
				tByte, 2,
				tByte, 3,
			},
		},
		{
			testRecSlice{testRecSlice{nil}, nil},
			[]byte{
				version, tType, id(testRecSlice{}), tList, 2,
				tType, id(testRecSlice{}), tList, 1, tType | null, id(testRecSlice{}), tList,
				tType | null, id(testRecSlice{}), tList,
			},
		},
		// Arrays
		{
			[0]int{},
			[]byte{version, tType, id([0]int{}), tList | fixed, 0},
		},
		{
			[3]byte{1, 2, 3},
			[]byte{version, tType, id([3]byte{}), tList | fixed, 3, tByte, 1, tByte, 2, tByte, 3},
		},
		{
			*(*[256]byte)(bytes.Repeat([]byte{1}, 256)),
			append([]byte{version, tType, id([256]byte{}), tList | fixed | 0b0001, 1, 0}, bytes.Repeat([]byte{tByte, 1}, 256)...),
		},
		{
			*(*[65536]byte)(bytes.Repeat([]byte{1}, 65536)),
			append([]byte{version, tType, id([65536]byte{}), tList | fixed | 0b0010, 1, 0, 0}, bytes.Repeat([]byte{tByte, 1}, 65536)...),
		},
		{
			[5]int{1, -1, 0, 1234, -1234},
			[]byte{
				version, tType, id([5]int{}), tList | fixed, 5,
				tInt | signed, 2,
				tInt | signed, 1,
				tInt | signed, 0,
				tInt | signed | 0b0001, 9, 164,
				tInt | signed | 0b0001, 9, 163,
			},
		},
		{
			[4]any{uint16(1), false, 1.23, "abc"},
			[]byte{
				version, tType, id([4]any{}), tList | fixed, 4,
				tInterface, tInt16 | meta, 1,
				tInterface, tBool,
				tInterface, tFloat | wide | 0b0111, 174, 71, 225, 122, 20, 174, 243, 63,
				tInterface, tString, 3, 97, 98, 99,
			},
		},
	}
	checkSerializer(args, t)
}

func TestMapSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		{
			map[string]int{},
			[]byte{version, tType, id(map[string]int{}), tMap, 0},
		},
		{
			map[string]byte{"a": 1},
			[]byte{version, tType, id(map[string]byte{}), tMap, 1, tString, 1, 97, tByte, 1},
		},
		{
			testRecMap{},
			[]byte{version, tType, id(testRecMap{}), tMap, 0},
		},
		{
			testRecMap{8: testRecMap{}},
			[]byte{version, tType, id(testRecMap{}), tMap, 1, tByte, 8, tType, id(testRecMap{}), tMap, 0},
		},
	}
	checkSerializer(args, t)
}

func TestPointerSerialization(t *testing.T) {
	args := make([]serializerTestArgs, 11)
	// *Nil
	args[0] = serializerTestArgs{
		(*any)(nil),
		[]byte{version, tPointer, tNil},
	}
	// *Bool
	v1 := false
	args[1] = serializerTestArgs{
		&v1,
		[]byte{version, tPointer, tBool},
	}
	v2 := true
	args[2] = serializerTestArgs{
		&v2,
		[]byte{version, tPointer, tBool | tru},
	}
	// *String
	v3 := ""
	args[3] = serializerTestArgs{
		&v3,
		[]byte{version, tPointer, tString, 0},
	}
	v4 := "abc"
	args[4] = serializerTestArgs{
		&v4,
		[]byte{version, tPointer, tString, 3, 97, 98, 99},
	}
	// *Int
	v5 := 123
	args[5] = serializerTestArgs{
		&v5,
		[]byte{version, tPointer, tInt | signed, 246},
	}
	v6 := -1234567
	args[6] = serializerTestArgs{
		&v6,
		[]byte{version, tPointer, tInt | signed | 0b0010, 37, 173, 13},
	}
	// *Uint
	v7 := uint(12345)
	args[7] = serializerTestArgs{
		&v7,
		[]byte{version, tPointer, tInt | 0b0001, 48, 57},
	}
	v8 := uint(12345678)
	args[8] = serializerTestArgs{
		&v8,
		[]byte{version, tPointer, tInt | 0b0010, 188, 97, 78},
	}
	v9 := true
	args[9] = serializerTestArgs{
		testPtr(&v9),
		[]byte{
			version,
			tType, id(testPtr(nil)),
			tPointer,
			tBool | tru,
		},
	}
	args[10] = serializerTestArgs{
		testRecPtr(nil),
		[]byte{
			version,
			tType, id(testRecPtr(nil)),
			tPointer,
			tNil,
		},
	}
	checkSerializer(args, t)
}

func TestReferenceSerialization(t *testing.T) {
	args := make([]serializerTestArgs, 3, 3)

	var v0 any
	v0 = &v0
	args[0] = serializerTestArgs{
		v0,
		[]byte{version, tPointer, tInterface, tRef, 1},
	}

	v1 := []any{nil, []byte{1, 2, 3}, nil}
	v1[0] = &(v1[1].([]byte)[0])
	v1[2] = &(v1[1].([]byte)[0])
	args[1] = serializerTestArgs{
		v1,
		[]byte{
			version,
			tType, id([]any{}), tList, 3, // slice header
			tInterface, tPointer, tByte, 1, // &(v1[1].([]byte)[0])
			tInterface, tType, id([]byte{}), tList, 3, tRef | val, 2, tByte, 2, tByte, 3, // nested slice
			tInterface, tRef, 2, // &(v1[1].([]byte)[0])
		},
	}

	b := [3]byte{1, 2, 3}
	v3 := []any{&b, &b[0]}
	args[2] = serializerTestArgs{
		v3,
		[]byte{
			version,
			tType, id([]any{}), tList, 2,
			tInterface, tPointer, tType, id([3]byte{1, 2, 3}), tList | fixed, 3, tByte, 1, tByte, 2, tByte, 3,
			tInterface, tRef, 3,
		},
	}

	checkSerializer(args, t)
}

func TestStructBasedSerialization(t *testing.T) {
	SetStructCodingMode(StructCodingModeDefault)
	s := &testStruct{
		F1: "abc",
		F2: true,
		F3: nil,
		F4: nil,
		f5: 321,
		f6: "#",
	}
	s.F3 = s
	s.F4 = &s.F1
	var args = []serializerTestArgs{
		{
			s,
			[]byte{
				version,
				tPointer,                            // *testStruct
				tType, id(testStruct{}), tStruct, 6, // struct header
				tString, 3, 97, 98, 99, // "abc"
				tBool | tru, // true
				tRef, 1,     // self ref
				tInterface, tRef, 2, // *s.F1
				tInt | signed | 0b0001, 2, 130, // 321
				tString, 1, 35,
			},
		},
		{
			errors.New("err"),
			[]byte{
				version, tPointer,
				tType, id(reflect.ValueOf(errors.New("")).Elem().Interface()), tStruct, 1,
				tString, 3, 101, 114, 114, // "err"
			},
		},
		{
			struct {
				f1 bool
				f2 byte
			}{
				true,
				123,
			},
			[]byte{
				version,
				tType, id(struct {
					f1 bool
					f2 byte
				}{}), tStruct, 2,
				tBool | tru,
				tByte, 123,
			},
		},
		{
			&testCustomStruct{
				f1: true,
				f2: "abc",
				f3: 123,
			},
			[]byte{
				version, tPointer,
				tType | custom, id(testCustomStruct{}), tStruct, 16,
				version, tType, id([]any{}), tList, 3,
				tInterface, tInt, 123,
				tInterface, tString, 3, 97, 98, 99,
				tInterface, tBool | tru,
			},
		},
		{
			testCustomUint(123),
			[]byte{
				version,
				tType | custom, id(testCustomUint(0)), tInt, 8,
				0, 0, 0, 0, 0, 0, 0, 123,
			},
		},
	}
	checkSerializer(args, t)

}
func TestIndexStructSerialization(t *testing.T) {
	SetStructCodingMode(StructCodingModeIndex)
	s := &testStruct{
		F1: "abc",
		F2: true,
		F3: nil,
		F4: nil,
		f5: 321,
		f6: "#",
	}
	s.F3 = s
	s.F4 = &s.F1
	var args = []serializerTestArgs{
		{
			s,
			[]byte{
				version,
				tPointer,                            // *testStruct
				tType, id(testStruct{}), tStruct, 6, // struct header
				56, 0, tString, 3, 97, 98, 99, // "abc"
				56, 2, tBool | tru, // true
				56, 4, tRef, 1, // self ref
				56, 6, tInterface, tRef, 2, // *s.F1
				56, 8, tInt | signed | 0b0001, 2, 130, // 321
				56, 10, tString, 1, 35,
			},
		},
		{
			errors.New("err"),
			[]byte{
				version, tPointer,
				tType, id(reflect.ValueOf(errors.New("")).Elem().Interface()), tStruct, 1,
				56, 0, tString, 3, 101, 114, 114, // "err"
			},
		},
		{
			struct {
				f1 bool
				f2 byte
			}{
				true,
				123,
			},
			[]byte{
				version,
				tType, id(struct {
					f1 bool
					f2 byte
				}{}), tStruct, 2,
				56, 0, tBool | tru,
				56, 2, tByte, 123,
			},
		},
		{
			&testCustomStruct{
				f1: true,
				f2: "abc",
				f3: 123,
			},
			[]byte{
				version, tPointer,
				tType | custom, id(testCustomStruct{}), tStruct, 16,
				version, tType, id([]any{}), tList, 3,
				tInterface, tInt, 123,
				tInterface, tString, 3, 97, 98, 99,
				tInterface, tBool | tru,
			},
		},
		{
			testCustomUint(123),
			[]byte{
				version,
				tType | custom, id(testCustomUint(0)), tInt, 8,
				0, 0, 0, 0, 0, 0, 0, 123,
			},
		},
	}
	checkSerializer(args, t)
}
func TestNameStructSerialization(t *testing.T) {
	SetStructCodingMode(StructCodingModeName)
	s := &testStruct{
		F1: "abc",
		F2: true,
		F3: nil,
		F4: nil,
		f5: 321,
		f6: "#",
	}
	s.F3 = s
	s.F4 = &s.F1
	var args = []serializerTestArgs{
		{
			s,
			[]byte{
				version,
				tPointer,                            // *testStruct
				tType, id(testStruct{}), tStruct, 6, // struct header
				112, 2, 70, 49, tString, 3, 97, 98, 99, // "abc"
				112, 2, 70, 50, tBool | tru, // true
				112, 2, 70, 51, tRef, 1, // self ref
				112, 2, 70, 52, tInterface, tRef, 2, // *s.F1
				112, 2, 102, 53, tInt | signed | 0b0001, 2, 130, // 321
				112, 2, 102, 54, tString, 1, 35,
			},
		},
		{
			errors.New("err"),
			[]byte{
				version, tPointer,
				tType, id(reflect.ValueOf(errors.New("")).Elem().Interface()), tStruct, 1,
				112, 1, 115, tString, 3, 101, 114, 114, // "err"
			},
		},

		{
			struct {
				first  int
				second bool
				third  byte
			}{
				10,
				false,
				112,
			},
			[]byte{
				version,
				tType, id(struct {
					first  int
					second bool
					third  byte
				}{}), tStruct, 3,
				112, 5, 102, 105, 114, 115, 116, tInt | signed, 20,
				112, 6, 115, 101, 99, 111, 110, 100, tBool,
				112, 5, 116, 104, 105, 114, 100, tByte, 112,
			},
		},
		{
			struct {
				f1 bool
				f2 byte
			}{
				true,
				123,
			},
			[]byte{
				version,
				tType, id(struct {
					f1 bool
					f2 byte
				}{}), tStruct, 2,
				112, 2, 102, 49, tBool | tru,
				112, 2, 102, 50, tByte, 123,
			},
		},
		{
			&testCustomStruct{
				f1: true,
				f2: "abc",
				f3: 123,
			},
			[]byte{
				version, tPointer,
				tType | custom, id(testCustomStruct{}), tStruct, 16,
				version, tType, id([]any{}), tList, 3,
				tInterface, tInt, 123,
				tInterface, tString, 3, 97, 98, 99,
				tInterface, tBool | tru,
			},
		},
		{
			testCustomUint(123),
			[]byte{
				version,
				tType | custom, id(testCustomUint(0)), tInt, 8,
				0, 0, 0, 0, 0, 0, 0, 123,
			},
		},
	}
	checkSerializer(args, t)
}
func TestChanSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		{
			(<-chan bool)(nil),
			[]byte{version, tType | null, id(make(<-chan bool)), tChan | byte(reflect.RecvDir)},
		},
		{
			make(chan int),
			[]byte{version, tType, id(make(chan int)), tChan | byte(reflect.BothDir), tInt, 0},
		},
		{
			make(chan<- bool, 1),
			[]byte{version, tType, id(make(chan<- bool)), tChan | byte(reflect.SendDir), tInt, 1},
		},
		{
			make(<-chan *testStruct, 2),
			[]byte{version, tType, id(make(<-chan *testStruct)), tChan | byte(reflect.RecvDir), tInt, 2},
		},
		{
			make(testChan, 10),
			[]byte{version, tType, id(testChan(nil)), tChan | byte(reflect.RecvDir), tInt, 10},
		},
	}
	checkSerializer(args, t)
}

func TestFuncSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		{
			(func(byte, bool) int8)(nil),
			[]byte{version, tType | null, id((func(byte, bool) int8)(nil)), tFunc},
		},
		{
			Serialize,
			[]byte{version, tType, id(Serialize), tFunc},
		},
		{
			Unserialize,
			[]byte{version, tType, id(Unserialize), tFunc},
		},
	}
	checkSerializer(args, t)
}

func checkSerializer(args []serializerTestArgs, t *testing.T) {
	registerTestTypes()
	serializer := NewSerializer()
	for _, arg := range args {
		data := serializer.Encode(arg.value)
		if !bytes.Equal(data, arg.data) {
			t.Errorf("Serializer::encode(%#v) expected %v, but actual value is %v", arg.value, arg.data, data)
		}
	}
}
