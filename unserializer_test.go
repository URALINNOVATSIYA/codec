package codec

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"
	"unsafe"
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

func TestUnserialization_Complex64(t *testing.T) {
	reg, _ := registry()
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
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Complex128(t *testing.T) {
	reg, _ := registry()
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
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Uintptr(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		uintptr(123456),
		testUintptr(1234567),
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_UnsafePointer(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		unsafe.Pointer(nil),
		unsafe.Pointer(uintptr(123456)),
		testUnsafePointer(nil),
		testUnsafePointer(uintptr(1234567)),
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Func(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		(func(bool, uint))(nil),
		Serialize,
		Unserialize,
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Chan(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		(chan<- []int8)(nil),
		make(chan int),
		make(chan<- bool, 1),
		make(<-chan *testStruct, 2),
		make(testChan, 10),
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Slice(t *testing.T) {
	reg, _ := registry()
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
		testRecSlice{testRecSlice{}, testRecSlice{testRecSlice{}}},
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Array(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		[0]int{},
		[1]int{-1},
		[3]uint{1, 2, 3},
		[4]bool{true, false, false, true},
		[2][2]string{{"a", "b"}, {"c", "d"}},
		[3][]bool{{false}, {}, {true, false, true}},
		*(*[256]byte)(bytes.Repeat([]byte{1}, 256)),
		testArray{-1, 2, -3},
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Map(t *testing.T) {
	reg, _ := registry()
	var items = []any{
		(map[byte]byte)(nil),
		map[string]bool{"a": true, "b": false},
		map[int32]int8{-1: 2, 3: -4, -123456: -128},
		map[byte]map[int]bool{5: {-7: true, -9: false}, 1: nil, 0: {}, 100: {2: false}},
		testMap{"a": 1, "b": 2, "c": 3},
		testRecMap{8: testRecMap{2: testRecMap{}, 1: testRecMap{}}},
	}
	checkDecodedValue(items, reg, t)
}

func TestUnserialization_Pointer(t *testing.T) {
	reg, _ := registry()
	values := []any{
		(*any)(nil),
		(*string)(nil),
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
		func() any {
			x := []uint{1, 2, 3}
			return &x
		}(),
		func() any {
			b := true
			return testBoolPtr(&b)
		}(),
		testRecPtr(nil),
		func() any {
			var x testRecPtr
			return testRecPtr(&x)
		}(),
	}
	checkDecodedValue(values, reg, t)
}

func TestUnserialization_Reference(t *testing.T) {
	reg, _ := registry()
	values := []any{
		// #1
		func() any {
			var x any
			x = &x
			return x
		}(),
		// #2
		func() any {
			var x testRecPtr
			x = &x
			return x
		}(),
		// #3
		func() any {
			var x1, x2 any
			x1 = &x2
			x2 = &x1
			return x1
		}(),
		// #4
		func() any {
			var x1, x2 testRecPtr
			x1 = &x2
			x2 = &x1
			return x1
		}(),
		// #5
		func() any {
			var x1, x2, x3 any
			x1 = &x2
			x2 = &x3
			x3 = &x1
			return x1
		}(),
		// #6
		func() any {
			var x1, x2, x3 testRecPtr
			x1 = &x2
			x2 = &x3
			x3 = &x1
			return x1
		}(),
		// #7
		func() any {
			var x1 any
			var x2 *any
			var x3 **any
			x2 = &x1
			x3 = &x2
			x1 = x3
			return x3
		}(),
		// #8
		func() any {
			var x1 any
			var x2 *any
			var x3 **any
			x2 = &x1
			x3 = &x2
			x1 = &x3
			return x3
		}(),
		// #9
		func() any {
			b := true
			return []*bool{&b, &b, &b}
		}(),
		// #10
		func() any {
			x := []any{nil, true, nil}
			x[0] = &x[1]
			x[2] = &x[1]
			return x
		}(),
		// #11
		func() any {
			x := []any{nil}
			x[0] = &x[0]
			return x
		}(),
		// #12
		func() any {
			x := []any{nil, nil}
			x[0] = &x[1]
			x[1] = &x[0]
			return x
		}(),
		// #13
		func() any {
			x := []any{nil, nil, nil}
			x[0] = &x[1]
			x[1] = &x[2]
			x[2] = &x[0]
			return x
		}(),
		// #14
		func() any {
			x := []any{nil}
			x[0] = x
			return x
		}(),
		// #15
		func() any {
			x := []any{nil, nil}
			x[1] = x
			return x
		}(),
		// #16
		func() any {
			x := []any{nil, nil}
			x[0] = &x[1]
			x[1] = x
			return x
		}(),
		// #17
		func() any {
			x := []any{nil, nil, nil}
			x[0] = &x[1]
			x[1] = &x[2]
			x[2] = x
			return x
		}(),
		// #18
		func() any {
			x := []any{nil, []any{nil}}
			x[0] = &x[1].([]any)[0]
			x[1].([]any)[0] = x
			return x
		}(),
	}
	// Complext pointers
	/*lst := newLst()
	nd1 := lst.push()
	nd2 := lst.push()
	nd1.next = nil
	nd1.lst = nil
	nd2.next = nil
	lst.root.next = nil
	values[10] = nd1*/
	checkDecodedValue(values, reg, t)
}

func checkDecodedValue(values []any, typeRegistry *TypeRegistry, t *testing.T) {
	serializer := NewSerializer().WithTypeRegistry(typeRegistry)
	unserializer := NewUnserializer().WithTypeRegistry(typeRegistry)
	for i, expected := range values {
		data := serializer.Encode(expected)
		actual, err := unserializer.Decode(data)
		if err != nil {
			t.Errorf("Test #%d: Decode(%T) raises error: %q", i+1, expected, err)
		} else  {
			var equals bool
			if expected != nil {
				switch reflect.TypeOf(expected).Kind() {
				case reflect.Chan:
					equals = channelEqual(typeRegistry, expected, actual)
				case reflect.Func:
					equals = funcEqual(expected, actual)
				default:
					equals = reflect.DeepEqual(expected, actual)
				}
			} else {
				equals = reflect.DeepEqual(expected, actual)
			}
			if !equals {
				t.Errorf("Test #%d: Decode(%T) returns wrong value %T", i+1, expected, actual)
			}
		}
	}
}

func channelEqual(reg *TypeRegistry, expected any, actual any) bool {
	if reg.typeName(reflect.TypeOf(expected)) != reg.typeName(reflect.TypeOf(actual)) {
		return false
	}
	return reflect.ValueOf(expected).Cap() == reflect.ValueOf(actual).Cap()
}

func funcEqual(expected any, actual any) bool {
	return funcName(reflect.ValueOf(expected)) == funcName(reflect.ValueOf(actual))
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
