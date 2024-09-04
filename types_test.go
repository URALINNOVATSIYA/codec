package codec

import (
	"reflect"
	"testing"
	"unsafe"
)

// Test types

type (
	testBool     bool
	testStr      string
	testUint8    uint8
	testInt8     int8
	testUint16   uint16
	testInt16    int16
	testFloat32  float32
	testFloat64  float64
	testSlice    []string
	testRecSlice []testRecSlice
	testBoolPtr  *bool
	testRecPtr   *testRecPtr
)

// End test types

func id(v any) byte {
	id := GetDefaultTypeRegistry().typeIdByValue(reflect.ValueOf(v))
	return asBytesWithSize(uint64(id), 3)[0]
}

func TestTypeName(t *testing.T) {
	items := []struct {
		t    reflect.Type
		name string
	}{
		{reflect.TypeOf(nil), "nil"},
		{reflect.TypeOf(false), "bool"},
		{reflect.TypeOf(""), "string"},
		{reflect.TypeOf(uint8(0)), "uint8"},
		{reflect.TypeOf(int8(0)), "int8"},
		{reflect.TypeOf(uint16(0)), "uint16"},
		{reflect.TypeOf(int16(0)), "int16"},
		{reflect.TypeOf(uint32(0)), "uint32"},
		{reflect.TypeOf(int32(0)), "int32"},
		{reflect.TypeOf(uint64(0)), "uint64"},
		{reflect.TypeOf(int64(0)), "int64"},
		{reflect.TypeOf(uint(0)), "uint"},
		{reflect.TypeOf(int(0)), "int"},
		{reflect.TypeOf(float32(0)), "float32"},
		{reflect.TypeOf(float64(0)), "float64"},
		{reflect.TypeOf(complex64(0)), "complex64"},
		{reflect.TypeOf(complex128(0)), "complex128"},
		{reflect.TypeOf(uintptr(0)), "uintptr"},
		{reflect.TypeOf(unsafe.Pointer(nil)), "unsafe.Pointer"},
		{reflect.TypeOf([]any{}).Elem(), "interface {}"},
		{reflect.TypeOf((*any)(nil)), "*interface {}"},
		{reflect.TypeOf((*bool)(nil)), "*bool"},
		{reflect.TypeOf((***complex64)(nil)), "***complex64"},
		{reflect.TypeOf(testRecPtr(nil)), "github.com/URALINNOVATSIYA/codec.testRecPtr"},
	}
	reg := NewTypeRegistry(false)
	for i, item := range items {
		actual := reg.typeName(item.t)
		if item.name != actual {
			t.Errorf("name of type #%d must be %q, but received %q", i+1, item.name, actual)
		}
	}
}

func TestTypeIdByValue(t *testing.T) {
	items := []struct {
		value       any
		encodedType int
	}{
		{nil, 1},
		{false, 2},
		{true, 2},
		{"", 3},
		{0, 4},
		{(*any)(nil), 5},
		{(*bool)(nil), 6},
		{(*[]any)(nil), 7},
		{[]struct{}{}, 8},
		{(*int)(nil), 9},
		{testBoolPtr(nil), 10},
		{(*bool)(nil), 6},
		{testRecPtr(nil), 11},
	}
	reg := NewTypeRegistry(true)
	for i, item := range items {
		expected := item.encodedType
		actual := reg.typeIdByValue(reflect.ValueOf(item.value))
		if actual != expected {
			t.Errorf("type of value #%d (%#v) must be encoded as %#v, but received %#v", i+1, item.value, expected, actual)
		}
	}
}
