package codec

import (
	"bytes"
	"strings"
	"testing"
)

type serializerTestItems struct {
	value any
	data  []byte
}

func TestSerialization_Nil(t *testing.T) {
	items := []serializerTestItems{
		{
			nil,
			[]byte{version, id(nil), 0},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Bool(t *testing.T) {
	items := []serializerTestItems{
		{
			false,
			[]byte{version, id(false), 0},
		},
		{
			true,
			[]byte{version, id(false), 1},
		},
		{
			testBool(false),
			[]byte{version, id(testBool(false)), 0},
		},
		{
			testBool(true),
			[]byte{version, id(testBool(true)), 1},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_String(t *testing.T) {
	stringId := id("")
	items := []serializerTestItems{
		{
			"",
			[]byte{version, stringId, 0b0001_0000},
		},
		{
			"0123456789",
			[]byte{version, stringId, 0b0001_0000 | 10, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57},
		},
		{
			strings.Repeat("a", 255),
			append([]byte{version, stringId, 0b0010_0000, 255}, []byte(strings.Repeat("a", 255))...),
		},
		{
			strings.Repeat("a", 65536),
			append([]byte{version, stringId, 0b0011_0000 | 1, 0, 0}, []byte(strings.Repeat("a", 65536))...),
		},
		{
			testStr("abcd"),
			[]byte{version, id(testStr("")), 0b0001_0000 | 4, 97, 98, 99, 100},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Uint8(t *testing.T) {
	uint8Id := id(uint8(0))
	items := []serializerTestItems{
		{
			uint8(0),
			[]byte{version, uint8Id, 0},
		},
		{
			uint8(1),
			[]byte{version, uint8Id, 1},
		},
		{
			uint8(255),
			[]byte{version, uint8Id, 255},
		},
		{
			testUint8(123),
			[]byte{version, id(testUint8(0)), 123},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Int8(t *testing.T) {
	int8Id := id(int8(0))
	var items = []serializerTestItems{
		{
			int8(0),
			[]byte{version, int8Id, 0},
		},
		{
			int8(1),
			[]byte{version, int8Id, 2},
		},
		{
			int8(-1),
			[]byte{version, int8Id, 1},
		},
		{
			int8(127),
			[]byte{version, int8Id, 254},
		},
		{
			int8(-128),
			[]byte{version, int8Id, 255},
		},
		{
			testInt8(123),
			[]byte{version, id(testInt8(0)), 246},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Uint16(t *testing.T) {
	uint16Id := id(uint16(0))
	var items = []serializerTestItems{
		{
			uint16(0),
			[]byte{version, uint16Id, 0b0100_0000},
		},
		{
			uint16(1),
			[]byte{version, uint16Id, 0b0100_0000 | 1},
		},
		{
			uint16(256),
			[]byte{version, uint16Id, 0b1000_0000 | 1, 0},
		},
		{
			uint16(65535),
			[]byte{version, uint16Id, 0b1100_0000, 255, 255},
		},
		{
			testUint16(12345),
			[]byte{version, id(testUint16(0)), 0b1000_0000 | 48, 57},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Float32(t *testing.T) {
	float32Id := id(float32(0))
	var items = []serializerTestItems{
		{
			float32(0),
			[]byte{version, float32Id, 0b0010_0000},
		},
		{
			float32(1),
			[]byte{version, float32Id, 0b0110_0000, 128, 63},
		},
		{
			float32(10),
			[]byte{version, float32Id, 0b0110_0000, 32, 65},
		},
		{
			float32(-1),
			[]byte{version, float32Id, 0b0110_0000, 128, 191},
		},
		{
			float32(-10),
			[]byte{version, float32Id, 0b0110_0000, 32, 193},
		},
		{
			float32(1.23),
			[]byte{version, float32Id, 0b1010_0000, 164, 112, 157, 63},
		},
		{
			float32(-1.23),
			[]byte{version, float32Id, 0b1010_0000, 164, 112, 157, 191},
		},
		{
			testFloat32(123),
			[]byte{version, id(testFloat32(0)), 0b0110_0000, 246, 66},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Float64(t *testing.T) {
	float64Id := id(float64(0))
	var items = []serializerTestItems{
		{
			float64(0),
			[]byte{version, float64Id, 0b0001_0000},
		},
		{
			float64(1),
			[]byte{version, float64Id, 0b0011_0000, 240, 63},
		},
		{
			float64(10),
			[]byte{version, float64Id, 0b0011_0000, 36, 64},
		},
		{
			float64(-1),
			[]byte{version, float64Id, 0b0011_0000, 240, 191},
		},
		{
			float64(-10),
			[]byte{version, float64Id, 0b0011_0000, 36, 192},
		},
		{
			1.23,
			[]byte{version, float64Id, 0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 63},
		},
		{
			-1.23,
			[]byte{version, float64Id, 0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 191},
		},
		{
			testFloat64(123),
			[]byte{version, id(testFloat64(0)), 0b0100_0000, 192, 94, 64},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Int16(t *testing.T) {
	int16Id := id(int16(0))
	var items = []serializerTestItems{
		{
			int16(0),
			[]byte{version, int16Id, 0b0100_0000},
		},
		{
			int16(1),
			[]byte{version, int16Id, 0b0100_0000 | 2},
		},
		{
			int16(-1),
			[]byte{version, int16Id, 0b0100_0000 | 1},
		},
		{
			int16(256),
			[]byte{version, int16Id, 0b1000_0000 | 2, 0},
		},
		{
			int16(-256),
			[]byte{version, int16Id, 0b1000_0000 | 1, 255},
		},
		{
			int16(32767),
			[]byte{version, int16Id, 0b1100_0000, 255, 254},
		},
		{
			int16(-32768),
			[]byte{version, int16Id, 0b1100_0000, 255, 255},
		},
		{
			testInt16(-12345),
			[]byte{version, id(testInt16(0)), 0b1100_0000, 96, 113},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Pointer(t *testing.T) {
	var items = []serializerTestItems{
		{
			(*byte)(nil),
			[]byte{version, id((*byte)(nil)), 0b1000_0000},
		},
		{
			(*any)(nil),
			[]byte{version, id((*any)(nil)), 0b1000_0000},
		},
		{
			func() any {
				b := byte(255)
				return &b
			}(),
			[]byte{version, id((*byte)(nil)), 0, 255},
		},
		{
			func() any {
				s := "123"
				return &s
			}(),
			[]byte{version, id((*string)(nil)), 0, 0b0001_0000 | 3, '1', '2', '3'},
		},
		{
			func() any {
				var x any = true
				return &x
			}(),
			[]byte{version, id((*any)(nil)), 0, id(false), 1},
		},
	}
	checkEncodedData(t, items)
}

func TestSerialization_Slice(t *testing.T) {
	var items = []serializerTestItems{
		{
			([]string)(nil),
			[]byte{version, id([]string{}), 1},
		},
		{
			[]bool{},
			[]byte{version, id([]bool{}), 0, 0b0001_0000, 0b0001_0000},
		},
		{
			[]byte{1, 2, 3},
			[]byte{version, id([]byte{}), 0, 0b0001_0000 | 3, 0b0001_0000, 1, 2, 3},
		},
		{
			bytes.Repeat([]byte{1}, 256),
			append(
				[]byte{version, id([]byte{}), 0, 0b0010_0000 | 1, 0, 0b0001_0000},
				bytes.Repeat([]byte{1}, 256)...,
			),
		},
		{
			bytes.Repeat([]byte{1}, 65536),
			append(
				[]byte{version, id([]byte{}), 0, 0b0011_0000 | 1, 0, 0, 0b0001_0000},
				bytes.Repeat([]byte{1}, 65536)...,
			),
		},
		{
			[]string{"a", "bc", "def"},
			[]byte{
				version, id([]string{}), 0, 0b0001_0000 | 3, 0b0001_0000,
				0b0001_0000 | 1, 'a',
				0b0001_0000 | 2, 'b', 'c',
				0b0001_0000 | 3, 'd', 'e', 'f',
			},
		},
		{
			[]any{uint16(1), true, 1.23, "abc", nil},
			[]byte{
				version, id([]any{}), 0, 0b0001_0000 | 5, 0b0001_0000,
				id(uint16(0)), 0b0100_0000 | 1,
				id(false), 1,
				id(float64(0)), 0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 63,
				id(""), 0b0001_0000 | 3, 'a', 'b', 'c',
				id(nil), 0,
			},
		},
		{
			testSlice{"a", "b", "c"},
			[]byte{
				version, id(testSlice{}), 0, 0b0001_0000 | 3, 0b0001_0000,
				0b0001_0000 | 1, 'a',
				0b0001_0000 | 1, 'b',
				0b0001_0000 | 1, 'c',
			},
		},
		{
			testRecSlice{testRecSlice{nil}, nil},
			[]byte{
				version, id(testRecSlice{}), 0, 0b0001_0000 | 2, 0b0001_0000,
				0, 0b0001_0000 | 1, 0b0001_0000, 1,
				1,
			},
		},
	}
	checkEncodedData(t, items)
}

func checkEncodedData(t *testing.T, items []serializerTestItems) {
	serializer := NewSerializer()
	for _, item := range items {
		data := serializer.Encode(item.value)
		if !bytes.Equal(data, item.data) {
			t.Errorf("Encode(%v) must return %v, but actual value is %v", item.value, item.data, data)
		}
	}
}

/*

func TestUint32Serialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testUint32(1234567),
			[]byte{version, tType, id(testUint32(0)), tInt32 | meta, 210, 214, 135},
		},
	}
	checkSerializer(args, t)
}

func TestInt32Serialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testInt32(1234567),
			[]byte{version, tType, id(testInt32(0)), tInt32 | signed | meta, 229, 173, 14},
		},
	}
	checkSerializer(args, t)
}

func TestUint64Serialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testUint64(1234567890),
			[]byte{version, tType, id(testUint64(0)), tInt64 | meta, 160, 73, 150, 2, 210},
		},
	}
	checkSerializer(args, t)
}

func TestInt64Serialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testInt64(1234567890),
			[]byte{version, tType, id(testInt64(0)), tInt64 | signed | meta, 160, 147, 44, 5, 164},
		},
	}
	checkSerializer(args, t)
}

func TestUintSerialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testUint(12345),
			[]byte{version, tType, id(testUint(0)), tInt | 0b001, 48, 57},
		},
	}
	checkSerializer(args, t)
}

func TestIntSerialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testInt(12345),
			[]byte{version, tType, id(testInt(0)), tInt | signed | 0b001, 96, 114},
		},
	}
	checkSerializer(args, t)
}

func TestComplex64Serialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testComplex64(1 + 2i),
			[]byte{
				version, tType, id(testComplex64(0)), tComplex,
				tFloat | 0b001, 128, 63,
				tFloat, 64,
			},
		},
	}
	checkSerializer(args, t)
}

func TestComplex128Serialization(t *testing.T) {
	var args = []serializerTestArgs{
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
		{
			testComplex128(1 + 2i),
			[]byte{
				version, tType, id(testComplex128(0)), tComplex | wide,
				tFloat | wide | 0b001, 240, 63,
				tFloat | wide, 64,
			},
		},
	}
	checkSerializer(args, t)
}

func TestUintptrSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		{
			uintptr(123456),
			[]byte{version, tUintptr | 0b0010, 1, 226, 64},
		},
		{
			testUintptr(12345),
			[]byte{version, tType, id(testUintptr(0)), tUintptr | 0b001, 48, 57},
		},
	}
	checkSerializer(args, t)
}

func TestUnsafePointerSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		{
			unsafe.Pointer(nil),
			[]byte{version, tUintptr | raw, 0},
		},
		{
			unsafe.Pointer(uintptr(123456)),
			[]byte{version, tUintptr | raw | 0b0010, 1, 226, 64},
		},
		{
			testUnsafePointer(nil),
			[]byte{version, tType, id(testUnsafePointer(nil)), tUintptr | raw, 0},
		},
		{
			testUnsafePointer(uintptr(12345)),
			[]byte{version, tType, id(testUnsafePointer(nil)), tUintptr | raw | 0b001, 48, 57},
		},
	}
	checkSerializer(args, t)
}


func TestArraySerialization(t *testing.T) {
	var args = []serializerTestArgs{
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

func TestStructDefaultModeSerialization(t *testing.T) {
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
				tType, id(testStruct{}), tStruct, 7, // struct header
				tString, 3, 97, 98, 99, // "abc"
				tBool | tru, // true
				tRef, 1,     // self ref
				tInterface, tRef, 2, // *s.F1
				tInt | signed | 0b0001, 2, 130, // 321
				tString, 1, 35, // #
				tPointer | null, tType, id(testStruct{}), tStruct, // s.f7 = *nil
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
	}
	checkSerializer(args, t)

}
func TestStructIndexModeSerialization(t *testing.T) {
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
				tType, id(testStruct{}), tStruct, 7, // struct header
				tInt, 0, tString, 3, 97, 98, 99, // "abc"
				tInt, 1, tBool | tru, // true
				tInt, 2, tRef, 1, // self ref
				tInt, 3, tInterface, tRef, 2, // *s.F1
				tInt, 4, tInt | signed | 0b0001, 2, 130, // 321
				tInt, 5, tString, 1, 35, // #
				tInt, 6, tPointer | null, tType, id(testStruct{}), tStruct, // t.f7 = *nil
			},
		},
		{
			errors.New("err"),
			[]byte{
				version, tPointer,
				tType, id(reflect.ValueOf(errors.New("")).Elem().Interface()), tStruct, 1,
				tInt, 0, tString, 3, 101, 114, 114, // "err"
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
				tInt, 0, tBool | tru,
				tInt, 1, tByte, 123,
			},
		},
	}
	checkSerializer(args, t)
}
func TestStructNameModeSerialization(t *testing.T) {
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
				tType, id(testStruct{}), tStruct, 7, // struct header
				tString, 2, 70, 49, tString, 3, 97, 98, 99, // "abc"
				tString, 2, 70, 50, tBool | tru, // true
				tString, 2, 70, 51, tRef, 1, // self ref
				tString, 2, 70, 52, tInterface, tRef, 2, // *s.F1
				tString, 2, 102, 53, tInt | signed | 0b0001, 2, 130, // 321
				tString, 2, 102, 54, tString, 1, 35, // #
				tString, 2, 102, 55, tPointer | null, tType, id(testStruct{}), tStruct, // t.f7 = *nil
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

func TestInterfaceSerialization(t *testing.T) {
	SetStructCodingMode(StructCodingModeDefault)
	v := testInterface{}
	ioReaderId := tChecker.typeId(reflect.TypeOf((*io.Reader)(nil)).Elem())
	var args = []serializerTestArgs{
		{
			v,
			[]byte{
				version, tType, id(v), tStruct, 3, // struct header
				tType | null, byte(ioReaderId), tInterface, // io.Reader nil interface
				tPointer | null, tType, byte(ioReaderId), tInterface, // nil pointer to io.Reader interface
				tPointer | null, tType, id(testStruct{}), tStruct, // nil pointer to testStruct
			},
		},
	}
	checkSerializer(args, t)
}

func TestSerializableSerialization(t *testing.T) {
	var args = []serializerTestArgs{
		{
			testCustomUint(123),
			[]byte{
				version,
				tType | custom, id(testCustomUint(0)), tInt, 8,
				0, 0, 0, 0, 0, 0, 0, 123,
			},
		},
		{
			testCustomStruct{
				f1: true,
				f2: "abc",
				f3: 123,
			},
			[]byte{
				version,
				tType | custom, id(testCustomStruct{}), tStruct, 16,
				version, tType, id([]any{}), tList, 3,
				tInterface, tInt, 123,
				tInterface, tString, 3, 97, 98, 99,
				tInterface, tBool | tru,
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
			&testCustomPointerStruct{
				f1: true,
				f2: "abc",
				f3: 123,
			},
			[]byte{
				version, tPointer,
				tType | custom, id(testCustomPointerStruct{}), tStruct, 16,
				version, tType, id([]any{}), tList, 3,
				tInterface, tInt, 123,
				tInterface, tString, 3, 97, 98, 99,
				tInterface, tBool | tru,
			},
		},
		{
			testCustomPointerStruct{
				f1: true,
				f2: "abc",
				f3: 123,
			},
			[]byte{
				version,
				tType, id(testCustomPointerStruct{}), tStruct, 3,
				tBool | tru,
				tString, 3, 97, 98, 99,
				tInt, 123,
			},
		},
		{
			testCustomNestedStruct{
				data: "d1",
				ptr: &testCustomNestedStruct{
					data: "d2",
					ptr: &testCustomNestedStruct{
						data: "d3",
						ptr:  nil,
					},
				},
			},
			[]byte{
				version,
				tType, id(testCustomNestedStruct{}), tStruct, 2,
				tString, 2, 100, 49,
				tPointer, tType | custom, id(testCustomNestedStruct{}), tStruct, 31,
				version, tType, id([]any{}), tList, 2,
				tInterface, tString, 2, 100, 50,
				tInterface, tPointer, tType | custom, id(testCustomNestedStruct{}), tStruct, 15,
				version, tType, id([]any{}), tList, 2,
				tInterface, tString, 2, 100, 51,
				tInterface, tPointer | null, tType, id(testCustomNestedStruct{}), tStruct,
			},
		},
	}
	checkSerializer(args, t)
}

func TestPointerSerialization(t *testing.T) {
	args := make([]serializerTestArgs, 12)
	// *Nil
	args[0] = serializerTestArgs{
		(*any)(nil),
		[]byte{version, tPointer | null, tInterface},
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
	// *bool
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
	// rec ptr
	args[10] = serializerTestArgs{
		testRecPtr(testRecPtr(nil)),
		[]byte{
			version,
			tType | null, id(testRecPtr(nil)),
			tPointer, tType, id(testRecPtr(nil)), tPointer,
		},
	}
	// rec ptr with other types
	args[11] = serializerTestArgs{
		[]any{testRecPtr(nil), nil},
		[]byte{
			version,
			tType, id([]any{}), tList, 2,
			tInterface, tType | null, id(testRecPtr(nil)),
			tPointer, tType, id(testRecPtr(nil)), tPointer,
			tInterface | null,
		},
	}
	checkSerializer(args, t)
}

func TestReferenceSerialization(t *testing.T) {
	args := make([]serializerTestArgs, 4)

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
			tInterface, tPointer, tType, id(b), tList | fixed, 3, tByte, 1, tByte, 2, tByte, 3,
			tInterface, tRef, 3,
		},
	}

	/*l := newLst()
	v4 := l.push()
	_ = l.push()
	// n1.prev = *lst.root
	// n1.next = *n2
	// n1.lst = *lst
	// n2.prev = *n1
	// n2.next = *lst.root
	// n2.lst = *lst
	// lst.root.prev = *n2
	// lst.root.next = *n1
	// lst.root.lst = nil
	args[3] = serializerTestArgs{
		v4,
		[]byte{
			version,
			tPointer, tType, id(node{}), tStruct, 3, // *n1
				tPointer, tType, id(node{}), tStruct, 3, // n1.prev = *lst.root
					tPointer, tType, id(node{}), tStruct, 3, // lst.root.prev = *n2
						tRef, 1, // n2.prev = ref (n1)
						tRef, 3, // n2.next = ref (lst.root)
						tPointer, tType, id(lst{}), tStruct, 1, // n2.lst
							tRef | val, 3, // lst.root value
					tRef, 1, // lst.root.next = *n1 (ref)
					tPointer | null, tType, id(lst{}), tStruct, // lst.root.lst = nil
				tRef, 5, // n1.next = *n2 (ref)
				tRef, 9, // n1.lst = *lst (ref)
		},
	}

	var n0, n1 any
	n1 = &n0
	n0 = &n1
	args[3] = serializerTestArgs{
		n0,
		[]byte{
			version,
			tPointer, tInterface,
			tPointer, tInterface,
			tRef, 1,
		},
	}

	checkSerializer(args, t)
}
*/
