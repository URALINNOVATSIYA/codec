package codec

import (
	"bytes"
	"errors"
	"fmt"

	"sync"

	"math"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/URALINNOVATSIYA/codec/testpkg"
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
		if item.data != nil && !bytes.Equal(data, item.data) {
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

func registryWithFuncId() (*TypeRegistry, func(v any) byte, func(v any) byte) {
	reg := NewTypeRegistry(true)
	return reg, func(v any) byte {
			id := reg.typeIdByValue(reflect.ValueOf(v))
			return u2bs(uint64(id), 3)[0]
		},
		func(v any) byte {
			id := reg.funcIdByValue(reflect.ValueOf(v))
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
		// #1
		{
			"",
			[]byte{version, typeId(""), c2b0(0)},
			nil,
		},
		// #2
		{
			"0123456789",
			[]byte{version, typeId(""), c2b0(10), '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'},
			nil,
		},
		// #3
		{
			strings.Repeat("a", 255),
			append(append([]byte{version, typeId("")}, c2b(255)...), []byte(strings.Repeat("a", 255))...),
			nil,
		},
		// #4
		{
			strings.Repeat("a", 65536),
			append(append([]byte{version, typeId("")}, c2b(65536)...), []byte(strings.Repeat("a", 65536))...),
			nil,
		},
		// #5
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
		{
			math.NaN(),
			[]byte{version, typeId(float64(0)), 129, 0, 0, 0, 0, 0, 248, 127},
			func(_ any, actual any) bool {
				return math.IsNaN(actual.(float64))
			},
		},
		{
			math.Inf(-1),
			[]byte{version, typeId(float64(0)), 48, 240, 255},
			func(_ any, actual any) bool {
				return math.IsInf(actual.(float64), -1)
			},
		},
		{
			math.Inf(1),
			[]byte{version, typeId(float64(0)), 48, 240, 127},
			func(_ any, actual any) bool {
				return math.IsInf(actual.(float64), 1)
			},
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
	reg, typeId, funcId := registryWithFuncId()
	items := []testItem{
		// #1
		{
			(func(byte, bool) int8)(nil),
			[]byte{version, typeId((func(byte, bool) int8)(nil)), meta_nil},
			nil,
		},
		// #2
		{
			registry,
			[]byte{version, typeId(registry), meta_nonil, funcId(registry)},
			nil,
		},
		// #3
		{
			math.Abs,
			[]byte{version, typeId(math.Abs), meta_nonil, funcId(math.Abs)},
			nil,
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
			nil,
			nil,
		},
		// #4
		{
			func() any {
				b := byte(255)
				return &b
			}(),
			nil,
			nil,
		},
		// #5
		{
			func() any {
				s := "123"
				return &s
			}(),
			nil,
			nil,
		},
		// #6
		{
			func() any {
				var x any = true
				return &x
			}(),
			nil,
			nil,
		},
		// #7
		{
			func() any {
				b := true
				return testBoolPtr(&b)
			}(),
			nil,
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_PointersToTheSameValue(t *testing.T) {
	reg, _ := registry()
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
			nil,
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
			nil,
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
			nil,
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_PointerChain(t *testing.T) {
	reg, _ := registry()
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
			nil,
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
			nil,
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_CyclicPointerChain(t *testing.T) {
	reg, _ := registry()
	items := []testItem{
		// #1
		{
			func() any {
				var x any
				x = &x
				return x
			}(),
			nil,
			func(expected, actual any) bool {
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
			nil,
			func(expected, actual any) bool {
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
			nil,
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
			nil,
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
			nil,
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
			nil,
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
			nil,
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
			nil,
			func(expected, actual any) bool {
				x3 := actual.(**any)
				x2 := *x3
				x1 := *x2
				return *x1.(***any) == x3
			},
		},
	}
	runTests(items, reg, t)
}

func Test_Interface(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			error(nil),
			[]byte{
				version, typeId(error(nil)), meta_nil,
			},
			nil,
		},
		// #2
		{
			[1]error{errors.New("test")},
			nil,
			nil,
		},
		// #3
		{
			testSerializableInt(123),
			[]byte{
				version, typeId(testSerializableInt(0)), meta_nonil, c2b0(3), 49, 50, 51,
			},
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Map(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			map[int]byte{},
			[]byte{
				version, typeId(map[int]byte{}), meta_nonil, c2b0(0),
			},
			nil,
		},
		// #2
		{
			map[int]bool{1: true, 2: true, 7: false, -123: false},
			nil,
			nil,
		},
		// #3
		{
			map[any]any{byte(1): true, "a": 1.0, struct{}{}: map[string]any{"z": -123}},
			nil,
			nil,
		},
		// #4
		{
			func() any {
				b := true
				return map[int]*bool{1: &b}
			}(),
			nil,
			func(expected, actual any) bool {
				return true
			},
		},
		// #5
		{
			func() any {
				i := 123
				return map[any]*int{1: &i}
			}(),
			nil,
			nil,
		},
		// #6
		{
			func() any {
				i := 123
				b := true
				return map[any]*int{&b: &i}
			}(),
			nil,
			func(expected, actual any) bool {
				m := actual.(map[any]*int)
				for k, v := range m {
					return *k.(*bool) == true && *v == 123
				}
				return false
			},
		},
		// #7
		{
			func() any {
				var x any
				x = &x
				return map[any]any{x: x}
			}(),
			nil,
			func(expected, actual any) bool {
				m := actual.(map[any]any)
				for k, v := range m {
					return k.(*any) == *k.(*any) && k == v
				}
				return true
			},
		},
		// #8
		{
			testMap{"abc": 1234},
			nil,
			nil,
		},
		// #9
		{
			testRecMap{},
			nil,
			nil,
		},
		// #10
		{
			testRecMap{8: testRecMap(nil)},
			nil,
			nil,
		},
		// #11
		{
			[1]any{map[int]bool{1: true}},
			nil,
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Array(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			[0]int{},
			[]byte{
				version, typeId([0]int{}),
			},
			nil,
		},
		// #2
		{
			[3]byte{1, 2, 3},
			[]byte{
				version, typeId([3]byte{}), 1, 2, 3,
			},
			nil,
		},
		// #3
		{
			[2][1]bool{{true}, {true}},
			[]byte{
				version, typeId([2][1]bool{}), meta_tru, meta_tru,
			},
			nil,
		},
		// #4
		{
			testArray{-1, 1, 0},
			[]byte{
				version, typeId(testArray{}), 17, 18, 16,
			},
			nil,
		},
		// #5
		{
			func() any {
				b := true
				return [3]*bool{&b, &b, &b}
			}(),
			nil,
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Struct(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			struct{}{},
			[]byte{version, typeId(struct{}{}), meta_strc},
			nil,
		},
		// #2
		{
			testpkg.Get(),
			nil,
			func(_, actual any) bool {
				return testpkg.Check(actual)
			},
		},
		// #3
		{
			func() any {
				m := &sync.Mutex{}
				m.Lock()
				return m
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				m := actual.(*sync.Mutex)
				return !m.TryLock()
			},
		},
		// #4
		{
			func() any {
				w := &sync.WaitGroup{}
				w.Add(5)
				return w
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				w := actual.(*sync.WaitGroup)
				w.Add(-5)
				w.Wait()
				return true
			},
		},
		// #5
		{
			func() any {
				m := map[any]bool{1: true}
				s := &testS3{}
				s.F1 = m
				s.F2 = [2]int{1, 2}
				s.F3 = &m
				return s
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3)
				s.F1 = false
				return (*s.F3.(*map[any]bool))[1]
			},
		},
		// #6
		{
			func() any {
				s := &testNestedS1{
					testS1{
						F1: 100,
					},
					testS1Ptr{},
					testS2{
						F1: 123,
						F2: true,
					},
				}
				return s
			}(),
			nil,
			nil,
		},
		// #7
		{
			testS1Tags{
				f1: 123,
				f2: true,
				F3: "abc",
				f5: "abc",
			},
			nil,
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_Slice(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		// #1
		{
			([]int)(nil),
			[]byte{
				version, typeId([]int{}), meta_slice | meta_nil,
			},
			nil,
		},
		// #2
		{
			[]bool{true, false, true},
			[]byte{
				version, typeId([]bool{}), meta_slice, c2b0(3), c2b0(3),
				meta_tru, meta_fls, meta_tru,
			},
			nil,
		},
		// #3
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				return [2][]byte{s, s[0:3:5]}
			}(),
			[]byte{
				version, typeId([2][]byte{}),
				meta_slice, c2b0(6), c2b0(6), 1, 2, 3, 4, 5, 6, // s
				meta_slice | meta_sexp, i2b0(2), c2b0(0), c2b0(3), c2b0(5), // s[0:3:5]
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.([2][]byte)
				p := a[0]
				s := a[1]
				p[0] = 100
				p[1] = 101
				p[2] = 102
				return s[0] == 100 && s[1] == 101 && s[2] == 102 && cap(s) == 5
			},
		},
		// #4
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				return [3][]byte{s, s[2:4], s[0:3:4]}
			}(),
			[]byte{
				version, typeId([3][]byte{}),
				meta_slice, c2b0(6), c2b0(6), 1, 2, 3, 4, 5, 6, // s
				meta_slice | meta_sexp, i2b0(2), c2b0(2), c2b0(4), c2b0(4), // s[2:4]
				meta_slice | meta_sexp, i2b0(2), c2b0(0), c2b0(3), c2b0(4), // s[0:3:4]
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.([3][]byte)
				p := a[0]
				s1 := a[1]
				s2 := a[2]
				p[0] = 100
				p[1] = 101
				p[2] = 102
				p[3] = 103
				return s2[0] == 100 && s2[1] == 101 && s2[2] == 102 && cap(s2) == 4 &&
					s1[0] == 102 && s1[1] == 103 && cap(s1) == 2
			},
		},
		// #5
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				return [3]any{s, s[2:4], s[0:3:4]}
			}(),
			[]byte{
				version, typeId([3]any{}),
				typeId([]byte{}), meta_slice, c2b0(6), c2b0(6), 1, 2, 3, 4, 5, 6, // s
				typeId([]byte{}), meta_slice | meta_sexp, i2b0(3), c2b0(2), c2b0(4), c2b0(4), // s[2:4]
				typeId([]byte{}), meta_slice | meta_sexp, i2b0(3), c2b0(0), c2b0(3), c2b0(4), // s[0:3:4]
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.([3]any)
				p := a[0].([]byte)
				s1 := a[1].([]byte)
				s2 := a[2].([]byte)
				p[0] = 100
				p[1] = 101
				p[2] = 102
				p[3] = 103
				return s2[0] == 100 && s2[1] == 101 && s2[2] == 102 && cap(s2) == 4 &&
					s1[0] == 102 && s1[1] == 103 && cap(s1) == 2
			},
		},
		// #6
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				return [3]any{s[2:4], s, s[0:3:4]}
			}(),
			[]byte{
				version, typeId([3]any{}),
				typeId([]byte{}), meta_slice | meta_sexp, i2b(6)[0], c2b0(2), c2b0(4), c2b0(4), // s[2:4]
				typeId([]byte{}), meta_slice, c2b0(6), c2b0(6), 1, 2, 3, 4, 5, 6, // s
				typeId([]byte{}), meta_slice | meta_sexp, i2b(6)[0], c2b0(0), c2b0(3), c2b0(4), // s[0:3:4]
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.([3]any)
				p := a[1].([]byte)
				s1 := a[0].([]byte)
				s2 := a[2].([]byte)
				p[0] = 100
				p[1] = 101
				p[2] = 102
				p[3] = 103
				return s2[0] == 100 && s2[1] == 101 && s2[2] == 102 && cap(s2) == 4 &&
					s1[0] == 102 && s1[1] == 103 && cap(s1) == 2
			},
		},
		// #7
		{
			func() any {
				a := &[3]any{}
				a[0] = a[0:2]
				a[1] = a[1:2]
				a[2] = a[1:3]
				return a
			}(),
			[]byte{
				version, typeId((*[3]any)(nil)), meta_nonil, // *a
				typeId([3]any{}),                                                            // a
				typeId([]any{}), meta_slice | meta_sexp, i2b0(1), c2b0(0), c2b0(2), c2b0(3), // a[0:2]
				typeId([]any{}), meta_slice | meta_sexp, i2b0(1), c2b0(1), c2b0(2), c2b0(2), // a[1:2]
				typeId([]any{}), meta_slice | meta_sexp, i2b0(1), c2b0(1), c2b0(3), c2b0(3), // a[1:3]
				// refs
				meta_ref, c2b0(1), c2b0(0),
			},
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.(*[3]any)
				a[1].([]any)[0] = 123
				return a[0].([]any)[1] == 123 && a[2].([]any)[0] == 123
			},
		},
		// #8
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				return [3][]byte{s[4:6], s[1:3], s[0:2]}
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.([3][]byte)
				a[1][0] = 123
				if a[2][1] != 123 {
					return false
				}
				a[1] = append(a[1], 111, 222)
				return a[0][0] == 222
			},
		},
	}
	runTests(items, reg, t)
}

func Test_ReferenceToTheSameValue(t *testing.T) {
	reg, typeId, funcId := registryWithFuncId()
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
				typeId(registry), meta_nonil, funcId(registry), // testS3.F1
				typeId(registry), meta_ref, c2b0(3), // testS3.F2
				typeId(nil), meta_nil, // testS3.F3
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
		// #4
		{
			[4]func(int, int) int{testSum, testDiv, testSum, testDiv},
			[]byte{
				version,
				typeId([4]func(int, int) int{}),
				meta_nonil, funcId(testSum),
				meta_nonil, funcId(testDiv),
				meta_ref, c2b0(2),
				meta_ref, c2b0(4),
			},
			func(expected, actual any) bool {
				e := expected.([4]func(int, int) int)
				a := actual.([4]func(int, int) int)
				return funcEqual(e[0], a[0]) && funcEqual(e[1], a[1]) && funcEqual(e[2], a[2]) &&
					funcEqual(e[3], a[3]) && funcEqual(a[0], a[2]) && funcEqual(a[1], a[3])
			},
		},
		// #5
		{
			func() any {
				m := map[byte]int{1: 123, 2: 213, 3: 321}
				return [3]any{m, m, m}
			}(),
			nil,
			nil,
		},
		// #6
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				return [4]any{s[2:4], s[2:4], s, s}
			}(),
			nil,
			nil,
		},
		// #7
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				return [4]any{&s, &s, s, s}
			}(),
			nil,
			nil,
		},
		// #8
		{
			func() any {
				s := []byte{1, 2, 3, 4, 5, 6}
				s1 := s[2:4]
				s2 := s[2:4]
				return [4]any{&s1, &s2, s, s}
			}(),
			nil,
			nil,
		},
		// #9
		{
			func() any {
				s := "abc"
				return [4]any{&s, &s, s, s}
			}(),
			nil,
			nil,
		},
	}
	runTests(items, reg, t)
}

func Test_BackwardPointerToContainer(t *testing.T) {
	reg, _ := registry()
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
			nil,
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
			nil,
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
			nil,
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
			nil,
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
			nil,
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
}

func Test_ForwardPointerToContainer(t *testing.T) {
	reg, _ := registry()
	items := []testItem{
		// #1
		{
			func() any {
				s := &testS1{}
				s.F1 = s
				x := &s.F1
				return x
			}(),
			nil,
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
				a := &[1]any{}
				a[0] = a
				x := &a[0]
				return x
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				x := actual.(*any)
				a := (*x).(*[1]any)
				return a[0] == a && x == &a[0]
			},
		},
		// #3
		{
			func() any {
				s := &testS1Ptr{}
				s.F1 = s
				x := &s.F1
				return x
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				x := actual.(**testS1Ptr)
				s := *x
				return s.F1 == s && x == &s.F1
			},
		},
		// #4
		{
			func() any {
				s := &testS4{}
				s.F1 = &s.F3
				s.F2 = &s.F3
				s.F3 = true
				s.F4 = &s.F3
				return s
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := (actual).(*testS4)
				return s.F1 == &s.F3 && s.F2 == &s.F3 && s.F4 == &s.F3
			},
		},
		// #5
		{
			func() any {
				s := &testS4{}
				s.F1 = &s.F3
				s.F2 = &s.F3
				s.F3 = &s.F1
				s.F4 = &s.F3
				return s
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := (actual).(*testS4)
				return s.F1 == &s.F3 && s.F2 == &s.F3 && s.F4 == &s.F3 && s.F3 == &s.F1
			},
		},
		// #6
		{
			func() any {
				a := &[4]any{}
				a[0] = &a[2]
				a[1] = &a[2]
				a[2] = &a[0]
				a[3] = &a[2]
				return a
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := (actual).(*[4]any)
				return a[0] == &a[2] && a[1] == &a[2] && a[3] == &a[2] && a[2] == &a[0]
			},
		},
		// #6
		{
			func() any {
				s := &testS2{}
				s.F1 = &s.F2
				s.F2 = &s.F1
				return s
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS2)
				return *s.F1.(*any) == s.F2 && *s.F2.(*any) == s.F1
			},
		},
		// #7
		{
			func() any {
				a := &[2]any{}
				a[0] = &a[1]
				a[1] = &a[0]
				return a
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.(*[2]any)
				return *a[0].(*any) == a[1] && *a[1].(*any) == a[0]
			},
		},
		// #8
		{
			func() any {
				s := &testS2{}
				x := &s.F2
				y := &s.F1
				s.F1 = &x
				s.F2 = &y
				return s
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS2)
				return **s.F1.(**any) == s.F2 && **s.F2.(**any) == s.F1
			},
		},
		// #9
		{
			func() any {
				a := &[2]any{}
				x := &a[1]
				y := &a[0]
				a[0] = &x
				a[1] = &y
				return a
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				a := actual.(*[2]any)
				return **a[0].(**any) == a[1] && **a[1].(**any) == a[0]
			},
		},
		// #10
		{
			func() any {
				s := &testS3{}
				x := &s.F3
				s.F1 = &x
				s.F2 = &x
				return s
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3)
				s.F3 = 123
				return **s.F1.(**any) == s.F3 && **s.F2.(**any) == s.F3 && s.F3 == 123
			},
		},
		// #11
		{
			func() any {
				s := &testS3{}
				x := &s.F2
				s.F1 = &x
				s.F3 = &x
				return s
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*testS3)
				s.F2 = 123
				return **s.F1.(**any) == s.F2 && **s.F3.(**any) == s.F2 && s.F2 == 123
			},
		},
	}
	runTests(items, reg, t)
}

func Test_MixedContainerPointers(t *testing.T) {
	reg, _ := registry()
	items := []testItem{
		// #1
		{
			func() any {
				s1 := &testS1{}
				s1.F1 = s1
				s2 := &testS2{}
				s2.F2 = &s1.F1
				x := &s2.F2
				s2.F1 = &x
				return s2
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s2 := actual.(*testS2)
				x := s2.F2.(*any)
				s1 := (*x).(*testS1)
				return s1.F1 == s1 && x == &s1.F1 && **s2.F1.(**any) == s2.F2 && s2.F2 == x
			},
		},
		// #2
		{
			func() any {
				s1 := &testS1{}
				s1.F1 = s1
				s2 := &testS3{}
				s2.F2 = &s1.F1
				x := &s2.F2
				s2.F1 = &x
				s2.F3 = s2
				return s2.F3
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s2 := actual.(*testS3)
				x := s2.F2.(*any)
				s1 := (*x).(*testS1)
				return s1.F1 == s1 && x == &s1.F1 && **s2.F1.(**any) == s2.F2 && s2.F2 == x && s2.F3 == s2
			},
		},
		// #3
		{
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
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s2 := actual.(*testS3)
				s1 := (*s2.F3.(*any)).(*testS3)
				return s2.F1 == &s2.F2 && s2.F2 == &s1.F2 && s2.F3 == &s1.F1 && s1.F3 == &s2.F3 && s1.F2 == &s1.F3 && s1.F1 == s1
			},
		},
		// #4
		{
			newLst(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*lst)
				return s.root.next == &s.root && s.root.prev == &s.root
			},
		},
		// #5
		{
			func() any {
				l := newLst()
				l.root.lst = l
				return l.root.next
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				r := actual.(*testNode)
				s := r.lst
				return s.root.next == &s.root && s.root.prev == &s.root && s.root.lst == s
			},
		},
		// #6
		{
			func() any {
				l := newLst()
				l.push()
				return l
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*lst)
				return s == s.root.next.lst && s.root.next == s.root.prev && s.root.next != &s.root &&
					s.root.prev.prev == &s.root && s.root.next.next == &s.root
			},
		},
		// #7
		{
			func() any {
				l := newLst()
				l.push()
				l.push()
				return l
			}(),
			nil,
			func(expected, actual any) bool {
				if !defaultEq(expected, actual) {
					return false
				}
				s := actual.(*lst)
				return s == s.root.next.lst && s == s.root.next.next.lst &&
					s.root.next.next.next == &s.root && s.root.prev.prev.prev == &s.root
			},
		},
	}
	runTests(items, reg, t)
}
