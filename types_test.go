package codec

import (
	"reflect"
	"testing"
	"unsafe"
)

// Test types

type (
	testBool          bool
	testStr           string
	testUint8         uint8
	testInt8          int8
	testUint16        uint16
	testInt16         int16
	testUint32        uint32
	testInt32         int32
	testUint64        uint64
	testInt64         int64
	testUint          uint
	testInt           int
	testUintptr       uintptr
	testUnsafePointer unsafe.Pointer
	testFloat32       float32
	testFloat64       float64
	testComplex64     complex64
	testComplex128    complex128
	testChan          <-chan *bool
	testArray         [3]int
	testSlice         []string
	testRecSlice      []testRecSlice
	testMap           map[string]uint64
	testRecMap        map[byte]testRecMap
	testBoolPtr       *bool
	testRecPtr        *testRecPtr
	testStruct1       struct {
		f1 int
		f2 bool
		F3 string
		F4 string
		f5 string
	}
	testStruct2 struct {
		f1 any
		f2 any
		f3 any
	}
	testStruct3 struct {
		f1 *bool
		f2 any
		f3 any
	}
	testStruct4 struct {
		f1 *any
		f2 any
		f3 any
	}
	testStruct5 struct {
		F1 string
		F2 bool
		F3 *testStruct2
		F4 any
		f5 int
		f6 string
		f7 *testStruct2
	}
)

type testInterface interface{}

// End test types

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
