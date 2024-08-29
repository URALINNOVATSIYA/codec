package codec

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
	"unsafe"
)

// types for test
type (
	testBool                            bool
	testString                          string
	testInt8                            int8
	testUint8                           uint8
	testInt16                           int16
	testUint16                          uint16
	testInt32                           int32
	testUint32                          uint32
	testInt64                           int64
	testUint64                          uint64
	testInt                             int
	testUint                            uint
	testFloat32                         float32
	testFloat64                         float64
	testComplex64                       complex64
	testComplex128                      complex128
	testArray                           [3]int
	testGenericArray[T comparable]      [2]T
	testSlice                           []string
	testGenericSlice[T comparable]      []T
	testRecSlice                        []testRecSlice
	testMap                             map[string]uint64
	testGenericMap[K comparable, V any] map[K]V
	testRecMap                          map[byte]testRecMap
	testPtr                             *bool
	testUnsafePointer                   unsafe.Pointer
	testUintptr                         uintptr
	testRecPtr                          *testRecPtr
	testChan                            <-chan *bool
	testStruct                          struct {
		F1 string
		F2 bool
		F3 *testStruct
		F4 any
		f5 int
		f6 string
		f7 *testStruct
	}
	testInterface struct {
		src io.Reader
		ref *io.Reader
		p   *testStruct
	}
)

type testCustomStruct struct {
	f1 bool
	f2 string
	f3 uint
}

func (s testCustomStruct) Serialize() []byte {
	b := []any{s.f3, s.f2, s.f1}
	return Serialize(b)
}

func (s testCustomStruct) Unserialize(data []byte) (any, error) {
	b, err := Unserialize(data)
	if err != nil {
		return nil, err
	}
	s.f1 = b.([]any)[2].(bool)
	s.f2 = b.([]any)[1].(string)
	s.f3 = b.([]any)[0].(uint)
	return s, nil
}

type testCustomPointerStruct struct {
	f1 bool
	f2 string
	f3 uint
}

func (s *testCustomPointerStruct) Serialize() []byte {
	b := []any{s.f3, s.f2, s.f1}
	return Serialize(b)
}

func (s *testCustomPointerStruct) Unserialize(data []byte) (any, error) {
	b, err := Unserialize(data)
	if err != nil {
		return nil, err
	}
	s.f1 = b.([]any)[2].(bool)
	s.f2 = b.([]any)[1].(string)
	s.f3 = b.([]any)[0].(uint)
	return s, nil
}

type testCustomNestedStruct struct {
	data string
	ptr  *testCustomNestedStruct
}

func (s *testCustomNestedStruct) Serialize() []byte {
	b := []any{s.data, s.ptr}
	return Serialize(b)
}

func (s *testCustomNestedStruct) Unserialize(data []byte) (any, error) {
	b, err := Unserialize(data)
	if err != nil {
		return nil, err
	}
	fields := b.([]any)
	s.data = fields[0].(string)
	if fields[1] != nil {
		s.ptr = fields[1].(*testCustomNestedStruct)
	}
	return s, nil
}

type testCustomUint uint

func (s testCustomUint) Serialize() []byte {
	return toBytes(uint64(s), 8)
}

func (s testCustomUint) Unserialize(data []byte) (any, error) {
	var i uint64
	for j, size := 0, len(data); j < size; j++ {
		i = i<<8 | uint64(data[j])
	}
	return testCustomUint(i), nil
}

var testTypeRegistered bool

func registerTestTypes() {
	if testTypeRegistered {
		return
	}
	testTypeRegistered = true
	RegisterTypeOf(testBool(false))
	RegisterTypeOf(testString(""))
	RegisterTypeOf(testInt8(0))
	RegisterTypeOf(testUint8(0))
	RegisterTypeOf(testInt16(0))
	RegisterTypeOf(testUint16(0))
	RegisterTypeOf(testInt32(0))
	RegisterTypeOf(testUint32(0))
	RegisterTypeOf(testInt64(0))
	RegisterTypeOf(testUint64(0))
	RegisterTypeOf(testInt(0))
	RegisterTypeOf(testUint(0))
	RegisterTypeOf(testFloat32(0))
	RegisterTypeOf(testFloat64(0))
	RegisterTypeOf(testComplex64(0))
	RegisterTypeOf(testComplex128(0))
	RegisterTypeOf(testArray{})
	RegisterTypeOf(testGenericArray[uint16]{})
	RegisterTypeOf(testSlice(nil))
	RegisterTypeOf(testRecSlice(nil))
	RegisterTypeOf(testGenericSlice[byte](nil))
	RegisterTypeOf(testMap(nil))
	RegisterTypeOf(testGenericMap[string, int32](nil))
	RegisterTypeOf(testRecMap(nil))
	RegisterTypeOf(testPtr(nil))
	RegisterTypeOf(testUnsafePointer(nil))
	RegisterTypeOf(testUintptr(0))
	RegisterTypeOf(testRecPtr(nil))
	RegisterTypeOf(testChan(nil))
	RegisterTypeOf(testStruct{})
	RegisterTypeOf(testCustomStruct{})
	RegisterTypeOf(testCustomUint(0))
	RegisterType(reflect.TypeOf(errors.New("")).Elem())
	RegisterType(reflect.TypeOf((*error)(nil)).Elem())
}

func TestBasicTypeSignature(t *testing.T) {
	var items = [][]any{
		{
			nil,
			[]byte{tNil},
		},
		{
			true,
			[]byte{tBool},
		},
		{
			uint8(123),
			[]byte{tInt8},
		},
		{
			int8(-123),
			[]byte{tInt8 | signed},
		},
		{
			uint16(12345),
			[]byte{tInt16},
		},
		{
			int16(-12345),
			[]byte{tInt16 | signed},
		},
		{
			uint32(1234567),
			[]byte{tInt32},
		},
		{
			int32(-1234567),
			[]byte{tInt32 | signed},
		},
		{
			uint64(1234567890),
			[]byte{tInt64},
		},
		{
			int64(-1234567890),
			[]byte{tInt64 | signed},
		},
		{
			uint(1234567890),
			[]byte{tInt},
		},
		{
			-1234567890,
			[]byte{tInt | signed},
		},
		{
			float32(1.23),
			[]byte{tFloat},
		},
		{
			1.23,
			[]byte{tFloat | wide},
		},
		{
			complex64(0),
			[]byte{tComplex},
		},
		{
			complex128(0),
			[]byte{tComplex | wide},
		},
		{
			"",
			[]byte{tString},
		},
		{
			(*any)(nil),
			[]byte{tPointer},
		},
		{
			uintptr(0),
			[]byte{tUintptr},
		},
		{
			unsafe.Pointer(nil),
			[]byte{tUintptr | raw},
		},
		{
			[]any{},
			[]byte{tType, id([]any{}), tList},
		},
		{
			[0]any{},
			[]byte{tType, id([0]any{}), tList | fixed},
		},
		{
			map[any]any{},
			[]byte{tType, id(map[any]any{}), tMap},
		},
		{
			struct{}{},
			[]byte{tType, id(struct{}{}), tStruct},
		},
	}
	checkTypeSignature(items, t)
}

func TestTypeAliasesSignature(t *testing.T) {
	var items = [][]any{
		{
			testBool(true),
			[]byte{tType, id(testBool(false)), tBool},
		},
		{
			testString(""),
			[]byte{tType, id(testString("")), tString},
		},
		{
			testInt8(0),
			[]byte{tType, id(testInt8(0)), tInt8 | signed},
		},
		{
			testUint8(0),
			[]byte{tType, id(testUint8(0)), tInt8},
		},
		{
			testInt16(0),
			[]byte{tType, id(testInt16(0)), tInt16 | signed},
		},
		{
			testUint16(0),
			[]byte{tType, id(testUint16(0)), tInt16},
		},
		{
			testInt32(0),
			[]byte{tType, id(testInt32(0)), tInt32 | signed},
		},
		{
			testUint32(0),
			[]byte{tType, id(testUint32(0)), tInt32},
		},
		{
			testInt64(0),
			[]byte{tType, id(testInt64(0)), tInt64 | signed},
		},
		{
			testUint64(0),
			[]byte{tType, id(testUint64(0)), tInt64},
		},
		{
			testInt(0),
			[]byte{tType, id(testInt(0)), tInt | signed},
		},
		{
			testUint(0),
			[]byte{tType, id(testUint(0)), tInt},
		},
		{
			testFloat32(0),
			[]byte{tType, id(testFloat32(0)), tFloat},
		},
		{
			testFloat64(0),
			[]byte{tType, id(testFloat64(0)), tFloat | wide},
		},
		{
			testComplex64(0),
			[]byte{tType, id(testComplex64(0)), tComplex},
		},
		{
			testComplex128(0),
			[]byte{tType, id(testComplex128(0)), tComplex | wide},
		},
		{
			testArray{},
			[]byte{tType, id(testArray{}), tList | fixed},
		},
		{
			testGenericArray[uint16]{},
			[]byte{tType, id(testGenericArray[uint16]{}), tList | fixed},
		},
		{
			testSlice{},
			[]byte{tType, id(testSlice{}), tList},
		},
		{
			testGenericSlice[string]{},
			[]byte{tType, id(testGenericSlice[string]{}), tList},
		},
		{
			testRecSlice{},
			[]byte{tType, id(testRecSlice{}), tList},
		},
		{
			testMap{},
			[]byte{tType, id(testMap{}), tMap},
		},
		{
			testGenericMap[bool, int]{},
			[]byte{tType, id(testGenericMap[bool, int]{}), tMap},
		},
		{
			testRecMap{},
			[]byte{tType, id(testRecMap{}), tMap},
		},
		{
			testPtr(nil),
			[]byte{tType, id(testPtr(nil)), tPointer},
		},
		{
			testRecPtr(nil),
			[]byte{tType, id(testRecPtr(nil)), tPointer},
		},
		{
			testUnsafePointer(nil),
			[]byte{tType, id(testUnsafePointer(nil)), tUintptr | raw},
		},
		{
			testUintptr(0),
			[]byte{tType, id(testUintptr(0)), tUintptr},
		},
		// mixed
		{
			map[testInt]int{},
			[]byte{tType, id(map[testInt]int{}), tMap},
		},
		{
			map[int]testInt{},
			[]byte{tType, id(map[int]testInt{}), tMap},
		},
		{
			map[testInt]map[bool]string{},
			[]byte{tType, id(map[testInt]map[bool]string{}), tMap},
		},
		{
			[]testSlice{},
			[]byte{tType, id([]testSlice{}), tList},
		},
		{
			[2][]testSlice{},
			[]byte{tType, id([2][]testSlice{}), tList | fixed},
		},
		{
			map[string]testSlice{},
			[]byte{tType, id(map[string]testSlice{}), tMap},
		},
		{
			map[string][]testSlice{},
			[]byte{tType, id(map[string][]testSlice{}), tMap},
		},
		{
			[]testMap{},
			[]byte{tType, id([]testMap{}), tList},
		},
		{
			map[uint][3]testMap{},
			[]byte{tType, id(map[uint][3]testMap{}), tMap},
		},
	}
	checkTypeSignature(items, t)
}

func TestStructAndInterfaceTypes(t *testing.T) {
	var items = [][]any{
		{
			testStruct{},
			[]byte{tType, id(testStruct{}), tStruct},
		},
	}
	checkTypeSignature(items, t)
}

func checkTypeSignature(items [][]any, t *testing.T) {
	registerTestTypes()
	for _, item := range items {
		actual := tChecker.typeSignatureOf(reflect.ValueOf(item[0]))
		expected := item[1].([]byte)
		if !bytes.Equal(expected, actual) {
			t.Errorf("typeSignatureOf(%#v) expected %v, but actual value is %v", item[0], expected, actual)
		}
	}
}
