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

func registry() (*TypeRegistry, func(v any) byte) {
	reg := NewTypeRegistry(true)
	return reg, func(v any) byte {
		id := reg.typeIdByValue(reflect.ValueOf(v))
		return asBytesWithSize(uint64(id), 3)[0]
	}
}

func interfaceId(reg *TypeRegistry) byte {
	id, exists := reg.typeIdByName("interface {}")
	if exists {
		return asBytesWithSize(uint64(id), 3)[0]
	}
	reg.RegisterType(reflect.TypeOf((*any)(nil)).Elem())
	return interfaceId(reg)
}

type serializerTestItems struct {
	value any
	data  []byte
}

func TestSerialization_Nil(t *testing.T) {
	reg, id := registry()
	items := []serializerTestItems{
		{
			nil,
			[]byte{version, id(nil), meta_nil},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Bool(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_String(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_Uint8(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_Int8(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_Uint16(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_Float32(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_Float64(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_Complex64(t *testing.T) {
	reg, id := registry()
	complex64Id := id(complex64(0))
	var items = []serializerTestItems{
		{
			complex(float32(0), float32(0)),
			[]byte{version, complex64Id, 0b0010_0000, 0b0010_0000},
		},
		{
			complex(float32(1), float32(0)),
			[]byte{version, complex64Id, 0b0110_0000, 128, 63, 0b0010_0000},
		},
		{
			complex(float32(0), float32(1)),
			[]byte{version, complex64Id, 0b0010_0000, 0b0110_0000, 128, 63},
		},
		{
			complex(float32(1.23), float32(-1.23)),
			[]byte{version, complex64Id, 0b1010_0000, 164, 112, 157, 63, 0b1010_0000, 164, 112, 157, 191},
		},
		{
			testComplex64(1 + 2i),
			[]byte{version, id(testComplex64(0)), 0b0110_0000, 128, 63, 0b0100_0000, 64},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Complex128(t *testing.T) {
	reg, id := registry()
	complex128Id := id(complex128(0))
	var items = []serializerTestItems{
		{
			complex(float64(0), float64(0)),
			[]byte{version, complex128Id, 0b0001_0000, 0b0001_0000},
		},
		{
			complex(float64(1), float64(0)),
			[]byte{version, complex128Id, 0b0011_0000, 240, 63, 0b0001_0000},
		},
		{
			complex(float64(0), float64(1)),
			[]byte{version, complex128Id, 0b0001_0000, 0b0011_0000, 240, 63},
		},
		{
			complex(1.23, -1.23),
			[]byte{
				version, complex128Id,
				0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 63,
				0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 191,
			},
		},
		{
			testComplex128(1 + 2i),
			[]byte{version, id(testComplex128(0)), 0b0011_0000, 240, 63, 0b0010_0000, 64},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Int16(t *testing.T) {
	reg, id := registry()
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
	checkEncodedData(t, reg, items)
}

func TestSerialization_Uint32(t *testing.T) {
	reg, id := registry()
	uint32Id := id(uint32(0))
	var items = []serializerTestItems{
		{
			uint32(0),
			[]byte{version, uint32Id, 0b0010_0000},
		},
		{
			uint32(1),
			[]byte{version, uint32Id, 0b0010_0000 | 1},
		},
		{
			uint32(256),
			[]byte{version, uint32Id, 0b0100_0000 | 1, 0},
		},
		{
			uint32(123456),
			[]byte{version, uint32Id, 0b0110_0000 | 1, 226, 64},
		},
		{
			uint32(math.MaxUint32),
			[]byte{version, uint32Id, 0b1010_0000, 255, 255, 255, 255},
		},
		{
			testUint32(12345678),
			[]byte{version, id(testUint32(0)), 0b1000_0000, 188, 97, 78},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Int32(t *testing.T) {
	reg, id := registry()
	int32Id := id(int32(0))
	var items = []serializerTestItems{
		{
			int32(0),
			[]byte{version, int32Id, 0b0010_0000},
		},
		{
			int32(1),
			[]byte{version, int32Id, 0b0010_0000 | 2},
		},
		{
			int32(-1),
			[]byte{version, int32Id, 0b0010_0000 | 1},
		},
		{
			int32(256),
			[]byte{version, int32Id, 0b0100_0000 | 2, 0},
		},
		{
			int32(-256),
			[]byte{version, int32Id, 0b0100_0000 | 1, 255},
		},
		{
			int32(123456),
			[]byte{version, int32Id, 0b0110_0000 | 3, 196, 128},
		},
		{
			int32(-123456),
			[]byte{version, int32Id, 0b0110_0000 | 3, 196, 127},
		},
		{
			int32(math.MaxInt32),
			[]byte{version, int32Id, 0b1010_0000, 255, 255, 255, 254},
		},
		{
			int32(math.MinInt32),
			[]byte{version, int32Id, 0b1010_0000, 255, 255, 255, 255},
		},
		{
			testInt32(1234567),
			[]byte{version, id(testInt32(0)), 0b1000_0000, 37, 173, 14},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Uint64(t *testing.T) {
	reg, id := registry()
	uint64Id := id(uint64(0))
	var items = []serializerTestItems{
		{
			uint64(0),
			[]byte{version, uint64Id, 0b0001_0000},
		},
		{
			uint64(1),
			[]byte{version, uint64Id, 0b0001_0000 | 1},
		},
		{
			uint64(1 << 8),
			[]byte{version, uint64Id, 0b0010_0000 | 1, 0},
		},
		{
			uint64(1 << 16),
			[]byte{version, uint64Id, 0b0011_0000 | 1, 0, 0},
		},
		{
			uint64(1 << 24),
			[]byte{version, uint64Id, 0b0100_0000 | 1, 0, 0, 0},
		},
		{
			uint64(1 << 32),
			[]byte{version, uint64Id, 0b0101_0000 | 1, 0, 0, 0, 0},
		},
		{
			uint64(1 << 40),
			[]byte{version, uint64Id, 0b0110_0000 | 1, 0, 0, 0, 0, 0},
		},
		{
			uint64(1 << 48),
			[]byte{version, uint64Id, 0b0111_0000 | 1, 0, 0, 0, 0, 0, 0},
		},
		{
			uint64(1 << 56),
			[]byte{version, uint64Id, 0b1000_0000 | 1, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			uint64(1<<64 - 1),
			[]byte{version, uint64Id, 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			testUint64(1234567890),
			[]byte{version, id(testUint64(0)), 0b0101_0000, 73, 150, 2, 210},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Int64(t *testing.T) {
	reg, id := registry()
	int64Id := id(int64(0))
	var items = []serializerTestItems{
		{
			int64(0),
			[]byte{version, int64Id, 0b0001_0000},
		},
		{
			int64(1),
			[]byte{version, int64Id, 0b0001_0000 | 2},
		},
		{
			int64(-1),
			[]byte{version, int64Id, 0b0001_0000 | 1},
		},
		{
			int64(1 << 8),
			[]byte{version, int64Id, 0b0010_0000 | 2, 0},
		},
		{
			int64(-1 << 8),
			[]byte{version, int64Id, 0b0010_0000 | 1, 255},
		},
		{
			int64(1 << 16),
			[]byte{version, int64Id, 0b0011_0000 | 2, 0, 0},
		},
		{
			int64(-1 << 16),
			[]byte{version, int64Id, 0b0011_0000 | 1, 255, 255},
		},
		{
			int64(1 << 24),
			[]byte{version, int64Id, 0b0100_0000 | 2, 0, 0, 0},
		},
		{
			int64(-1 << 24),
			[]byte{version, int64Id, 0b0100_0000 | 1, 255, 255, 255},
		},
		{
			int64(1 << 32),
			[]byte{version, int64Id, 0b0101_0000 | 2, 0, 0, 0, 0},
		},
		{
			int64(-1 << 32),
			[]byte{version, int64Id, 0b0101_0000 | 1, 255, 255, 255, 255},
		},
		{
			int64(1 << 40),
			[]byte{version, int64Id, 0b0110_0000 | 2, 0, 0, 0, 0, 0},
		},
		{
			int64(-1 << 40),
			[]byte{version, int64Id, 0b0110_0000 | 1, 255, 255, 255, 255, 255},
		},
		{
			int64(1 << 48),
			[]byte{version, int64Id, 0b0111_0000 | 2, 0, 0, 0, 0, 0, 0},
		},
		{
			int64(-1 << 48),
			[]byte{version, int64Id, 0b0111_0000 | 1, 255, 255, 255, 255, 255, 255},
		},
		{
			int64(1 << 56),
			[]byte{version, int64Id, 0b1000_0000 | 2, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			int64(-1 << 56),
			[]byte{version, int64Id, 0b1000_0000 | 1, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			int64(math.MaxInt64),
			[]byte{version, int64Id, 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 254},
		},
		{
			int64(math.MinInt64),
			[]byte{version, int64Id, 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			testInt64(1234567890),
			[]byte{version, id(testInt64(0)), 0b0101_0000, 147, 44, 5, 164},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Uint(t *testing.T) {
	reg, id := registry()
	uintId := id(uint(0))
	var items = []serializerTestItems{
		{
			uint(0),
			[]byte{version, uintId, 0b0001_0000},
		},
		{
			uint(1),
			[]byte{version, uintId, 0b0001_0000 | 1},
		},
		{
			uint(1<<8 - 1),
			[]byte{version, uintId, 0b0010_0000, 255},
		},
		{
			uint(1 << 8),
			[]byte{version, uintId, 0b0010_0000 | 1, 0},
		},
		{
			uint(1<<16 - 1),
			[]byte{version, uintId, 0b0011_0000, 255, 255},
		},
		{
			uint(1 << 16),
			[]byte{version, uintId, 0b0011_0000 | 1, 0, 0},
		},
		{
			uint(1<<24 - 1),
			[]byte{version, uintId, 0b0100_0000, 255, 255, 255},
		},
		{
			uint(1 << 24),
			[]byte{version, uintId, 0b0100_0000 | 1, 0, 0, 0},
		},
		{
			uint(1<<32 - 1),
			[]byte{version, uintId, 0b0101_0000, 255, 255, 255, 255},
		},
	}
	if math.MaxUint == math.MaxUint64 {
		items = append(items, []serializerTestItems{
			{
				uint(1 << 32),
				[]byte{version, uintId, 0b0101_0000 | 1, 0, 0, 0, 0},
			},
			{
				uint(1<<40 - 1),
				[]byte{version, uintId, 0b0110_0000, 255, 255, 255, 255, 255},
			},
			{
				uint(1 << 40),
				[]byte{version, uintId, 0b0110_0000 | 1, 0, 0, 0, 0, 0},
			},
			{
				uint(1<<48 - 1),
				[]byte{version, uintId, 0b0111_0000, 255, 255, 255, 255, 255, 255},
			},
			{
				uint(1 << 48),
				[]byte{version, uintId, 0b0111_0000 | 1, 0, 0, 0, 0, 0, 0},
			},
			{
				uint(1<<56 - 1),
				[]byte{version, uintId, 0b1000_0000, 255, 255, 255, 255, 255, 255, 255},
			},
			{
				uint(1 << 56),
				[]byte{version, uintId, 0b1000_0000 | 1, 0, 0, 0, 0, 0, 0, 0},
			},
			{
				uint(1<<64 - 1),
				[]byte{version, uintId, 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
			},
			{
				testUint(12345),
				[]byte{version, id(testUint(0)), 0b0011_0000, 48, 57},
			},
		}...)
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Int(t *testing.T) {
	reg, id := registry()
	intId := id(0)
	var items = []serializerTestItems{
		{
			0,
			[]byte{version, intId, 0b0001_0000},
		},
		{
			1<<7 - 1,
			[]byte{version, intId, 0b0010_0000, 254},
		},
		{
			-1 << 7,
			[]byte{version, intId, 0b0010_0000, 255},
		},
		{
			1 << 7,
			[]byte{version, intId, 0b0010_0000 | 1, 0},
		},
		{
			-1<<7 - 1,
			[]byte{version, intId, 0b0010_0000 | 1, 1},
		},
		{
			1<<15 - 1,
			[]byte{version, intId, 0b0011_0000, 255, 254},
		},
		{
			-1 << 15,
			[]byte{version, intId, 0b0011_0000, 255, 255},
		},
		{
			1 << 15,
			[]byte{version, intId, 0b0011_0000 | 1, 0, 0},
		},
		{
			-1<<15 - 1,
			[]byte{version, intId, 0b0011_0000 | 1, 0, 1},
		},
		{
			1<<23 - 1,
			[]byte{version, intId, 0b0100_0000, 255, 255, 254},
		},
		{
			-1 << 23,
			[]byte{version, intId, 0b0100_0000, 255, 255, 255},
		},
		{
			1 << 23,
			[]byte{version, intId, 0b0100_0000 | 1, 0, 0, 0},
		},
		{
			-1<<23 - 1,
			[]byte{version, intId, 0b0100_0000 | 1, 0, 0, 1},
		},
	}
	if math.MaxInt == math.MaxInt64 {
		items = append(items, []serializerTestItems{
			{
				1<<32 - 1,
				[]byte{version, intId, 0b0101_0000 | 1, 255, 255, 255, 254},
			},
			{
				-1 << 32,
				[]byte{version, intId, 0b0101_0000 | 1, 255, 255, 255, 255},
			},
			{
				1 << 32,
				[]byte{version, intId, 0b0101_0000 | 2, 0, 0, 0, 0},
			},
			{
				-1<<32 - 1,
				[]byte{version, intId, 0b0101_0000 | 2, 0, 0, 0, 1},
			},
			{
				1<<40 - 1,
				[]byte{version, intId, 0b0110_0000 | 1, 255, 255, 255, 255, 254},
			},
			{
				-1 << 40,
				[]byte{version, intId, 0b0110_0000 | 1, 255, 255, 255, 255, 255},
			},
			{
				1 << 40,
				[]byte{version, intId, 0b0110_0000 | 2, 0, 0, 0, 0, 0},
			},
			{
				-1<<40 - 1,
				[]byte{version, intId, 0b0110_0000 | 2, 0, 0, 0, 0, 1},
			},
			{
				1<<48 - 1,
				[]byte{version, intId, 0b0111_0000 | 1, 255, 255, 255, 255, 255, 254},
			},
			{
				-1 << 48,
				[]byte{version, intId, 0b0111_0000 | 1, 255, 255, 255, 255, 255, 255},
			},
			{
				1 << 48,
				[]byte{version, intId, 0b0111_0000 | 2, 0, 0, 0, 0, 0, 0},
			},
			{
				-1<<48 - 1,
				[]byte{version, intId, 0b0111_0000 | 2, 0, 0, 0, 0, 0, 1},
			},
			{
				1<<56 - 1,
				[]byte{version, intId, 0b1000_0000 | 1, 255, 255, 255, 255, 255, 255, 254},
			},
			{
				-1 << 56,
				[]byte{version, intId, 0b1000_0000 | 1, 255, 255, 255, 255, 255, 255, 255},
			},
			{
				1 << 56,
				[]byte{version, intId, 0b1000_0000 | 2, 0, 0, 0, 0, 0, 0, 0},
			},
			{
				-1<<56 - 1,
				[]byte{version, intId, 0b1000_0000 | 2, 0, 0, 0, 0, 0, 0, 1},
			},
			{
				math.MaxInt,
				[]byte{version, intId, 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 254},
			},
			{
				math.MinInt,
				[]byte{version, intId, 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
			},
			{
				testInt(12345),
				[]byte{version, id(testInt(0)), 0b0011_0000, 96, 114},
			},
		}...)
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Uintptr(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			uintptr(123456),
			[]byte{version, id(uintptr(0)), 0b0011_0000 | 1, 226, 64},
		},
		{
			testUintptr(12345),
			[]byte{version, id(testUintptr(0)), 0b0011_0000, 48, 57},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_UnsafePointer(t *testing.T) {
	reg, id := registry()
	ptrId := id(unsafe.Pointer(nil))
	var items = []serializerTestItems{
		{
			unsafe.Pointer(nil),
			[]byte{version, ptrId, 0b0001_0000},
		},
		{
			unsafe.Pointer(uintptr(0)),
			[]byte{version, ptrId, 0b0001_0000},
		},
		{
			unsafe.Pointer(uintptr(123456)),
			[]byte{version, ptrId, 0b0011_0000 | 1, 226, 64},
		},
		{
			testUnsafePointer(nil),
			[]byte{version, id(testUnsafePointer(nil)), 0b0001_0000},
		},
		{
			testUnsafePointer(uintptr(12345)),
			[]byte{version, id(testUnsafePointer(nil)), 0b0011_0000, 48, 57},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Chan(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			(<-chan bool)(nil),
			[]byte{version, id(make(<-chan bool)), meta_nil | byte(reflect.RecvDir)},
		},
		{
			make(chan int),
			[]byte{version, id(make(chan int)), byte(reflect.BothDir), 0b0001_0000},
		},
		{
			make(chan<- bool, 1),
			[]byte{version, id(make(chan<- bool)), byte(reflect.SendDir), 0b0001_0000 | 1},
		},
		{
			make(testChan, 10),
			[]byte{version, id(testChan(nil)), byte(reflect.RecvDir), 0b0001_0000 | 10},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Func(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			(func(byte, bool) int8)(nil),
			[]byte{version, id((func(byte, bool) int8)(nil)), meta_nil},
		},
		{
			registry,
			[]byte{version, id(registry), 0},
		},
		{
			math.Abs,
			[]byte{version, id(math.Abs), 0},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Slice(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			([]string)(nil),
			[]byte{version, id([]string{}), meta_nil},
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
				version,
				id([]string{}), 0, 0b0001_0000 | 3, 0b0001_0000,
				0b0001_0000 | 1, 'a',
				0b0001_0000 | 2, 'b', 'c',
				0b0001_0000 | 3, 'd', 'e', 'f',
			},
		},
		{
			[]any{uint16(1), true, 1.23, "abc", nil},
			[]byte{
				version,
				id([]any{}), 0, 0b0001_0000 | 5, 0b0001_0000,
				id(uint16(0)), 0b0100_0000 | 1,
				id(false), 1,
				id(float64(0)), 0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 63,
				id(""), 0b0001_0000 | 3, 'a', 'b', 'c',
				id(nil), meta_nil,
			},
		},
		{
			testSlice{"a", "b", "c"},
			[]byte{
				version,
				id(testSlice{}), 0, 0b0001_0000 | 3, 0b0001_0000,
				0b0001_0000 | 1, 'a',
				0b0001_0000 | 1, 'b',
				0b0001_0000 | 1, 'c',
			},
		},
		{
			testRecSlice{testRecSlice{nil}, nil},
			[]byte{
				version,
				id(testRecSlice{}), 0, 0b0001_0000 | 2, 0b0001_0000,
				0, 0b0001_0000 | 1, 0b0001_0000, meta_nil,
				meta_nil,
			},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Array(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			[0]int{},
			[]byte{version, id([0]int{}), meta_fixed, 0b0001_0000, 0b0001_0000},
		},
		{
			[3]byte{1, 2, 3},
			[]byte{version, id([3]byte{}), meta_fixed, 0b0001_0000 | 3, 0b0001_0000, 1, 2, 3},
		},
		{
			*(*[256]byte)(bytes.Repeat([]byte{1}, 256)),
			append(
				[]byte{version, id([256]byte{}), meta_fixed, 0b0010_0000 | 1, 0, 0b0001_0000},
				bytes.Repeat([]byte{1}, 256)...,
			),
		},
		{
			[5]int{1, -1, 0, 1234, -1234},
			[]byte{
				version,
				id([5]int{}), meta_fixed, 0b0001_0000 | 5, 0b0001_0000,
				0b0001_0000 | 2,
				0b0001_0000 | 1,
				0b0001_0000,
				0b0010_0000 | 9, 164,
				0b0010_0000 | 9, 163,
			},
		},
		{
			[4]any{uint16(1), false, 1.23, "abc"},
			[]byte{
				version,
				id([4]any{}), meta_fixed, 0b0001_0000 | 4, 0b0001_0000,
				id(uint16(0)), 0b0100_0000 | 1,
				id(false), 0,
				id(float64(0)), 0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 63,
				id(""), 0b0001_0000 | 3, 'a', 'b', 'c',
			},
		},
		{
			testArray{-1, 1, 0},
			[]byte{
				version,
				id(testArray{}), meta_fixed, 0b0001_0000 | 3, 0b0001_0000,
				0b0001_0000 | 1, 0b0001_0000 | 2, 0b0001_0000,
			},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Map(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			(map[byte]bool)(nil),
			[]byte{version, id((map[byte]bool)(nil)), meta_nil},
		},
		{
			map[string]int{},
			[]byte{version, id((map[string]int)(nil)), 0, 0b0001_0000, 0b0001_0000},
		},
		{
			map[string]byte{"a": 1},
			[]byte{version, id((map[string]byte)(nil)), 0, 0b0001_0000 | 1, 0b0001_0000, 0b0001_0000 | 1, 'a', 1},
		},
		{
			testRecMap{},
			[]byte{version, id(testRecMap(nil)), 0, 0b0001_0000, 0b0001_0000},
		},
		{
			testRecMap{8: testRecMap(nil)},
			[]byte{version, id(testRecMap(nil)), 0, 0b0001_0000 | 1, 0b0001_0000, 8, meta_nil},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_StructDefaultCodingMode(t *testing.T) {
	SetDefaultStructCodingMode(StructCodingModeDefault)
	reg, id := registry()
	var items = []serializerTestItems{
		{
			func() any {
				s := &testStruct{
					F1: "abc",
					F2: true,
					F3: nil,
					F4: nil,
					f5: 321,
					f6: "#",
					f7: nil,
				}
				s.F3 = s
				s.F4 = &s.F1
				return s
			}(),
			[]byte{
				version,
				id((*testStruct)(nil)), 0, // *testStruct
				0b0001_0000 | 7,                        // field count
				id(""), 0b0001_0000 | 3, 'a', 'b', 'c', // F1
				id(false), 1, // F2
				id((*testStruct)(nil)), 0, meta_ref, 0b0010_0000 | 1, // F3 = &testStruct
				interfaceId(reg), id((*string)(nil)), 0, meta_ref, 0b0010_0000 | 2, // F4 = &F1
				id(0), 0b0010_0000 | 2, 130, // f5
				id(""), 0b0001_0000 | 1, '#', // f6
				id((*testStruct)(nil)), meta_nil, // f7
			},
		},
		{
			errors.New("err"),
			[]byte{
				version,
				id(reflect.ValueOf(errors.New("")).Interface()), 0, // &errors.errorString
				0b0001_0000 | 1,                        // errors.errorString
				id(""), 0b0001_0000 | 3, 'e', 'r', 'r', // "err"
			},
		},
		{
			struct {
				_  string
				f1 bool
				f2 byte
			}{
				"",
				true,
				123,
			},
			[]byte{
				version,
				id(struct {
					_  string
					f1 bool
					f2 byte
				}{}), 0b0001_0000 | 3,
				id(""), 0b0001_0000,
				id(false), 1,
				id(byte(0)), 123,
			},
		},
	}
	checkEncodedData(t, reg, items)
	SetDefaultStructCodingMode(StructCodingModeDefault)
}

func TestSerialization_StructIndexCodingMode(t *testing.T) {
	SetDefaultStructCodingMode(StructCodingModeIndex)
	reg, id := registry()
	var items = []serializerTestItems{
		{
			func() any {
				s := &testStruct{
					F1: "abc",
					F2: true,
					F3: nil,
					F4: nil,
					f5: 321,
					f6: "#",
					f7: nil,
				}
				s.F3 = s
				s.F4 = &s.F1
				return s
			}(),
			[]byte{
				version,
				id((*testStruct)(nil)), 0, // *testStruct
				0b0001_0000 | 7,                        // field count
				0b0001_0000,                            // F1 index = 0
				id(""), 0b0001_0000 | 3, 'a', 'b', 'c', // F1
				0b0001_0000 | 1, // F2 index = 1
				id(false), 1,    // F2
				0b0001_0000 | 2,                                      // F3 index = 2
				id((*testStruct)(nil)), 0, meta_ref, 0b0010_0000 | 1, // F3 = &testStruct
				0b0001_0000 | 3,                                                    // F4 index = 3
				interfaceId(reg), id((*string)(nil)), 0, meta_ref, 0b0010_0000 | 2, // F4 = &F1
				0b0001_0000 | 4,             // f5 index = 4
				id(0), 0b0010_0000 | 2, 130, // f5
				0b0001_0000 | 5,              // f6 index = 5
				id(""), 0b0001_0000 | 1, '#', // f6
				0b0001_0000 | 6,                  // f7 index = 6
				id((*testStruct)(nil)), meta_nil, // f7
			},
		},
		{
			errors.New("err"),
			[]byte{
				version,
				id(reflect.ValueOf(errors.New("")).Interface()), 0, // &errors.errorString
				0b0001_0000 | 1,                        // errors.errorString
				0b0001_0000,                            // field index
				id(""), 0b0001_0000 | 3, 'e', 'r', 'r', // "err"
			},
		},
		{
			struct {
				_  string
				f1 bool
				f2 byte
			}{
				"",
				true,
				123,
			},
			[]byte{
				version,
				id(struct {
					_  string
					f1 bool
					f2 byte
				}{}), 0b0001_0000 | 3,
				0b0001_0000, id(""), 0b0001_0000,
				0b0001_0000 | 1, id(false), 1,
				0b0001_0000 | 2, id(byte(0)), 123,
			},
		},
	}
	checkEncodedData(t, reg, items)
	SetDefaultStructCodingMode(StructCodingModeDefault)
}

func TestSerialization_StructNameCodingMode(t *testing.T) {
	SetDefaultStructCodingMode(StructCodingModeName)
	reg, id := registry()
	var items = []serializerTestItems{
		{
			func() any {
				s := &testStruct{
					F1: "abc",
					F2: true,
					F3: nil,
					F4: nil,
					f5: 321,
					f6: "#",
					f7: nil,
				}
				s.F3 = s
				s.F4 = &s.F1
				return s
			}(),
			[]byte{
				version,
				id((*testStruct)(nil)), 0, // *testStruct
				0b0001_0000 | 7,           // field count
				0b0001_0000 | 2, 'F', '1', // F1 name
				id(""), 0b0001_0000 | 3, 'a', 'b', 'c', // F1
				0b0001_0000 | 2, 'F', '2', // F2 name
				id(false), 1, // F2
				0b0001_0000 | 2, 'F', '3', // F3 name
				id((*testStruct)(nil)), 0, meta_ref, 0b0010_0000 | 1, // F3 = &testStruct
				0b0001_0000 | 2, 'F', '4', // F4 name
				interfaceId(reg), id((*string)(nil)), 0, meta_ref, 0b0010_0000 | 2, // F4 = &F1
				0b0001_0000 | 2, 'f', '5', // f5 name
				id(0), 0b0010_0000 | 2, 130, // f5
				0b0001_0000 | 2, 'f', '6', // f6 name
				id(""), 0b0001_0000 | 1, '#', // f6
				0b0001_0000 | 2, 'f', '7', // f7 name
				id((*testStruct)(nil)), meta_nil, // f7
			},
		},
		{
			errors.New("err"),
			[]byte{
				version,
				id(reflect.ValueOf(errors.New("")).Interface()), 0, // &errors.errorString
				0b0001_0000 | 1,      // errors.errorString
				0b0001_0000 | 1, 's', // field name
				id(""), 0b0001_0000 | 3, 'e', 'r', 'r', // "err"
			},
		},
		{
			struct {
				_  string
				f1 bool
				f2 byte
			}{
				"",
				true,
				123,
			},
			[]byte{
				version,
				id(struct {
					_  string
					f1 bool
					f2 byte
				}{}), 0b0001_0000 | 3,
				0b0001_0000 | 1, '_', 0b0001_0000, // _ name + index
				id(""), 0b0001_0000, // _ value
				0b0001_0000 | 2, 'f', '1', // f1 name
				id(false), 1, // f1 value
				0b0001_0000 | 2, 'f', '2', // f2 name
				id(byte(0)), 123, // f2 value
			},
		},
	}
	checkEncodedData(t, reg, items)
	SetDefaultStructCodingMode(StructCodingModeDefault)
}

func TestSerialization_Pointer(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			(*byte)(nil),
			[]byte{version, id((*byte)(nil)), meta_nil},
		},
		{
			(*any)(nil),
			[]byte{version, id((*any)(nil)), meta_nil},
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
		{
			func() any {
				b := true
				return testBoolPtr(&b)
			}(),
			[]byte{version, id(testBoolPtr(nil)), 0, 1},
		},
		{
			testRecPtr(nil),
			[]byte{version, id(testRecPtr(nil)), meta_nil},
		},
	}
	checkEncodedData(t, reg, items)
}

func TestSerialization_Reference(t *testing.T) {
	reg, id := registry()
	var items = []serializerTestItems{
		{
			func() any {
				var x any
				x = &x
				return x
			}(),
			[]byte{
				version,
				id((*any)(nil)), 0, // pointer type (pointer is not nil)
				meta_ref, 0b0010_0000, // pointer value (referenced value)
			},
		},
		{
			func() any {
				var x1, x2 any
				x1 = &x2
				x2 = &x1
				return x1
			}(),
			[]byte{
				version,
				id((*any)(nil)), 0, // x1
				id((*any)(nil)), 0, // x2
				meta_ref, 0b0010_0000, // referenced value
			},
		},
		{
			func() any {
				b := true
				return []*bool{&b, &b, &b}
			}(),
			[]byte{
				version,
				id(([]*bool)(nil)), 0, 0b0001_0000 | 3, 0b0001_0000,
				0, 1,
				0, meta_ref, 0b0010_0000 | 2,
				0, meta_ref, 0b0010_0000 | 2,
			},
		},
		{
			func() any {
				x := []any{nil, true, nil}
				x[0] = &x[1]
				x[2] = &x[1]
				return x
			}(),
			[]byte{
				version,
				id(([]any)(nil)), 0, 0b0001_0000 | 3,
				0b0001_0000 | 2, 0b0001_0000 | 1, 0b0001_0000 | 2, // list of ref indexes
				id((*any)(nil)), 0, id(false), 1, // x[0] = &x[1]
				meta_ref, 0b0010_0000 | 3, // x[1] = true
				meta_ref, 0b0010_0000 | 2, // x[2] = &x[1]
			},
		},
		{
			func() any {
				var x1 any
				var x2 *any
				var x3 **any
				x2 = &x1
				x3 = &x2
				x1 = x3
				return x3
			}(),
			[]byte{
				version,
				id((**any)(nil)), 0, // **any
				0,                     // *any
				meta_ref, 0b0010_0000, // ref to **any
			},
		},
		{
			func() any {
				var x1 any
				var x2 *any
				var x3 **any
				x2 = &x1
				x3 = &x2
				x1 = &x3
				return x3
			}(),
			[]byte{
				version,
				id((**any)(nil)), 0, // **any &x2
				0,                    // *any  x2 = &x1
				id((***any)(nil)), 0, // ***any x1 = &x3
				0,                         // **any = x3
				meta_ref, 0b0010_0000 | 1, // ref to x2
			},
		},
	}
	checkEncodedData(t, reg, items)
}

func checkEncodedData(t *testing.T, typeRegistry *TypeRegistry, items []serializerTestItems) {
	serializer := NewSerializer().WithTypeRegistry(typeRegistry)
	for i, item := range items {
		data := serializer.Encode(item.value)
		if !bytes.Equal(data, item.data) {
			t.Errorf("Test #%d: Encode(%v) must return %v, but actual value is %v", i+1, item.value, item.data, data)
		}
	}
}

/*

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
