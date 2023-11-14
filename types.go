package codec

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

const (
	tNil       byte = 0b0000_0000
	tBool      byte = 0b0001_0000
	tByte      byte = 0b0010_0000
	tInt8      byte = 0b0010_0000
	tInt16     byte = 0b0010_0001
	tInt32     byte = 0b0010_0010
	tInt64     byte = 0b0010_0011
	tInt       byte = 0b0011_0000
	tFloat     byte = 0b0100_0000
	tComplex   byte = 0b0101_0000
	tPointer   byte = 0b0110_0000
	tString    byte = 0b0111_0000
	tList      byte = 0b1000_0000
	tMap       byte = 0b1001_0000
	tUintptr   byte = 0b1010_0000
	tRef       byte = 0b1011_0000
	tStruct    byte = 0b1100_0000
	tChan      byte = 0b1101_0000
	tFunc      byte = 0b1101_1000
	tType      byte = 0b1110_0000
	tInterface byte = 0b1111_0000
)

const (
	version byte = 0b0000_0001 // 1 - version of the serializer
	signed  byte = 0b0000_1000 // for signed and unsigned integers
	meta    byte = 0b0000_0100 // for fixed size integers to determine whether the integer representation has meta information about its byte size
	wide    byte = 0b0000_1000 // == 1 for larger bit representations (for floats and complex numbers)
	tru     byte = 0b0000_0001 // for true booleans
	fixed   byte = 0b0000_1000 // for lists that are arrays (i.e. have fixed size)
	raw     byte = 0b0000_1000 // determines unsafe.Pointer for tUintptr
	val     byte = 0b0000_1000 // determines a reference that contains a value that some pointer points to
	custom  byte = 0b0000_1000 // determines whether the type implements custom serialization
	null    byte = 0b0000_0100 // determines whether underlying type value is nil
	mask    byte = 0b1111_0000 // type mask
)

// Booleans
// ---------------------------------------------------------------------------------------------------------------------
// true:  tBool|tru
// false: tBool

// Strings
// ---------------------------------------------------------------------------------------------------------------------
// tString|(length's size in bytes - 1) + length bytes + string bytes

// Fixed size integers
// ---------------------------------------------------------------------------------------------------------------------
// int8:
// tInt8|signed + byte

// uint8:
// tInt8 + byte

// int16
// tInt16|signed|meta + byte        (if -128 <= number <= 127)
// tInt16|signed      + byte + byte (if number > 127 or number < -128)

// uint16:
// tInt16|meta + byte        (if number <= 255)
// tInt16      + byte + byte (if number > 255)

// int32:
// tInt32|signed|meta + 0b01 + null bits + number bits (if min number bits <= 6)
// tInt32|signed|meta + 0b10 + null bits + number bits (if 6 < min number bits <= 14)
// tInt32|signed|meta + 0b11 + null bits + number bits (if 14 < min number bits <= 22)
// tInt32|signed      + number bytes (if 22 < min number bits <= 32)

// uint32:
// tInt32|meta + 0b01 + null bits + number bits (if min number bits <= 6)
// tInt32|meta + 0b10 + null bits + number bits (if 6 < min number bits <= 14)
// tInt32|meta + 0b11 + null bits + number bits (if 14 < min number bits <= 22)
// tInt32      + number bytes (if 22 < min number bits <= 32)

// int64:
// tInt64|signed|meta + 0b001 + null bits + number bits (if min number bits <= 5)
// tInt64|signed|meta + 0b010 + null bits + number bits (if 5 < min number bits <= 13)
// tInt64|signed|meta + 0b011 + null bits + number bits (if 13 < min number bits <= 21)
// tInt64|signed|meta + 0b100 + null bits + number bits (if 21 < min number bits <= 29)
// tInt64|signed|meta + 0b101 + null bits + number bits (if 29 < min number bits <= 37)
// tInt64|signed|meta + 0b110 + null bits + number bits (if 37 < min number bits <= 45)
// tInt64|signed|meta + 0b110 + null bits + number bits (if 45 < min number bits <= 53)
// tInt64|signed      + number bytes (if 53 < min number bits <= 64)
// uint64:
// tInt64|meta + 0b001 + null bits + number bits (if min number bits <= 5)
// tInt64|meta + 0b010 + null bits + number bits (if 5 < min number bits <= 13)
// tInt64|meta + 0b011 + null bits + number bits (if 13 < min number bits <= 21)
// tInt64|meta + 0b100 + null bits + number bits (if 21 < min number bits <= 29)
// tInt64|meta + 0b101 + null bits + number bits (if 29 < min number bits <= 37)
// tInt64|meta + 0b110 + null bits + number bits (if 37 < min number bits <= 45)
// tInt64|meta + 0b110 + null bits + number bits (if 45 < min number bits <= 53)
// tInt64      + number bytes (if 53 < min number bits <= 64)

// Platform dependent integers
// ---------------------------------------------------------------------------------------------------------------------
// int
// tInt|signed|(number's byte count - 1) + number bytes

// uint
// tInt|(number's byte count - 1) + number bytes

// uintptr
// tUintptr|(pointer's byte count - 1) + pointer bytes

// unsafe.Pointer
// tUintptr|raw|(pointer's byte count - 1) + pointer bytes

// Floating point numbers
// ---------------------------------------------------------------------------------------------------------------------
// float32:
// tFloat|(number's byte count - 1) + number bytes

// float64:
// tFloat|wide|(number's byte count - 1) + number bytes

// complex64:
// tComplex + encoded real part + encoded imaginary part

// complex128:
// tComplex|wide + encoded real part + encoded imaginary part

// Lists and maps
// ---------------------------------------------------------------------------------------------------------------------
// slices
// tType + type id + tList|(length's size in bytes - 1) + encoded elements

// arrays
// tType + type id + tList|fixed|(length's size in bytes - 1) + encoded elements

// maps
// tType + type id + tMap|(length's size in bytes - 1) + encoded key/value pairs

// Structs
// ---------------------------------------------------------------------------------------------------------------------
// tType + type id + tStruct|(size of number of fields in bytes - 1) + encoded fields

// Channels
// ---------------------------------------------------------------------------------------------------------------------
// tType + type id + tChan|reflect.ChanDir + encoded (as uint) channel capacity

// Functions
// ---------------------------------------------------------------------------------------------------------------------
// tType + func id + tFunc

// Any serializable objects
// ---------------------------------------------------------------------------------------------------------------------
// tType|custom + type id + underlying base type (tBool, tInt, etc)|(size of custom encoding's length in bytes - 1)
// + custom encoding

const numOfKnownTypes = 19

var serializableInterfaceType = reflect.TypeOf((*Serializable)(nil)).Elem()

func isSerializable(v reflect.Value) bool {
	if v.IsValid() {
		t := v.Type()
		if isSimplePointer(t) {
			return false
		}
		return t.Implements(serializableInterfaceType)
	}
	return false
}

func isSimplePointer(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer && isCommonType(t)
}

func isCommonType(t reflect.Type) bool {
	name := t.Name()
	if name == "" {
		return true
	}
	if name == t.Kind().String() {
		return true
	}
	if t.Kind() == reflect.UnsafePointer && name == "Pointer" {
		return true
	}
	return false
}

func funcName(f reflect.Value) string {
	if f.Kind() != reflect.Func {
		return ""
	}
	return runtime.FuncForPC(f.Pointer()).Name()
}

func changeValue(value reflect.Value, newValue any) {
	value.Set(reflect.ValueOf(newValue).Convert(value.Type()))
}

type typeChecker struct {
	initiated      bool
	typeAutoReg    bool
	types          map[string]reflect.Type  // registered types
	funcs          map[string]reflect.Value // registered functions
	typeSignatures map[int][]byte           // type signatures
	typeNames      map[string]int           // type full names
	tmx            sync.RWMutex
	nmx            sync.RWMutex
}

var tChecker = &typeChecker{
	typeAutoReg:    true,
	types:          make(map[string]reflect.Type, numOfKnownTypes),
	funcs:          make(map[string]reflect.Value),
	typeSignatures: make(map[int][]byte, numOfKnownTypes),
	typeNames:      make(map[string]int),
}

func (ch *typeChecker) init() {
	if ch.initiated {
		return
	}
	ch.initiated = true
	types := []reflect.Type{
		reflect.TypeOf([]any{}).Elem(), // interface {}
		reflect.TypeOf(false),
		reflect.TypeOf(""),
		reflect.TypeOf(int8(0)),
		reflect.TypeOf(uint8(0)),
		reflect.TypeOf(int16(0)),
		reflect.TypeOf(uint16(0)),
		reflect.TypeOf(int32(0)),
		reflect.TypeOf(uint32(0)),
		reflect.TypeOf(int64(0)),
		reflect.TypeOf(uint64(0)),
		reflect.TypeOf(0),
		reflect.TypeOf(uint(0)),
		reflect.TypeOf(float32(0)),
		reflect.TypeOf(float64(0)),
		reflect.TypeOf(complex64(0)),
		reflect.TypeOf(complex128(0)),
		reflect.TypeOf(uintptr(0)),
		reflect.TypeOf(unsafe.Pointer(uintptr(0))),
	}
	for _, t := range types {
		ch.registerType(t)
	}
}

func (ch *typeChecker) registerTypeOf(v any) []byte {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Func {
		return ch.registerFunc(reflect.ValueOf(v))
	}
	return ch.registerType(reflect.TypeOf(v))
}

func (ch *typeChecker) registerType(t reflect.Type) []byte {
	if isSimplePointer(t) {
		return []byte{tPointer}
	}
	ch.tmx.Lock()
	s, id := ch.formTypeSignature(t)
	ch.types[string(s)] = t
	ch.typeSignatures[id] = s
	ch.tmx.Unlock()
	return append([]byte{}, s...)
}

func (ch *typeChecker) registerFunc(f reflect.Value) []byte {
	ch.tmx.Lock()
	s, id := ch.formFuncSignature(f)
	if f.IsNil() {
		ch.types[string(s)] = f.Type()
	} else {
		ch.funcs[string(s)] = f
	}
	ch.typeSignatures[id] = s
	ch.tmx.Unlock()
	return append([]byte{}, s...)
}

func (ch *typeChecker) typeSignatureOf(v reflect.Value) []byte {
	if !v.IsValid() {
		return []byte{tNil}
	}
	if v.Kind() == reflect.Func {
		return ch.funcSignature(v)
	}
	return ch.typeSignature(v.Type())
}

func (ch *typeChecker) typeSignature(t reflect.Type) []byte {
	if t == nil {
		return []byte{tNil}
	}
	if isSimplePointer(t) {
		return []byte{tPointer}
	}
	ch.tmx.RLock()
	id := ch.typeId(t)
	signature, exists := ch.typeSignatures[id]
	if exists {
		ch.tmx.RUnlock()
		return append([]byte{}, signature...)
	}
	ch.tmx.RUnlock()
	if ch.typeAutoReg {
		return ch.registerType(t)
	}
	panic(fmt.Sprintf("unregistered type: %s", ch.fullTypeName(t)))
}

func (ch *typeChecker) funcSignature(f reflect.Value) []byte {
	ch.tmx.RLock()
	id := ch.funcId(f)
	signature, exists := ch.typeSignatures[id]
	if exists {
		ch.tmx.RUnlock()
		return append([]byte{}, signature...)
	}
	ch.tmx.RUnlock()
	if ch.typeAutoReg {
		return ch.registerFunc(f)
	}
	panic(fmt.Sprintf("unregistered function: %s", funcName(f)))
}

func (ch *typeChecker) formTypeSignature(t reflect.Type) ([]byte, int) {
	if t == nil {
		return []byte{tNil}, -1
	}
	var ut byte
	switch t.Kind() {
	case reflect.Invalid:
		ut = tNil
	case reflect.Bool:
		ut = tBool
	case reflect.Uint8:
		ut = tInt8
	case reflect.Int8:
		ut = tInt8 | signed
	case reflect.Uint16:
		ut = tInt16
	case reflect.Int16:
		ut = tInt16 | signed
	case reflect.Uint32:
		ut = tInt32
	case reflect.Int32:
		ut = tInt32 | signed
	case reflect.Uint64:
		ut = tInt64
	case reflect.Int64:
		ut = tInt64 | signed
	case reflect.Uint:
		ut = tInt
	case reflect.Int:
		ut = tInt | signed
	case reflect.String:
		ut = tString
	case reflect.Float32:
		ut = tFloat
	case reflect.Float64:
		ut = tFloat | wide
	case reflect.Complex64:
		ut = tComplex
	case reflect.Complex128:
		ut = tComplex | wide
	case reflect.Uintptr:
		ut = tUintptr
	case reflect.UnsafePointer:
		ut = tUintptr | raw
	case reflect.Interface:
		ut = tInterface
	case reflect.Pointer:
		ut = tPointer
		goto encodeTypeName
	case reflect.Slice:
		ut = tList
		goto encodeTypeName
	case reflect.Array:
		ut = tList | fixed
		goto encodeTypeName
	case reflect.Map:
		ut = tMap
		goto encodeTypeName
	case reflect.Struct:
		ut = tStruct
		goto encodeTypeName
	case reflect.Chan:
		ut = tChan | byte(t.ChanDir())
		goto encodeTypeName
	case reflect.Func:
		ut = tFunc
		goto encodeTypeName
	default:
		panic(fmt.Sprintf("unsupported type kind %q", t.Kind()))
	}
	if isCommonType(t) {
		id := ch.typeId(t)
		return []byte{ut}, id
	}
encodeTypeName:
	return ch.encodedTypeName(t, ut)
}

func (ch *typeChecker) formFuncSignature(f reflect.Value) ([]byte, int) {
	return ch.encodedTypeId(ch.funcId(f), tFunc)
}

func (ch *typeChecker) encodedTypeName(t reflect.Type, underlyingType byte) ([]byte, int) {
	return ch.encodedTypeId(ch.typeId(t), underlyingType)
}

func (ch *typeChecker) encodedTypeId(id int, underlyingType byte) ([]byte, int) {
	byteSize, idBytes := toMinBytes(uint64(id))
	b := make([]byte, 1, byteSize+2)
	b[0] = tType | byte(byteSize-1)
	return append(append(b, idBytes...), underlyingType), id
}

func (ch *typeChecker) typeId(t reflect.Type) int {
	return ch.nameToId(ch.fullTypeName(t))
}

func (ch *typeChecker) funcId(f reflect.Value) int {
	name := funcName(f)
	if name != "" {
		return ch.nameToId(name)
	}
	return ch.typeId(f.Type())
}

func (ch *typeChecker) nameToId(name string) int {
	ch.nmx.RLock()
	id, exists := ch.typeNames[name]
	ch.nmx.RUnlock()
	if exists {
		return id
	}
	ch.nmx.Lock()
	id, exists = ch.typeNames[name]
	if !exists {
		id = len(ch.typeNames)
		ch.typeNames[name] = id
	}
	ch.nmx.Unlock()
	return id
}

func (ch *typeChecker) fullTypeName(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	if t.Name() != "" {
		name := t.Name()
		if t.PkgPath() != "" {
			name = t.PkgPath() + "." + name
		}
		return name
	}
	switch t.Kind() {
	case reflect.Invalid:
		return "<nil>"
	case reflect.Pointer:
		return "*" + ch.fullTypeName(t.Elem())
	case reflect.Slice:
		return "[]" + ch.fullTypeName(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), ch.fullTypeName(t.Elem()))
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", ch.fullTypeName(t.Key()), ch.fullTypeName(t.Elem()))
	case reflect.Struct:
		f := ""
		for i, numFields := 0, t.NumField(); i < numFields; i++ {
			field := t.Field(i)
			if i > 0 {
				f += "; "
			}
			f += fmt.Sprintf("%s %s", field.Name, ch.fullTypeName(t.Field(i).Type))
		}
		return fmt.Sprintf("struct { %s }", f)
	case reflect.Chan:
		switch t.ChanDir() {
		case reflect.BothDir:
			return "chan " + ch.fullTypeName(t.Elem())
		case reflect.RecvDir:
			return "<-chan" + ch.fullTypeName(t.Elem())
		default:
			return "chan<-" + ch.fullTypeName(t.Elem())
		}
	}
	return t.String()
}

func (ch *typeChecker) valueOf(typeSignature []byte) (reflect.Value, error) {
	if typeSignature[len(typeSignature)-1] == tFunc {
		f, err := ch.funcOf(typeSignature)
		if err == nil {
			return f, nil
		}
	}
	t, err := ch.typeOf(typeSignature)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.New(t).Elem(), nil
}

func (ch *typeChecker) typeOf(typeSignature []byte) (reflect.Type, error) {
	ch.tmx.RLock()
	t, exists := ch.types[*(*string)(unsafe.Pointer(&typeSignature))]
	ch.tmx.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unrecognized type with signature: %v", typeSignature)
	}
	return t, nil
}

func (ch *typeChecker) funcOf(typeSignature []byte) (reflect.Value, error) {
	ch.tmx.RLock()
	f, exists := ch.funcs[*(*string)(unsafe.Pointer(&typeSignature))]
	ch.tmx.RUnlock()
	if !exists {
		return reflect.Value{}, fmt.Errorf("unrecognized function with signature: %v", typeSignature)
	}
	return f, nil
}

func RegisterType(t reflect.Type) {
	tChecker.registerType(t)
}

func RegisterTypeOf(value any) {
	tChecker.registerTypeOf(value)
}

func TurnOffTypeAutoRegistration() {
	tChecker.typeAutoReg = false
}

func TurnOnTypeAutoRegistration() {
	tChecker.typeAutoReg = true
}

func id(v any) byte {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Func {
		return byte(tChecker.funcId(reflect.ValueOf(v)))
	}
	return byte(tChecker.typeId(t))
}

func init() {
	tChecker.init()
}

func GetRegisteredTypeNames() []string {
	var typeNames []string

	for typeName := range tChecker.typeNames {
		typeNames = append(typeNames, typeName)
	}
	return typeNames
}
