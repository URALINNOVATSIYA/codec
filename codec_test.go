package codec

import (
	"bytes"
	"fmt"

	"math"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/URALINNOVATSIYA/reflex"
)

type eq func(any, any) bool

type testItem struct {
	value any
	data  []byte
	eq    eq
}

func runTests(items []testItem, typeRegistry *TypeRegistry, t *testing.T) {
	serializer := NewSerializer().WithTypeRegistry(typeRegistry)
	unserializer := NewUnserializer().WithTypeRegistry(typeRegistry)
	for i, item := range items {
		expected := item.value
		data := serializer.Encode(expected)
		if !bytes.Equal(data, item.data) {
			t.Errorf("Test #%d: Encode(%T) must return %v, but actual value is %v", i+1, expected, item.data, data)
			continue
		}
		actual, err := unserializer.Decode(data)
		if err != nil {
			t.Errorf("Test #%d: Decode(%T) raises error: %q", i+1, expected, err)
		} else if res, err := equal(item.eq, expected, actual); res == false {
			t.Errorf("Test #%d: Decode(%T) returns wrong value %T (check err: %v)", i+1, expected, actual, err)
		}
	}
}

func equal(customEq eq, expectd, actual any) (result bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%s", e)
		}
	}()
	if customEq == nil {
		return defaultEq(expectd, actual), err
	}
	return customEq(expectd, actual), err
}

func defaultEq(expected, actual any) bool {
	if expected != nil {
		switch reflect.TypeOf(expected).Kind() {
		case reflect.Chan:
			return chanEqual(expected, actual)
		case reflect.Func:
			return funcEqual(expected, actual)
		}
	}
	return reflect.DeepEqual(expected, actual)
}

func chanEqual(expected any, actual any) bool {
	if reflex.NameOf(reflect.TypeOf(expected)) != reflex.NameOf(reflect.TypeOf(actual)) {
		return false
	}
	return reflect.ValueOf(expected).Cap() == reflect.ValueOf(actual).Cap()
}

func funcEqual(expected any, actual any) bool {
	return reflex.FuncNameOf(reflect.ValueOf(expected)) == reflex.FuncNameOf(reflect.ValueOf(actual))
}

func registry() (*TypeRegistry, func(v any) byte) {
	reg := NewTypeRegistry(true)
	return reg, func(v any) byte {
		id := reg.typeIdByValue(reflect.ValueOf(v))
		return u2bs(uint64(id), 3)[0]
	}
}

func interfaceId(reg *TypeRegistry) byte {
	if id, exists := reg.typeIdByName("interface {}"); exists {
		return u2bs(uint64(id), 3)[0]
	}
	reg.RegisterType(reflect.TypeFor[any]())
	return interfaceId(reg)
}

func Test_Nil(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			nil,
			[]byte{version, typeId(nil), meta_nil},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Bool(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			false,
			[]byte{version, typeId(false), meta_fls},
			nil,
		},
		{
			true,
			[]byte{version, typeId(false), meta_tru},
			nil,
		},
		{
			testBool(false),
			[]byte{version, typeId(testBool(false)), meta_fls},
			nil,
		},
		{
			testBool(true),
			[]byte{version, typeId(testBool(false)), meta_tru},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_String(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			"",
			[]byte{version, typeId(""), c2b0(0)},
			nil,
		},
		{
			"0123456789",
			[]byte{version, typeId(""), c2b0(10), '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'},
			nil,
		},
		{
			strings.Repeat("a", 255),
			append(append([]byte{version, typeId("")}, c2b(255)...), []byte(strings.Repeat("a", 255))...),
			nil,
		},
		{
			strings.Repeat("a", 65536),
			append(append([]byte{version, typeId("")}, c2b(65536)...), []byte(strings.Repeat("a", 65536))...),
			nil,
		},
		{
			testStr("abcd"),
			[]byte{version, typeId(testStr("")), c2b0(4), 97, 98, 99, 100},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Uint8(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			uint8(0),
			[]byte{version, typeId(uint8(0)), 0},
			nil,
		},
		{
			uint8(1),
			[]byte{version, typeId(uint8(0)), 1},
			nil,
		},
		{
			uint8(255),
			[]byte{version, typeId(uint8(0)), 255},
			nil,
		},
		{
			testUint8(123),
			[]byte{version, typeId(testUint8(0)), 123},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Int8(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			int8(0),
			[]byte{version, typeId(int8(0)), 0},
			nil,
		},
		{
			int8(1),
			[]byte{version, typeId(int8(0)), 2},
			nil,
		},
		{
			int8(-1),
			[]byte{version, typeId(int8(0)), 1},
			nil,
		},
		{
			int8(127),
			[]byte{version, typeId(int8(0)), 254},
			nil,
		},
		{
			int8(-128),
			[]byte{version, typeId(int8(0)), 255},
			nil,
		},
		{
			testInt8(123),
			[]byte{version, typeId(testInt8(0)), 246},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Uint16(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			uint16(0),
			[]byte{version, typeId(uint16(0)), 0b0100_0000},
			nil,
		},
		{
			uint16(1),
			[]byte{version, typeId(uint16(0)), 0b0100_0000 | 1},
			nil,
		},
		{
			uint16(256),
			[]byte{version, typeId(uint16(0)), 0b1000_0000 | 1, 0},
			nil,
		},
		{
			uint16(65535),
			[]byte{version, typeId(uint16(0)), 0b1100_0000, 255, 255},
			nil,
		},
		{
			testUint16(12345),
			[]byte{version, typeId(testUint16(0)), 0b1000_0000 | 48, 57},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Int16(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			int16(0),
			[]byte{version, typeId(int16(0)), 0b0100_0000},
			nil,
		},
		{
			int16(1),
			[]byte{version, typeId(int16(0)), 0b0100_0000 | 2},
			nil,
		},
		{
			int16(-1),
			[]byte{version, typeId(int16(0)), 0b0100_0000 | 1},
			nil,
		},
		{
			int16(256),
			[]byte{version, typeId(int16(0)), 0b1000_0000 | 2, 0},
			nil,
		},
		{
			int16(-256),
			[]byte{version, typeId(int16(0)), 0b1000_0000 | 1, 255},
			nil,
		},
		{
			int16(32767),
			[]byte{version, typeId(int16(0)), 0b1100_0000, 255, 254},
			nil,
		},
		{
			int16(-32768),
			[]byte{version, typeId(int16(0)), 0b1100_0000, 255, 255},
			nil,
		},
		{
			testInt16(-12345),
			[]byte{version, typeId(testInt16(0)), 0b1100_0000, 96, 113},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Uint32(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			uint32(0),
			[]byte{version, typeId(uint32(0)), 0b0010_0000},
			nil,
		},
		{
			uint32(1),
			[]byte{version, typeId(uint32(0)), 0b0010_0000 | 1},
			nil,
		},
		{
			uint32(256),
			[]byte{version, typeId(uint32(0)), 0b0100_0000 | 1, 0},
			nil,
		},
		{
			uint32(123456),
			[]byte{version, typeId(uint32(0)), 0b0110_0000 | 1, 226, 64},
			nil,
		},
		{
			uint32(math.MaxUint32),
			[]byte{version, typeId(uint32(0)), 0b1010_0000, 255, 255, 255, 255},
			nil,
		},
		{
			testUint32(12345678),
			[]byte{version, typeId(testUint32(0)), 0b1000_0000, 188, 97, 78},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Int32(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			int32(0),
			[]byte{version, typeId(int32(0)), 0b0010_0000},
			nil,
		},
		{
			int32(1),
			[]byte{version, typeId(int32(0)), 0b0010_0000 | 2},
			nil,
		},
		{
			int32(-1),
			[]byte{version, typeId(int32(0)), 0b0010_0000 | 1},
			nil,
		},
		{
			int32(256),
			[]byte{version, typeId(int32(0)), 0b0100_0000 | 2, 0},
			nil,
		},
		{
			int32(-256),
			[]byte{version, typeId(int32(0)), 0b0100_0000 | 1, 255},
			nil,
		},
		{
			int32(123456),
			[]byte{version, typeId(int32(0)), 0b0110_0000 | 3, 196, 128},
			nil,
		},
		{
			int32(-123456),
			[]byte{version, typeId(int32(0)), 0b0110_0000 | 3, 196, 127},
			nil,
		},
		{
			int32(math.MaxInt32),
			[]byte{version, typeId(int32(0)), 0b1010_0000, 255, 255, 255, 254},
			nil,
		},
		{
			int32(math.MinInt32),
			[]byte{version, typeId(int32(0)), 0b1010_0000, 255, 255, 255, 255},
			nil,
		},
		{
			testInt32(1234567),
			[]byte{version, typeId(testInt32(0)), 0b1000_0000, 37, 173, 14},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Uint64(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			uint64(0),
			[]byte{version, typeId(uint64(0)), 0b0001_0000},
			nil,
		},
		{
			uint64(1),
			[]byte{version, typeId(uint64(0)), 0b0001_0000 | 1},
			nil,
		},
		{
			uint64(1 << 8),
			[]byte{version, typeId(uint64(0)), 0b0010_0000 | 1, 0},
			nil,
		},
		{
			uint64(1 << 16),
			[]byte{version, typeId(uint64(0)), 0b0011_0000 | 1, 0, 0},
			nil,
		},
		{
			uint64(1 << 24),
			[]byte{version, typeId(uint64(0)), 0b0100_0000 | 1, 0, 0, 0},
			nil,
		},
		{
			uint64(1 << 32),
			[]byte{version, typeId(uint64(0)), 0b0101_0000 | 1, 0, 0, 0, 0},
			nil,
		},
		{
			uint64(1 << 40),
			[]byte{version, typeId(uint64(0)), 0b0110_0000 | 1, 0, 0, 0, 0, 0},
			nil,
		},
		{
			uint64(1 << 48),
			[]byte{version, typeId(uint64(0)), 0b0111_0000 | 1, 0, 0, 0, 0, 0, 0},
			nil,
		},
		{
			uint64(1 << 56),
			[]byte{version, typeId(uint64(0)), 0b1000_0000 | 1, 0, 0, 0, 0, 0, 0, 0},
			nil,
		},
		{
			uint64(1<<64 - 1),
			[]byte{version, typeId(uint64(0)), 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
			nil,
		},
		{
			testUint64(1234567890),
			[]byte{version, typeId(testUint64(0)), 0b0101_0000, 73, 150, 2, 210},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Int64(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			int64(0),
			[]byte{version, typeId(int64(0)), 0b0001_0000},
			nil,
		},
		{
			int64(1),
			[]byte{version, typeId(int64(0)), 0b0001_0000 | 2},
			nil,
		},
		{
			int64(-1),
			[]byte{version, typeId(int64(0)), 0b0001_0000 | 1},
			nil,
		},
		{
			int64(1 << 8),
			[]byte{version, typeId(int64(0)), 0b0010_0000 | 2, 0},
			nil,
		},
		{
			int64(-1 << 8),
			[]byte{version, typeId(int64(0)), 0b0010_0000 | 1, 255},
			nil,
		},
		{
			int64(1 << 16),
			[]byte{version, typeId(int64(0)), 0b0011_0000 | 2, 0, 0},
			nil,
		},
		{
			int64(-1 << 16),
			[]byte{version, typeId(int64(0)), 0b0011_0000 | 1, 255, 255},
			nil,
		},
		{
			int64(1 << 24),
			[]byte{version, typeId(int64(0)), 0b0100_0000 | 2, 0, 0, 0},
			nil,
		},
		{
			int64(-1 << 24),
			[]byte{version, typeId(int64(0)), 0b0100_0000 | 1, 255, 255, 255},
			nil,
		},
		{
			int64(1 << 32),
			[]byte{version, typeId(int64(0)), 0b0101_0000 | 2, 0, 0, 0, 0},
			nil,
		},
		{
			int64(-1 << 32),
			[]byte{version, typeId(int64(0)), 0b0101_0000 | 1, 255, 255, 255, 255},
			nil,
		},
		{
			int64(1 << 40),
			[]byte{version, typeId(int64(0)), 0b0110_0000 | 2, 0, 0, 0, 0, 0},
			nil,
		},
		{
			int64(-1 << 40),
			[]byte{version, typeId(int64(0)), 0b0110_0000 | 1, 255, 255, 255, 255, 255},
			nil,
		},
		{
			int64(1 << 48),
			[]byte{version, typeId(int64(0)), 0b0111_0000 | 2, 0, 0, 0, 0, 0, 0},
			nil,
		},
		{
			int64(-1 << 48),
			[]byte{version, typeId(int64(0)), 0b0111_0000 | 1, 255, 255, 255, 255, 255, 255},
			nil,
		},
		{
			int64(1 << 56),
			[]byte{version, typeId(int64(0)), 0b1000_0000 | 2, 0, 0, 0, 0, 0, 0, 0},
			nil,
		},
		{
			int64(-1 << 56),
			[]byte{version, typeId(int64(0)), 0b1000_0000 | 1, 255, 255, 255, 255, 255, 255, 255},
			nil,
		},
		{
			int64(math.MaxInt64),
			[]byte{version, typeId(int64(0)), 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 254},
			nil,
		},
		{
			int64(math.MinInt64),
			[]byte{version, typeId(int64(0)), 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
			nil,
		},
		{
			testInt64(1234567890),
			[]byte{version, typeId(testInt64(0)), 0b0101_0000, 147, 44, 5, 164},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Uint(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			uint(0),
			[]byte{version, typeId(uint(0)), 0b0001_0000},
			nil,
		},
		{
			uint(1),
			[]byte{version, typeId(uint(0)), 0b0001_0000 | 1},
			nil,
		},
		{
			uint(1<<8 - 1),
			[]byte{version, typeId(uint(0)), 0b0010_0000, 255},
			nil,
		},
		{
			uint(1 << 8),
			[]byte{version, typeId(uint(0)), 0b0010_0000 | 1, 0},
			nil,
		},
		{
			uint(1<<16 - 1),
			[]byte{version, typeId(uint(0)), 0b0011_0000, 255, 255},
			nil,
		},
		{
			uint(1 << 16),
			[]byte{version, typeId(uint(0)), 0b0011_0000 | 1, 0, 0},
			nil,
		},
		{
			uint(1<<24 - 1),
			[]byte{version, typeId(uint(0)), 0b0100_0000, 255, 255, 255},
			nil,
		},
		{
			uint(1 << 24),
			[]byte{version, typeId(uint(0)), 0b0100_0000 | 1, 0, 0, 0},
			nil,
		},
		{
			uint(1<<32 - 1),
			[]byte{version, typeId(uint(0)), 0b0101_0000, 255, 255, 255, 255},
			nil,
		},
		{
			testUint(12345),
			[]byte{version, typeId(testUint(0)), 0b0011_0000, 48, 57},
			nil,
		},
	}
	if math.MaxUint == math.MaxUint64 {
		items = append(items, []testItem{
			{
				uint(1 << 32),
				[]byte{version, typeId(uint(0)), 0b0101_0000 | 1, 0, 0, 0, 0},
				nil,
			},
			{
				uint(1<<40 - 1),
				[]byte{version, typeId(uint(0)), 0b0110_0000, 255, 255, 255, 255, 255},
				nil,
			},
			{
				uint(1 << 40),
				[]byte{version, typeId(uint(0)), 0b0110_0000 | 1, 0, 0, 0, 0, 0},
				nil,
			},
			{
				uint(1<<48 - 1),
				[]byte{version, typeId(uint(0)), 0b0111_0000, 255, 255, 255, 255, 255, 255},
				nil,
			},
			{
				uint(1 << 48),
				[]byte{version, typeId(uint(0)), 0b0111_0000 | 1, 0, 0, 0, 0, 0, 0},
				nil,
			},
			{
				uint(1<<56 - 1),
				[]byte{version, typeId(uint(0)), 0b1000_0000, 255, 255, 255, 255, 255, 255, 255},
				nil,
			},
			{
				uint(1 << 56),
				[]byte{version, typeId(uint(0)), 0b1000_0000 | 1, 0, 0, 0, 0, 0, 0, 0},
				nil,
			},
			{
				uint(1<<64 - 1),
				[]byte{version, typeId(uint(0)), 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
				nil,
			},
		}...)
	}
	runTests(items, reg, t)
}

func Test_Int(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			0,
			[]byte{version, typeId(0), 0b0001_0000},
			nil,
		},
		{
			1<<7 - 1,
			[]byte{version, typeId(0), 0b0010_0000, 254},
			nil,
		},
		{
			-1 << 7,
			[]byte{version, typeId(0), 0b0010_0000, 255},
			nil,
		},
		{
			1 << 7,
			[]byte{version, typeId(0), 0b0010_0000 | 1, 0},
			nil,
		},
		{
			-1<<7 - 1,
			[]byte{version, typeId(0), 0b0010_0000 | 1, 1},
			nil,
		},
		{
			1<<15 - 1,
			[]byte{version, typeId(0), 0b0011_0000, 255, 254},
			nil,
		},
		{
			-1 << 15,
			[]byte{version, typeId(0), 0b0011_0000, 255, 255},
			nil,
		},
		{
			1 << 15,
			[]byte{version, typeId(0), 0b0011_0000 | 1, 0, 0},
			nil,
		},
		{
			-1<<15 - 1,
			[]byte{version, typeId(0), 0b0011_0000 | 1, 0, 1},
			nil,
		},
		{
			1<<23 - 1,
			[]byte{version, typeId(0), 0b0100_0000, 255, 255, 254},
			nil,
		},
		{
			-1 << 23,
			[]byte{version, typeId(0), 0b0100_0000, 255, 255, 255},
			nil,
		},
		{
			1 << 23,
			[]byte{version, typeId(0), 0b0100_0000 | 1, 0, 0, 0},
			nil,
		},
		{
			-1<<23 - 1,
			[]byte{version, typeId(0), 0b0100_0000 | 1, 0, 0, 1},
			nil,
		},
		{
			testInt(12345),
			[]byte{version, typeId(testInt(0)), 0b0011_0000, 96, 114},
			nil,
		},
	}
	if math.MaxInt == math.MaxInt64 {
		items = append(items, []testItem{
			{
				1<<32 - 1,
				[]byte{version, typeId(0), 0b0101_0000 | 1, 255, 255, 255, 254},
				nil,
			},
			{
				-1 << 32,
				[]byte{version, typeId(0), 0b0101_0000 | 1, 255, 255, 255, 255},
				nil,
			},
			{
				1 << 32,
				[]byte{version, typeId(0), 0b0101_0000 | 2, 0, 0, 0, 0},
				nil,
			},
			{
				-1<<32 - 1,
				[]byte{version, typeId(0), 0b0101_0000 | 2, 0, 0, 0, 1},
				nil,
			},
			{
				1<<40 - 1,
				[]byte{version, typeId(0), 0b0110_0000 | 1, 255, 255, 255, 255, 254},
				nil,
			},
			{
				-1 << 40,
				[]byte{version, typeId(0), 0b0110_0000 | 1, 255, 255, 255, 255, 255},
				nil,
			},
			{
				1 << 40,
				[]byte{version, typeId(0), 0b0110_0000 | 2, 0, 0, 0, 0, 0},
				nil,
			},
			{
				-1<<40 - 1,
				[]byte{version, typeId(0), 0b0110_0000 | 2, 0, 0, 0, 0, 1},
				nil,
			},
			{
				1<<48 - 1,
				[]byte{version, typeId(0), 0b0111_0000 | 1, 255, 255, 255, 255, 255, 254},
				nil,
			},
			{
				-1 << 48,
				[]byte{version, typeId(0), 0b0111_0000 | 1, 255, 255, 255, 255, 255, 255},
				nil,
			},
			{
				1 << 48,
				[]byte{version, typeId(0), 0b0111_0000 | 2, 0, 0, 0, 0, 0, 0},
				nil,
			},
			{
				-1<<48 - 1,
				[]byte{version, typeId(0), 0b0111_0000 | 2, 0, 0, 0, 0, 0, 1},
				nil,
			},
			{
				1<<56 - 1,
				[]byte{version, typeId(0), 0b1000_0000 | 1, 255, 255, 255, 255, 255, 255, 254},
				nil,
			},
			{
				-1 << 56,
				[]byte{version, typeId(0), 0b1000_0000 | 1, 255, 255, 255, 255, 255, 255, 255},
				nil,
			},
			{
				1 << 56,
				[]byte{version, typeId(0), 0b1000_0000 | 2, 0, 0, 0, 0, 0, 0, 0},
				nil,
			},
			{
				-1<<56 - 1,
				[]byte{version, typeId(0), 0b1000_0000 | 2, 0, 0, 0, 0, 0, 0, 1},
				nil,
			},
			{
				math.MaxInt,
				[]byte{version, typeId(0), 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 254},
				nil,
			},
			{
				math.MinInt,
				[]byte{version, typeId(0), 0b1001_0000, 255, 255, 255, 255, 255, 255, 255, 255},
				nil,
			},
		}...)
	}
	runTests(items, reg, t)
}

func Test_Uintptr(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			uintptr(123456),
			[]byte{version, typeId(uintptr(0)), 0b0011_0000 | 1, 226, 64},
			nil,
		},
		{
			testUintptr(12345),
			[]byte{version, typeId(testUintptr(0)), 0b0011_0000, 48, 57},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Float32(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			float32(0),
			[]byte{version, typeId(float32(0)), 0b0010_0000},
			nil,
		},
		{
			float32(1),
			[]byte{version, typeId(float32(0)), 0b0110_0000, 128, 63},
			nil,
		},
		{
			float32(10),
			[]byte{version, typeId(float32(0)), 0b0110_0000, 32, 65},
			nil,
		},
		{
			float32(-1),
			[]byte{version, typeId(float32(0)), 0b0110_0000, 128, 191},
			nil,
		},
		{
			float32(-10),
			[]byte{version, typeId(float32(0)), 0b0110_0000, 32, 193},
			nil,
		},
		{
			float32(1.23),
			[]byte{version, typeId(float32(0)), 0b1010_0000, 164, 112, 157, 63},
			nil,
		},
		{
			float32(-1.23),
			[]byte{version, typeId(float32(0)), 0b1010_0000, 164, 112, 157, 191},
			nil,
		},
		{
			testFloat32(123),
			[]byte{version, typeId(testFloat32(0)), 0b0110_0000, 246, 66},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Float64(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			float64(0),
			[]byte{version, typeId(float64(0)), 0b0001_0000},
			nil,
		},
		{
			float64(1),
			[]byte{version, typeId(float64(0)), 0b0011_0000, 240, 63},
			nil,
		},
		{
			float64(10),
			[]byte{version, typeId(float64(0)), 0b0011_0000, 36, 64},
			nil,
		},
		{
			float64(-1),
			[]byte{version, typeId(float64(0)), 0b0011_0000, 240, 191},
			nil,
		},
		{
			float64(-10),
			[]byte{version, typeId(float64(0)), 0b0011_0000, 36, 192},
			nil,
		},
		{
			1.23,
			[]byte{version, typeId(float64(0)), 0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 63},
			nil,
		},
		{
			-1.23,
			[]byte{version, typeId(float64(0)), 0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 191},
			nil,
		},
		{
			testFloat64(123),
			[]byte{version, typeId(testFloat64(0)), 0b0100_0000, 192, 94, 64},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Complex64(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			complex(float32(0), float32(0)),
			[]byte{version, typeId(complex64(0)), 0b0010_0000, 0b0010_0000},
			nil,
		},
		{
			complex(float32(1), float32(0)),
			[]byte{version, typeId(complex64(0)), 0b0110_0000, 128, 63, 0b0010_0000},
			nil,
		},
		{
			complex(float32(0), float32(1)),
			[]byte{version, typeId(complex64(0)), 0b0010_0000, 0b0110_0000, 128, 63},
			nil,
		},
		{
			complex(float32(1.23), float32(-1.23)),
			[]byte{version, typeId(complex64(0)), 0b1010_0000, 164, 112, 157, 63, 0b1010_0000, 164, 112, 157, 191},
			nil,
		},
		{
			testComplex64(1 + 2i),
			[]byte{version, typeId(testComplex64(0)), 0b0110_0000, 128, 63, 0b0100_0000, 64},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Complex128(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			complex(float64(0), float64(0)),
			[]byte{version, typeId(complex128(0)), 0b0001_0000, 0b0001_0000},
			nil,
		},
		{
			complex(float64(1), float64(0)),
			[]byte{version, typeId(complex128(0)), 0b0011_0000, 240, 63, 0b0001_0000},
			nil,
		},
		{
			complex(float64(0), float64(1)),
			[]byte{version, typeId(complex128(0)), 0b0001_0000, 0b0011_0000, 240, 63},
			nil,
		},
		{
			complex(1.23, -1.23),
			[]byte{
				version, typeId(complex128(0)),
				0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 63,
				0b1001_0000, 174, 71, 225, 122, 20, 174, 243, 191,
			},
			nil,
		},
		{
			testComplex128(1 + 2i),
			[]byte{version, typeId(testComplex128(0)), 0b0011_0000, 240, 63, 0b0010_0000, 64},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_UnsafePointer(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			unsafe.Pointer(nil),
			[]byte{version, typeId(unsafe.Pointer(nil)), meta_nil},
			nil,
		},
		{
			unsafe.Pointer(uintptr(0)),
			[]byte{version, typeId(unsafe.Pointer(nil)), meta_nil},
			nil,
		},
		{
			unsafe.Pointer(uintptr(123456)),
			[]byte{version, typeId(unsafe.Pointer(nil)), 0b0011_0000 | 1, 226, 64},
			nil,
		},
		{
			testUnsafePointer(nil),
			[]byte{version, typeId(testUnsafePointer(nil)), meta_nil},
			nil,
		},
		{
			testUnsafePointer(uintptr(12345)),
			[]byte{version, typeId(testUnsafePointer(nil)), 0b0011_0000, 48, 57},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Chan(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			(<-chan bool)(nil),
			[]byte{version, typeId(make(<-chan bool)), meta_nil},
			nil,
		},
		{
			make(chan int),
			[]byte{version, typeId(make(chan int)), meta_nonil, c2b0(0)},
			nil,
		},
		{
			make(chan<- bool, 1),
			[]byte{version, typeId(make(chan<- bool)), meta_nonil, c2b0(1)},
			nil,
		},
		{
			make(testChan, 10),
			[]byte{version, typeId(testChan(nil)), meta_nonil, c2b0(10)},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Func(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			(func(byte, bool) int8)(nil),
			[]byte{version, typeId((func(byte, bool) int8)(nil)), meta_nil},
			nil,
		},
		{
			registry,
			[]byte{version, typeId(registry), meta_nonil},
			nil,
		},
		{
			math.Abs,
			[]byte{version, typeId(math.Abs), meta_nonil},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_ReferenceToTheSameValue(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			func() any {
				s := testS3{}
				s.F1 = "abc"
				s.F3 = "abc"
				return s
			}(),
			[]byte{
				version,
				typeId(testS3{}), meta_strc, // testS3 header
				typeId(""), c2b0(3), 'a', 'b', 'c', // testS3.F1 (id = 1)
				typeId(nil), meta_nil, // testS3.F2 (id = 4)
				typeId(""), meta_ref, c2b0(3), // testS3.F3 (id = 7)
			},
			nil,
		},
		// #2
		{
			func() any {
				s := testS3{}
				s.F1 = registry
				s.F2 = registry
				return s
			}(),
			[]byte{
				version,
				typeId(testS3{}), meta_strc, // testS3 header
				typeId(registry), meta_nonil, // testS3.F1 (id = 1)
				typeId(registry), meta_ref, c2b0(3), // testS3.F2 (id = 4)
				typeId(nil), meta_nil, // testS3.F3 (id = 7)
			},
			func(expected, actual any) bool {
				e := expected.(testS3)
				a := actual.(testS3)
				return funcEqual(e.F1, a.F1) && funcEqual(e.F2, a.F2) && a.F3 == nil
			},
		},
		// #3
		{
			func() any {
				ch := make(chan<- byte, 15)
				s := testS3{}
				s.F2 = ch
				s.F3 = ch
				return s
			}(),
			[]byte{
				version,
				typeId(testS3{}), meta_strc, // testS3 header
				typeId(nil), meta_nil, // testS3.F1 (id = 1)
				typeId((chan<- byte)(nil)), meta_nonil, c2b0(15), // testS3.F2 (id = 4)
				typeId((chan<- byte)(nil)), meta_ref, c2b0(6), // testS3.F3 (id = 7)
			},
			func(expected, actual any) bool {
				e := expected.(testS3)
				a := actual.(testS3)
				return chanEqual(e.F2, a.F2) && chanEqual(e.F3, a.F3) && a.F1 == nil
			},
		},
	}
	runTests(items, reg, t)
}

func Test_PointerToSingleValue(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			(*any)(nil),
			[]byte{version, typeId((*any)(nil)), meta_nil},
			nil,
		},
		// #2
		{
			(*byte)(nil),
			[]byte{version, typeId((*byte)(nil)), meta_nil},
			nil,
		},
		// #3
		{
			func() any {
				b := true
				return &b
			}(),
			[]byte{version, typeId((*bool)(nil)), meta_nonil, meta_tru},
			nil,
		},
		// #4
		{
			func() any {
				b := byte(255)
				return &b
			}(),
			[]byte{version, typeId((*byte)(nil)), meta_nonil, 255},
			nil,
		},
		// #5
		{
			func() any {
				s := "123"
				return &s
			}(),
			[]byte{version, typeId((*string)(nil)), meta_nonil, 0b0001_0000 | 3, '1', '2', '3'},
			nil,
		},
		// #6
		{
			func() any {
				var x any = true
				return &x
			}(),
			[]byte{version, typeId((*any)(nil)), meta_nonil, typeId(false), meta_tru},
			nil,
		},
		// #7
		{
			func() any {
				b := true
				return testBoolPtr(&b)
			}(),
			[]byte{version, typeId(testBoolPtr(nil)), meta_nonil, meta_tru},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_PointersToTheSameValue(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			func() any {
				b1 := byte(1)
				b2 := byte(1)
				s := &testS3{}
				s.F1 = &b1
				s.F2 = &b2
				s.F3 = &b1
				return s
			}(),
			[]byte{
				version, typeId((*testS3)(nil)), meta_nonil, meta_strc, // *testS3
				typeId((*byte)(nil)), meta_nonil, 1, // testS3.F1
				typeId((*byte)(nil)), meta_nonil, 1, // testS3.F2
				typeId((*byte)(nil)), meta_ref, c2b0(4), // testS3.F3 is ref to F1 value
			},
			nil,
		},
		// #2
		{
			func() any {
				b := "abc"
				s := &testS3{}
				s.F2 = &b
				s.F3 = &b
				return s
			}(),
			[]byte{
				version, typeId((*testS3)(nil)), meta_nonil, meta_strc, // *testS3
				typeId(nil), meta_nil, // testS3.F1
				typeId((*string)(nil)), meta_nonil, c2b0(3), 'a', 'b', 'c', // testS3.F2
				typeId((*string)(nil)), meta_ref, c2b0(7), // testS3.F3 is ref to F2 value
			},
			nil,
		},
		// #3
		{
			func() any {
				s := &testS3{}
				s.F1 = s
				s.F2 = s
				return s
			}(),
			[]byte{
				version, typeId((*testS3)(nil)), meta_nonil, meta_strc, // *testS3
				typeId((*testS3)(nil)), meta_ref, c2b0(0), // testS3.F1 (id = 2) is ref to struct
				typeId((*testS3)(nil)), meta_ref, c2b0(0), // testS3.F2 (id = 5) is ref to struct
				typeId(nil), meta_nil, // testS3.F3
			},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_PointerChain(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			func() any {
				var x1, x2 any
				y := byte(111)
				x1 = &x2
				x2 = &y
				return x1
			}(),
			[]byte{
				version,
				typeId((*any)(nil)), meta_nonil, // x1
				typeId((*byte)(nil)), meta_nonil, 111, // x2
			},
			nil,
		},
		// #2
		{
			func() any {
				var x1, x2, x3 any
				y := byte(111)
				x1 = &x2
				x2 = &x3
				x3 = &y
				return x1
			}(),
			[]byte{
				version,
				typeId((*any)(nil)), meta_nonil, // x1
				typeId((*any)(nil)), meta_nonil, // x2
				typeId((*byte)(nil)), meta_nonil, 111, // x3
			},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_CyclicPointerChain(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			func() any {
				var x any
				x = &x
				return x
			}(),
			[]byte{version, typeId((*any)(nil)), meta_nonil, typeId((*any)(nil)), meta_ref, c2b0(0)},
			func(_, actual any) bool {
				v := actual.(*any)
				return v == *v
			},
		},
		// #2
		{
			func() any {
				var x1, x2 any
				x1 = &x2
				x2 = &x1
				return x1
			}(),
			[]byte{
				version,
				typeId((*any)(nil)), meta_nonil, // x1
				typeId((*any)(nil)), meta_nonil, typeId((*any)(nil)), meta_ref, c2b0(0), // x2
			},
			func(_, actual any) bool {
				x1 := actual.(*any)
				x2 := (*x1).(*any)
				return *x2 == x1 && x2 != x1
			},
		},
		// #3
		{
			func() any {
				var x1, x2, x3 any
				x1 = &x2
				x2 = &x3
				x3 = &x1
				return x1
			}(),
			[]byte{
				version,
				typeId((*any)(nil)), meta_nonil, // x1
				typeId((*any)(nil)), meta_nonil, // x2
				typeId((*any)(nil)), meta_nonil, typeId((*any)(nil)), meta_ref, c2b0(0), // x3
			},
			func(_, actual any) bool {
				x1 := actual.(*any)
				x2 := (*x1).(*any)
				x3 := (*x2).(*any)
				return *x3 == x1 && x1 != x2 && x2 != x3
			},
		},
		// #4
		{
			func() any {
				var x testRecPtr
				x = &x
				return x
			}(),
			[]byte{version, typeId(testRecPtr(nil)), meta_nonil, meta_ref, c2b0(0)},
			func(_, actual any) bool {
				v := actual.(testRecPtr)
				return v == *v
			},
		},
		// #5
		{
			func() any {
				var x1, x2 testRecPtr
				x1 = &x2
				x2 = &x1
				return x1
			}(),
			[]byte{
				version,
				typeId(testRecPtr(nil)), meta_nonil, // x1
				meta_nonil, meta_ref, c2b0(0), // x2
			},
			func(_, actual any) bool {
				x1 := actual.(testRecPtr)
				x2 := *x1
				return *x2 == x1 && x2 != x1
			},
		},
		// #6
		{
			func() any {
				var x1, x2, x3 testRecPtr
				x1 = &x2
				x2 = &x3
				x3 = &x1
				return x1
			}(),
			[]byte{
				version,
				typeId(testRecPtr(nil)), meta_nonil, // x1
				meta_nonil,                    // x2
				meta_nonil, meta_ref, c2b0(0), // x3
			},
			func(_, actual any) bool {
				x1 := actual.(testRecPtr)
				x2 := *x1
				x3 := *x2
				return *x3 == x1 && x1 != x2 && x2 != x3
			},
		},
		// #7
		{
			func() any {
				var x1 any
				var x2 *any
				var x3 **any
				x3 = &x2
				x2 = &x1
				x1 = x3
				return x3
			}(),
			[]byte{
				version,
				typeId((**any)(nil)), meta_nonil, // x3
				meta_nonil,        // x2
				typeId((**any)(nil)), meta_ref, c2b0(0), // x1 == x3
			},
			func(_, actual any) bool {
				x3 := actual.(**any)
				x2 := *x3
				x1 := *x2
				return x1 == x3
			},
		},
		// #8
		{
			func() any {
				var x1 any
				var x2 *any
				var x3 **any
				x3 = &x2
				x2 = &x1
				x1 = &x3
				return x3
			}(),
			[]byte{
				version,
				typeId((**any)(nil)), meta_nonil, // x3
				meta_nonil,                                           // x2
				typeId((***any)(nil)), meta_nonil, meta_ref, c2b0(0), // x1 == x3
			},
			func(_, actual any) bool {
				x3 := actual.(**any)
				x2 := *x3
				x1 := *x2
				return *x1.(***any) == x3
			},
		},
	}
	runTests(items, reg, t)
}

/*func Test_BackwardPointerToContainer(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			func() any {
				s := &testS3{}
				s.F1 = true
				s.F2 = &s.F1
				s.F3 = &s.F1
				return s
			}(),
			[]byte{
				version, typeId((*testS3)(nil)), meta_nonil, meta_aggr, // *testS3
				typeId(false), meta_tru, // F1
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(2), // F2 is *F1
				meta_ref, c2b0(7), // f3 is *F1
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3)
				s.F1 = false
				return *s.F2.(*any) == false && *s.F3.(*any) == false
			},
		},
		// #2
		{
			func() any {
				b := true
				s := &testS3Bool{}
				s.F1 = &b
				s.F2 = &s.F1
				s.F3 = &s.F1
				return s
			}(),
			[]byte{
				version, typeId((*testS3Bool)(nil)), meta_nonil, meta_aggr, // *testS3Bool
				meta_nonil, meta_tru, // F1
				typeId((**bool)(nil)), meta_nonil, meta_ref, c2b0(2), // F2 is *F1
				meta_ref, c2b0(7), // F3 is *F1
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3Bool)
				s.F1 = nil
				return *s.F2.(**bool) == nil && *s.F3.(**bool) == nil
			},
		},
		// #3
		{
			func() any {
				var x any = true
				s := &testS3Any{}
				s.F1 = &x
				s.F2 = &s.F1
				s.F3 = &s.F1
				return s
			}(),
			[]byte{
				version, typeId((*testS3Any)(nil)), meta_nonil, meta_aggr, // *testS3Any
				meta_nonil, typeId(false), meta_tru, // F1
				typeId((**any)(nil)), meta_nonil, meta_ref, c2b0(2), // F2 is *F1
				meta_ref, c2b0(8), // F3 is *F1
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3Any)
				*s.F1 = byte(123)
				return **s.F2.(**any) == *s.F1 && **s.F3.(**any) == *s.F1
			},
		},
		// #4
		{
			func() any {
				s := &testS3{}
				s.F1 = s
				s.F2 = &s.F1
				s.F3 = &s.F1
				return s
			}(),
			[]byte{
				version, typeId((*testS3)(nil)), meta_nonil, meta_aggr, // *testS3
				meta_ref, c2b0(0), // F1 is ref to s
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(2), // F2 is *F1
				meta_ref, c2b0(6), // F3 is ref to F2 value
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3)
				s.F1 = byte(123)
				return *s.F2.(*any) == s.F1 && *s.F3.(*any) == s.F1
			},
		},
		// #5
		{
			func() any {
				s := &testS3{}
				s.F2 = &s.F2
				return s
			}(),
			[]byte{
				version, typeId((*testS3)(nil)), meta_nonil, meta_aggr, // *testS3
				typeId(nil), meta_nil, // F1
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(5), // F2 is ref to F2
				typeId(nil), meta_nil, // F3
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3)
				return *s.F2.(*any) == s.F2 && s.F2 != s
			},
		},
	}
	runTests(items, reg, t)
}*/

func Test_ForwardPointerToContainer(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			func() any {
				s := &testS1{}
				s.F1 = s
				x := &s.F1
				return x
			}(),
			[]byte{
				version, typeId((*any)(nil)), meta_nonil, // *any
				meta_ref, c2b0(4), // ref to s.F1
				typeId((*testS1)(nil)), meta_nonil, // *testS1
				meta_strc, // testS1
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				x := actual.(*any)
				s := (*x).(*testS1)
				return s.F1 == s && x == &s.F1
			},
		},
		// #2
		{
			func() any {
				s := &testS4{}
				s.F1 = &s.F3
				s.F2 = &s.F3
				s.F3 = true
				s.F4 = &s.F3
				return s
			}(),
			[]byte{
				version, typeId((*testS4)(nil)), meta_nonil, // *testS4
				meta_strc,                                           // testS4
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(10), // s.F1 is *s.F3
				typeId(false), meta_tru, // s.F3 is true
				typeId((*any)(nil)), meta_ref, c2b0(4), // s.F2
				typeId((*any)(nil)), meta_ref, c2b0(4), // s.F4
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := (actual).(*testS4)
				return s.F1 == &s.F3 && s.F2 == &s.F3 && s.F4 == &s.F3
			},
		},
		// #3
		{
			func() any {
				s := &testS4{}
				s.F1 = &s.F3
				s.F2 = &s.F3
				s.F3 = &s.F1
				s.F4 = &s.F3
				return s
			}(),
			[]byte{
				version, typeId((*testS4)(nil)), meta_nonil, // *testS4
				meta_strc,                                           // testS4
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(11), // s.F1 is *s.F3
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(2), // s.F3 is true
				typeId((*any)(nil)), meta_ref, c2b0(4), // s.F2
				typeId((*any)(nil)), meta_ref, c2b0(4), // s.F4
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := (actual).(*testS4)
				return s.F1 == &s.F3 && s.F2 == &s.F3 && s.F4 == &s.F3 && s.F3 == &s.F1
			},
		},
		// #4
		{
			func() any {
				s := &testS2{}
				x := &s.F2
				y := &s.F1
				s.F1 = &x
				s.F2 = &y
				return s
			}(),
			[]byte{
				version, typeId((*testS2)(nil)), meta_nonil, // *testS2
				meta_strc,                                                        // testS2
				typeId((**any)(nil)), meta_nonil, meta_nonil, meta_ref, c2b0(10), // F1
				typeId((**any)(nil)), meta_nonil, meta_nonil, meta_ref, c2b0(2), // F2
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS2)
				return **s.F1.(**any) == s.F2 && **s.F2.(**any) == s.F1
			},
		},
		// #3
		/*{
			func() any {
				s1 := &testS1{}
				s1.F1 = s1
				s2 := &testS2{}
				s2.F2 = &s1.F1
				x := &s2.F2
				s2.F1 = &x
				return s2
			}(),
			[]byte{
				version, typeId((*testS2)(nil)), meta_nonil, meta_aggr, // *testS2
				typeId((**any)(nil)), meta_nonil, meta_nonil, meta_ref, c2b0(6), // s2.F1 is ref to s2.F2
				typeId((*any)(nil)), meta_nonil, // s2.F2
				meta_cntr, interfaceId(reg), c2b0(12), // s1.F1
				typeId((*testS1)(nil)), meta_nonil, meta_aggr, // *testS1
				meta_ref, c2b0(12), // ref to s1.F1
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s2 := actual.(*testS2)
				x := s2.F2.(*any)
				s1 := (*x).(*testS1)
				return s1.F1 == s1 && x == &s1.F1 && **s2.F1.(**any) == s2.F2 && s2.F2 == x
			},
		},*/
		// #4
		/*{
			func() any {
				s1 := &testS3{}
				s2 := &testS3{}
				s1.F1 = s1
				s1.F2 = &s1.F3
				s1.F3 = &s2.F3
				s2.F1 = &s2.F2
				s2.F2 = &s1.F2
				s2.F3 = &s1.F1
				return s2
			}(),
			[]byte{
				version, typeId((*testS3)(nil)), meta_nonil, meta_aggr, // *testS3
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(5),     // s2.F1 is ref to s2.F2
				typeId((*any)(nil)), meta_nonil,                        // s2.F2 is *s1.F2
				meta_cntr, interfaceId(reg), c2b(18)[0], c2b(18)[1],    // s1.F2 is *s1.F3
				typeId((*testS3)(nil)), meta_nonil, meta_aggr,          // *testS3
				meta_ref, c2b(19)[0], c2b(19)[1],                       // ref to s1.F1
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				return true
			},
		},*/
		// #2
		/*{
			func() any {
				s := &testStruct2{}
				x := &s.f3
				s.f1 = &x
				s.f2 = &x
				return s
			}(),
			[]byte{
				version, typeId((*testStruct2)(nil)), meta_nonil, meta_aggr, // *testStruct2
				typeId((**any)(nil)), meta_nonil, meta_nonil, meta_ref, c2b0(8), // f1 is ref to f3 (id = 2)
				meta_ref, c2b0(4), // f2 (id = 6)
				typeId(nil), meta_nil, // f3 (id = 8)
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testStruct2)
				s.f3 = byte(123)
				return **s.f1.(**any) == s.f3 && **s.f2.(**any) == s.f3 && s.f3 == byte(123)
			},
		},
		// #3
		{
			func() any {
				s := &testStruct2{}
				x := &s.f2
				s.f1 = &x
				s.f3 = &x
				return s
			}(),
			[]byte{
				version, typeId((*testStruct2)(nil)), meta_nonil, meta_aggr, // *testStruct2
				typeId((**any)(nil)), meta_nonil, meta_nonil, meta_ref, c2b0(6), // f1 is ref to f2 (id = 2)
				typeId(nil), meta_nil, // f2 (id = 6)
				meta_ref, c2b0(4), // f3 (id = 8)

			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testStruct2)
				s.f2 = byte(123)
				return **s.f1.(**any) == s.f2 && **s.f3.(**any) == s.f2 && s.f2 == byte(123)
			},
		},
		// #4
		{
			func() any {
				s := &testStruct2{}
				s.f1 = &s.f3
				return s
			}(),
			[]byte{
				version, typeId((*testStruct2)(nil)), meta_nonil, meta_aggr, // *testStruct2
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(8), // f1 is ref to f3 (id = 2)
				typeId(nil), meta_nil, // f2 (id = 5)
				typeId(nil), meta_nil, // f3 (id = 8)
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testStruct2)
				s.f3 = byte(123)
				return *s.f1.(*any) == s.f3 && s.f3 == byte(123)
			},
		},
		// #5
		{
			func() any {
				s := &testStruct2{}
				s.f1 = &s.f3
				s.f2 = &s.f3
				return s
			}(),
			[]byte{
				version, typeId((*testStruct2)(nil)), meta_nonil, meta_aggr, // *testStruct2
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(7), // f1 is ref to f3 (id = 2)
				meta_ref, c2b0(4), // f2 (id = 5)
				typeId(nil), meta_nil, // f3 (id = 7)
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testStruct2)
				s.f3 = byte(123)
				return *s.f1.(*any) == s.f3 && *s.f2.(*any) == s.f3 && s.f3 == byte(123)
			},
		},
		// #6
		{
			func() any {
				s := &testStruct2{}
				s.f1 = &s.f2
				s.f3 = &s.f2
				return s
			}(),
			[]byte{
				version, typeId((*testStruct2)(nil)), meta_nonil, meta_aggr, // *testStruct2
				typeId((*any)(nil)), meta_nonil, meta_ref, c2b0(5), // f1 is ref to f2 (id = 2)
				typeId(nil), meta_nil, // f2 (id = 5)
				meta_ref, c2b0(4), // f3 (id = 7)

			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testStruct2)
				s.f2 = byte(123)
				return *s.f1.(*any) == s.f2 && *s.f3.(*any) == s.f2 && s.f2 == byte(123)
			},
		},*/
	}
	runTests(items, reg, t)
}

/*func Test_StructWithoutTags(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			testStruct1{
				123,
				true,
				"abc",
				0,
				"abc",
			},
			[]byte{
				version,
				typeId(testStruct1{}), meta_aggr, // testStruct1 header
				0b0010_0000, 246, // testStruct1.f1 (id = 1)
				meta_tru,               // testStruct1.f2 (id = 3)
				c2b0(3), 'a', 'b', 'c', // testStruct1.F3 (id = 5)
				0,                 // testStruct1.F4 (id = 7)
				meta_ref, c2b0(6), // ref to testStruct1.F3 (id = 9)
			},
			nil,
		},
		// #2
		{
			testStruct2{
				testStruct1{
					111,
					true,
					"abcde",
					0,
					"",
				},
				nil,
				testStruct1{
					0,
					false,
					"",
					128,
					"abcde",
				},
			},
			[]byte{
				version,
				typeId(testStruct2{}), meta_aggr, // testStruct2 header
				typeId(testStruct1{}), meta_aggr, // testStruct2.f1 (id = 2)
				0b0010_0000, 222, // testStruct2.f1.f1 (id = 4)
				meta_tru,                         // testStruct2.f1.f2 (id = 6)
				c2b0(5), 'a', 'b', 'c', 'd', 'e', // testStruct2.f1.F3 (id = 8)
				0,                     // testStruct2.f1.F4 (id = 10)
				c2b0(0),               // testStruct2.f1.f5 (id = 12)
				typeId(nil), meta_nil, // testStruct2.f2
				typeId(testStruct1{}), meta_aggr, // testStruct2.f3
				0b0001_0000,        // testStruct2.f3.f1
				meta_fls,           // testStruct2.f3.f2
				meta_ref, c2b0(13), // testStruct2.f3.F3
				128,               // testStruct2.f3.F4
				meta_ref, c2b0(9), // testStruct2.f3.f5
			},
			nil,
		},
	}
	runTests(items, reg, t)
}*/

/*func Test_ComplexValue(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			newLst(),
			[]byte{
				version, typeId((*lst)(nil)), meta_nonil, meta_aggr, // *lst
				meta_aggr,                                           // lst.root header
				meta_nonil, meta_ref, c2b0(2),                       // lst.root.next = &lst.root (id = 4)
				meta_ref, c2b0(5),                                   // lst.root.prev = &lst.root
				meta_nil,                                            // lst.root.lst = nil
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*lst)
				root := s.root
				s.root = testNode{}
				return root.next == root.prev && root.next == &s.root
			},
		},
		// #2
		{
			func() any {
				l := newLst()
				l.push()
				return l
			}(),
			[]byte{
				version, typeId((*lst)(nil)), meta_nonil, meta_aggr, // *lst
				meta_aggr,                                           // lst.root header
				meta_nonil,                                          // lst.root.next = &testNode (id = 4)
				meta_aggr,                                           // testNode header (id = 5)
				meta_nonil, meta_ref, c2b0(2),                       // testNode.next = &lst.root (id = 7)
				meta_ref, c2b0(8),                                   // testNode.prev = &lst.root
				meta_ref, c2b0(0),                                   // testNode.lst = &lst
				meta_ref, c2b0(5),                                   // lst.root.prev = &testNode
				meta_nil,                                            // lst.root.lst = nil
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*lst)
				return s == s.root.next.lst && s.root.next == s.root.prev && s.root.next != &s.root &&
					s.root.prev.prev == &s.root && s.root.next.next == &s.root
			},
		},
		// #3
		{
			func() any {
				l := newLst()
				l.push()
				l.push()
				return l
			}(),
			[]byte{
				version, typeId((*lst)(nil)), meta_nonil, meta_aggr, // *lst
				meta_aggr,                                           // lst.root header
				meta_nonil,                                          // lst.root.next = &testNode1
				meta_aggr,                                           // testNode1 header
				meta_nonil,                                          // testNode1.next = &testNode2
				meta_aggr,                                           // testNode2 header
				meta_nonil, meta_ref, c2b0(2),                       // testNode2.next = &lst.root
				meta_ref, c2b0(5),                                   // testNode2.prev = &testNode1
				meta_ref, c2b0(0),                                   // testNode2.lst = lst
				meta_ref, c2b0(11),                                  // testNode1.prev = &lst.root
				meta_ref, c2b0(0),                                   // testNode1.lst = lst
				meta_ref, c2b0(8),                                   // lst.root.prev = &testNode2
				meta_nil,                                            // lst.eoot.lst = nil
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*lst)
				return s == s.root.next.lst && s == s.root.next.next.lst &&
					s.root.next.next.next == &s.root && s.root.prev.prev.prev == &s.root
			},
		},
		// #4
		{
			func() any {
				l := newLst()
				l.root.lst = l
				return l.root.next
			}(),
			[]byte{
				version,
				typeId((*testNode)(nil)), meta_nonil, meta_ref, c2b0(7), // *lst.root
				typeId(testNode{}), meta_aggr, // testNode header

			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				//s := actual.(*lst)
				return true
			},
		},
	}
	runTests(items, reg, t)
}*/

/*func TestPtr(t *testing.T) {
	elemType := reflect.TypeOf((*any)(nil)).Elem()

	v3 := reflex.Zero(elemType)
	v3.Set(reflex.PtrAt(elemType, reflex.Zero(elemType)))

	v2 := reflex.Zero(elemType)
	v2.Set(reflex.PtrAt(elemType, reflex.Zero(elemType)))
	if (v3.Kind() == reflect.Pointer && elemType.Kind() == reflect.Interface) {
		v2.Elem().Set(v3)
	} else {
		v2.Set(reflex.PtrAt(elemType, v3))
	}

	v1 := reflex.Zero(elemType)
	v1.Set(reflex.PtrAt(elemType, reflex.Zero(elemType)))
	if (v2.Kind() == reflect.Pointer && elemType.Kind() == reflect.Interface) {
		v1.Elem().Set(v2)
	} else {
		v1.Set(reflex.PtrAt(elemType, v2))
	}

	tru := reflex.Zero(reflect.TypeOf(false))
	tru.SetBool(true)
	v3.Set(reflex.PtrAt(reflect.TypeOf(false), tru))

	v := v1.Interface()
	fmt.Println(v)
}*/
