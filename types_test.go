package codec

import (
	"reflect"
	"testing"
	"unsafe"
)

// Test types

type (
	testStr     string
	testBoolPtr *bool
	testRecPtr  *testRecPtr
)

// End test types

func id(v any) byte {
	return GetDefaultTypeRegistry().encodedType(reflect.ValueOf(v))[0]
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

func TestEncodedType(t *testing.T) {
	items := []struct {
		value       any
		encodedType byte
	}{
		{nil, 0},
		{false, 1},
		{true, 1},
		{"", 2},
		{0, 3},
		{(*any)(nil), 4},
		{(*bool)(nil), 5},
		{(*[]any)(nil), 6},
		{[]struct{}{}, 7},
		{(*int)(nil), 8},
		{testBoolPtr(nil), 9},
		{(*bool)(nil), 5},
		{testRecPtr(nil), 10},
	}
	reg := NewTypeRegistry(true)
	for i, item := range items {
		expected := []byte{item.encodedType}
		actual := reg.encodedType(reflect.ValueOf(item.value))
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("type of value #%d (%#v) must be encoded as %#v, but received %#v", i+1, item.value, expected, actual)
		}
	}
}
