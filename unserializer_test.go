package codec

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestUnserialization_Nil(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		nil,
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Bool(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		false,
		true,
		testBool(false),
		testBool(true),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_String(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		"",
		"a",
		"ab",
		strings.Repeat("a", 256),
		strings.Repeat("a", 65536),
		testStr(""),
		testStr("abcdef"),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Uint8(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		uint8(0),
		uint8(1),
		uint8(255),
		testUint8(0),
		testUint8(255),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Int8(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		int8(0),
		int8(1),
		int8(-1),
		int8(-128),
		int8(127),
		testInt8(0),
		testInt8(-128),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Uint16(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		uint16(0),
		uint16(1),
		uint16(256),
		uint16(65535),
		testUint16(0),
		testUint16(65535),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Int16(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		int16(0),
		int16(1),
		int16(-1),
		int16(-256),
		int16(256),
		int16(-32768),
		int16(32767),
		testInt16(0),
		testInt16(-32768),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Uint32(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		uint32(0),
		uint32(1),
		uint32(256),
		uint32(256 << 8),
		uint32(256 << 16),
		uint32(math.MaxUint32),
		testUint32(0),
		testUint32(12345678),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Int32(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		int32(0),
		int32(1),
		int32(-1),
		int32(256),
		int32(-256),
		int32(256 << 8),
		int32(-(256 << 8)),
		int32(256 << 16),
		int32(-(256 << 16)),
		int32(math.MaxInt32),
		int32(math.MinInt32),
		testInt32(0),
		testInt32(-12345678),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Uint64(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		uint64(0),
		uint64(1),
		uint64(256),
		uint64(256 << 8),
		uint64(256 << 16),
		uint64(256 << 24),
		uint64(256 << 32),
		uint64(256 << 40),
		uint64(math.MaxUint64),
		testUint64(0),
		testUint64(1234567890),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Int64(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		int64(0),
		int64(1),
		int64(-1),
		int64(256),
		int64(-256),
		int64(256 << 8),
		int64(-(256 << 8)),
		int64(256 << 16),
		int64(-(256 << 16)),
		int64(256 << 24),
		int64(-(256 << 24)),
		int64(256 << 32),
		int64(-(256 << 32)),
		int64(256 << 40),
		int64(-(256 << 40)),
		int64(math.MaxInt64),
		int64(math.MinInt64),
		testInt64(0),
		testInt64(-1234567890),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Uint(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		uint(0),
		uint(1),
		uint(256),
		uint(256 << 8),
		uint(256 << 16),
		uint(256 << 24),
		uint(256 << 32),
		uint(math.MaxUint),
		testUint(0),
		testUint(1234567),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Int(t *testing.T) {
	reg, _ := registry()
	var values = []any{
		0,
		1,
		-1,
		127,
		-128,
		-129,
		-123456,
		123456,
		12345678,
		-123456789,
		math.MaxInt,
		math.MinInt,
		testInt(0),
		testInt(-123456),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Float32(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		float32(0),
		float32(1),
		float32(-1),
		float32(123),
		float32(-123),
		float32(1.23),
		float32(-1.23),
		testFloat32(0),
		testFloat32(1.23),
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Float64(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		float64(0),
		float64(1),
		float64(-1),
		float64(123),
		float64(-123),
		1.23,
		-1.23,
		testFloat64(0),
		testFloat64(-1.23),
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Pointer(t *testing.T) {
	reg, _ := registry()
	values := []any{
		(*any)(nil),
		func() any {
			b := true
			return &b
		}(),
		func() any {
			b := false
			return &b
		}(),
		func() any {
			s := ""
			return &s
		}(),
		func() any {
			s := "abc"
			return &s
		}(),
		func() any {
			var n0, n1, n2 any
			n2 = -123
			n1 = &n2
			n0 = &n1
			return n0
		}(),
	}
	// *[]uint
	/*v5 := []uint{1, 2, 3}
	values[5] = &v5
	// testPtr
	b := true
	v6 := testPtr(&b)
	values[6] = v6
	// testRectPtr
	values[7] = testRecPtr(nil)
	// nil ptr
	var s *string
	values[8] = s
	// cyclic refs
	var n0, n1 any
	n1 = &n0
	n0 = &n1
	values[9] = n0
    // Complext pointers
	lst := newLst()
	nd1 := lst.push()
	nd2 := lst.push()
	nd1.next = nil
	nd1.lst = nil
	nd2.next = nil
	lst.root.next = nil
	values[10] = nd1*/
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Reference(t *testing.T) {
	reg, _ := registry()
	values := []any{
		func() any {
			var x any
			x = &x
			return x
		}(),
		func() any {
			var x0, x1 any
			x0 = &x1
			x1 = &x0
			return x0
		}(),
		func() any {
			var x0, x1, x2 any
			x0 = &x1
			x1 = &x2
			x2 = &x0
			return x0
		}(),
		func() any {
			var x1 any
			var x2 *any
			var x3 **any
			x2 = &x1
			x3 = &x2
			x1 = x3
			return x3
		}(),
		func() any {
			var x1 any
			var x2 *any
			var x3 **any
			x2 = &x1
			x3 = &x2
			x1 = &x3
			return x3
		}(),
	}
	checkDecodedValue(values, reg, t)
}

func checkDecodedValue(values []any, typeRegistry *TypeRegistry, t *testing.T) {
	serializer := NewSerializer().WithTypeRegistry(typeRegistry)
	unserializer := NewUnserializer().WithTypeRegistry(typeRegistry)
	for i, expected := range values {
		data := serializer.Encode(expected)
		actual, err := unserializer.Decode(data)
		if err != nil {
			t.Errorf("Test #%d: Decode(%#v) raises error: %q", i+1, expected, err)
		} else if !reflect.DeepEqual(expected, actual) {
			t.Errorf("Test #%d: Decode(%#v) returns wrong value %#v", i+1, expected, actual)
		}
	}
}

/*import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"text/scanner"
	"unsafe"

	"github.com/URALINNOVATSIYA/codec/tstpkg"
)

func TestComplex64Unserialization(t *testing.T) {
	var items = []any{
		complex(float32(0), float32(0)),
		complex(float32(1), float32(0)),
		complex(float32(0), float32(1)),
		complex(float32(1.23), float32(123)),
		complex(float32(123), float32(1.23)),
		complex(float32(1.23), float32(-1.23)),
		complex(float32(-1.23), float32(1.23)),
		testComplex64(complex(float32(1.23), float32(2))),
	}
	checkUnserializer(items, t)
}

func TestComplex128Unserialization(t *testing.T) {
	var items = []any{
		complex(float64(0), float64(0)),
		complex(float64(1), float64(0)),
		complex(float64(0), float64(1)),
		complex(1.23, float64(123)),
		complex(float64(123), 1.23),
		complex(1.23, -1.23),
		complex(-1.23, 1.23),
		testComplex128(complex(-1.23, 2.34)),
	}
	checkUnserializer(items, t)
}

func TestUintptrUnserialization(t *testing.T) {
	var items = []any{
		uintptr(123456),
		testUintptr(1234567),
	}
	checkUnserializer(items, t)
}

func TestUnsafePointerUnserialization(t *testing.T) {
	var items = []any{
		unsafe.Pointer(nil),
		unsafe.Pointer(uintptr(123456)),
		testUnsafePointer(nil),
		testUnsafePointer(uintptr(1234567)),
	}
	checkUnserializer(items, t)
}

func TestSliceUnserialization(t *testing.T) {
	var items = []any{
		([]byte)(nil),
		[]int{},
		[]int{1, -1, 0, 1234, -1234},
		[]uint{},
		[]uint{1, 0, 1234, math.MaxUint},
		[]string{"a", "ab", "abc"},
		[]bool{true, false},
		[]any{1, true, 1.23, "abc"},
		[][]bool{{false}, {}, {true, false, true}},
		testSlice{"a", "ab", "abc"},
		testGenericSlice[int]{1, 2, 3},
		testRecSlice{testRecSlice{}, testRecSlice{testRecSlice{}}},
	}
	checkUnserializer(items, t)
}

func TestArrayUnserialization(t *testing.T) {
	var items = []any{
		[0]int{},
		[1]int{-1},
		[3]uint{1, 2, 3},
		[4]bool{true, false, false, true},
		[3][]bool{{false}, {}, {true, false, true}},
		*(*[256]byte)(bytes.Repeat([]byte{1}, 256)),
		testArray{-1, 2, -3},
		testGenericArray[float32]{0, -1.23},
	}
	checkUnserializer(items, t)
}

func TestMapUnserialization(t *testing.T) {
	var items = []any{
		(map[byte]byte)(nil),
		map[string]bool{"a": true, "b": false},
		map[int32]int8{-1: 2, 3: -4, -123456: -128},
		testMap{"a": 1, "b": 2, "c": 3},
		testGenericMap[string, int]{"a": 1, "b": -2, "": 0},
		testRecMap{8: testRecMap{2: testRecMap{}, 1: testRecMap{}}},
	}
	checkUnserializer(items, t)
}

func TestChanUnserialization(t *testing.T) {
	var items = []any{
		(chan<- []int8)(nil),
		make(chan int),
		make(chan<- bool, 1),
		make(<-chan *testStruct, 2),
		make(testChan, 10),
	}
	checkUnserializer(items, t)
}

func TestFuncUnserialization(t *testing.T) {
	var items = []any{
		(func(bool, uint))(nil),
		Serialize,
		Unserialize,
	}
	checkUnserializer(items, t)
}

func TestStructUnserialization(t *testing.T) {
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
	var items = []any{
		s,
		struct {
			f1 bool
			f2 byte
		}{
			true,
			123,
		},
	}
	SetStructCodingMode(StructCodingModeIndex)
	checkUnserializer(items, t)
	SetStructCodingMode(StructCodingModeName)
	checkUnserializer(items, t)
	SetStructCodingMode(StructCodingModeDefault)
	checkUnserializer(items, t)
}

func TestStructWithPrivateFieldsUnserialization(t *testing.T) {
	scan := scanner.Scanner{}
	scan.Init(strings.NewReader("test"))
	wg := &sync.WaitGroup{}
	wg.Add(5)
	var items = []any{
		errors.New("err"),
		scan,
		strings.NewReplacer("test1", "test2"),
		wg,
	}
	SetStructCodingMode(StructCodingModeIndex)
	checkUnserializer(items, t)
	SetStructCodingMode(StructCodingModeName)
	checkUnserializer(items, t)
	SetStructCodingMode(StructCodingModeDefault)
	checkUnserializer(items, t)
}

func TestUnexportedTypeUnserialize(t *testing.T) {
	expected := tstpkg.Get()
	data := Serialize(expected)
	actual, err := Unserialize(data)
	if err != nil {
		t.Errorf("Unserializer::decode(%#v) raises error: %q", expected, err)
	} else if !tstpkg.Check(actual) {
		t.Errorf("Unserializer::decode(%#v) returns wrong value %#v", expected, actual)
	}
}

func TestSerializableUnserialization(t *testing.T) {
	var items = []any{
		testCustomUint(123),
		testCustomStruct{
			f1: true,
			f2: "abc",
			f3: 123,
		},
		&testCustomStruct{
			f1: true,
			f2: "abc",
			f3: 123,
		},
		&testCustomPointerStruct{
			f1: true,
			f2: "abc",
			f3: 123,
		},
		testCustomPointerStruct{
			f1: true,
			f2: "abc",
			f3: 123,
		},
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
	}
	checkUnserializer(items, t)
}

func TestInterfaceUnserialization(t *testing.T) {
	var items = []any{
		testInterface{
			p: &testStruct{},
		},
	}
	checkUnserializer(items, t)
}

func TestPointerUnserialization(t *testing.T) {
	values := make([]any, 11)
	// *Nil
	values[0] = (*any)(nil)
	// *Bool
	v1 := true
	values[1] = &v1
	v2 := false
	values[2] = &v2
	// *String
	v3 := ""
	values[3] = &v3
	v4 := "abc"
	values[4] = &v4
	// *[]uint
	v5 := []uint{1, 2, 3}
	values[5] = &v5
	// testPtr
	b := true
	v6 := testPtr(&b)
	values[6] = v6
	// testRectPtr
	values[7] = testRecPtr(nil)
	// nil ptr
	var s *string
	values[8] = s
	// cyclic refs
	var n0, n1 any
	n1 = &n0
	n0 = &n1
	values[9] = n0
    // Complext pointers
	/*lst := newLst()
	nd1 := lst.push()
	nd2 := lst.push()
	nd1.next = nil
	nd1.lst = nil
	nd2.next = nil
	lst.root.next = nil
	values[10] = nd1
	checkUnserializer([]any{[]any{testRecPtr(nil), nil}}, t)
}

func TestRecursiveReferenceUnserialization(t *testing.T) {
	var x any
	x = &x

	y := unserializedValue(x, t)

	if y.(*any) != *y.(*any) || y.(*any) != *(*y.(*any)).(*any) {
		t.Errorf("Unserializer::decode(%#v): variable does not point to itself", x)
	}
}

func TestReferenceOnCommonDataUnserialization(t *testing.T) {
	a := make([]*byte, 3)
	d := byte(123)
	a[0] = &d
	a[1] = &d
	a[2] = &d

	b := unserializedValue(a, t).([]*byte)

	*b[1] = 0
	if *b[0] != *b[2] || *b[0] != 0 {
		t.Errorf("Unserializer::decode(%#v): elements do not point to the common data", a)
	}
}

func TestReferenceOrderUnserialization(t *testing.T) {
	a := make([]any, 7)
	a[0] = &a[1]
	a[1] = &a[3]
	a[2] = ">"
	a[3] = 123
	a[4] = "<"
	a[5] = &a[3]
	a[6] = &a[5]

	b := unserializedValue(a, t).([]any)

	b[3] = 321
	if b[0] != &b[1] {
		t.Errorf("Unserializer::decode(%#v): element #0 does not point to element #1", a)
	}
	if b[6] != &b[5] {
		t.Errorf("Unserializer::decode(%#v): element #6 does not point to element #5", a)
	}
	if (*b[1].(*any)) != 321 {
		t.Errorf("Unserializer::decode(%#v): element #1 does not point to element #2", a)
	}
	if (*b[5].(*any)) != 321 {
		t.Errorf("Unserializer::decode(%#v): element #5 does not point to element #2", a)
	}
}

func TestMixedReferenceUnserialization(t *testing.T) {
	s := &testStruct{
		F1: "abc",
		F2: true,
		F3: nil,
		F4: nil,
		f5: 321,
		f6: "#",
	}
	s.F3 = s
	s.F4 = []any{&s.F1, nil, [3]byte{1, 2, 3}, s, nil}
	s.F4.([]any)[1] = &s.F4.([]any)[2]
	s.F4.([]any)[4] = &s.f6

	p := unserializedValue(s, t).(*testStruct)

	if p.F3 != p {
		t.Errorf("Unserializer::decode(%#v): field F3 does not point to the structure itself", s)
	}
	if p.F4.([]any)[0] != &p.F1 {
		t.Errorf("Unserializer::decode(%#v): element #0 of F4 does not point to F1", s)
	}
	if p.F4.([]any)[1] != &p.F4.([]any)[2] {
		t.Errorf("Unserializer::decode(%#v): element #1 of F4 does not point to its element #2", s)
	}
	if p.F4.([]any)[3] != p {
		t.Errorf("Unserializer::decode(%#v): element #3 of F4 does not point to the structure itself", s)
	}
	if p.F4.([]any)[4] != &p.f6 {
		t.Errorf("Unserializer::decode(%#v): element #4 of F4 does not point to f6", s)
	}
}

func checkUnserializer(values []any, t *testing.T) {
	registerTestTypes()
	serializer := NewSerializer(GetStructCodingMode())
	unserializer := NewUnserializer(GetStructCodingMode())
	for _, expected := range values {
		data := serializer.Encode(expected)
		actual, err := unserializer.Decode(data)
		if err != nil {
			t.Errorf("Unserializer::decode(%#v) raises error: %q", expected, err)
		} else {
			var equals bool
			if expected != nil {
				switch reflect.TypeOf(expected).Kind() {
				case reflect.Chan:
					equals = channelEqual(expected, actual)
				case reflect.Func:
					equals = funcEqual(expected, actual)
				default:
					equals = reflect.DeepEqual(expected, actual)
				}
			} else {
				equals = reflect.DeepEqual(expected, actual)
			}
			if !equals {
				t.Errorf("Unserializer::decode(%#v) returns wrong value %#v", expected, actual)
			}
		}
	}
}

func channelEqual(expected any, actual any) bool {
	if tChecker.fullTypeName(reflect.TypeOf(expected)) != tChecker.fullTypeName(reflect.TypeOf(actual)) {
		return false
	}
	return reflect.ValueOf(expected).Cap() == reflect.ValueOf(actual).Cap()
}

func funcEqual(expected any, actual any) bool {
	return funcName(reflect.ValueOf(expected)) == funcName(reflect.ValueOf(actual))
}

func unserializedValue(expected any, t *testing.T) any {
	registerTestTypes()
	data := Serialize(expected)
	actual, err := Unserialize(data)
	if err != nil {
		t.Errorf("Unserializer::decode(%#v) raises error: %q", expected, err)
	} else if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unserializer::decode(%#v) returns wrong value %#v", expected, actual)
	}
	return actual
}
*/
