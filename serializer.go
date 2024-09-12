package codec

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"
	//"unsafe"
)

const (
	meta_ref   byte = 0b0000_0000 // pseudo type for referenced values
	meta_nil   byte = 0b0001_0000 // determines whether underlying value is nil
	meta_fixed byte = 0b0010_0000 // for lists that are arrays (i.e. have fixed size)
	meta_prf   byte = 0b0100_0000 //
)

type Serializer struct {
	refs             map[reflect.Value]uint32
	cnt              uint32
	typeRegistry     *TypeRegistry
	structCodingMode byte
}

func NewSerializer() *Serializer {
	return &Serializer{
		typeRegistry:     GetDefaultTypeRegistry(),
		structCodingMode: GetDefaultStructCodingMode(),
	}
}

func (s *Serializer) WithOptions(options []any) *Serializer {
	for _, option := range options {
		if v, ok := option.(byte); ok {
			s.WithStructCodingMode(v)
			continue
		}
		if v, ok := option.(*TypeRegistry); ok {
			s.WithTypeRegistry(v)
			continue
		}
		panic(fmt.Errorf("invalid option type %T", option))
	}
	return s
}

func (s *Serializer) WithTypeRegistry(registry *TypeRegistry) *Serializer {
	s.typeRegistry = registry
	return s
}

func (s *Serializer) WithStructCodingMode(mode byte) *Serializer {
	s.structCodingMode = mode
	return s
}

func (s *Serializer) Encode(value any) []byte {
	s.cnt = 0
	s.refs = make(map[reflect.Value]uint32)
	bytes, _ := s.encode(reflect.ValueOf(value))
	return append([]byte{version}, bytes...)
}

func (s *Serializer) encode(v reflect.Value) ([]byte, bool) {
	value, isReferencedValue := s.encodeValue(v)
	if isReferencedValue {
		return value, true
	}
	return append(s.encodeType(v), value...), false
}

func (s *Serializer) encodeType(v reflect.Value) []byte {
	id := s.typeRegistry.typeIdByValue(v)
	return asBytesWithSize(uint64(id), 3)
}

func (s *Serializer) encodeValue(v reflect.Value) (bytes []byte, isReferencedValue bool) {
	if cnt, ok := s.refs[v]; ok {
		return s.encodeReference(cnt), true
	}
	s.refs[v] = s.cnt
	s.cnt++
	if isSerializable(v) {
		return s.encodeSerializable(v), false
	}
	switch v.Kind() {
	case reflect.Invalid: // nil
		bytes = s.encodeNil()
	case reflect.Bool:
		bytes = s.encodeBool(v)
	case reflect.String:
		bytes = s.encodeString(v)
	case reflect.Uint8:
		bytes = s.encodeUint8(v)
	case reflect.Int8:
		bytes = s.encodeInt8(v)
	case reflect.Uint16:
		bytes = s.encodeUint16(v)
	case reflect.Int16:
		bytes = s.encodeInt16(v)
	case reflect.Uint32:
		bytes = s.encodeUint32(v)
	case reflect.Int32:
		bytes = s.encodeInt32(v)
	case reflect.Uint64:
		bytes = s.encodeUint64(v)
	case reflect.Int64:
		bytes = s.encodeInt(v)
	case reflect.Uint:
		bytes = s.encodeUint(v)
	case reflect.Int:
		bytes = s.encodeInt(v)
	case reflect.Float32:
		bytes = s.encodeFloat32(v)
	case reflect.Float64:
		bytes = s.encodeFloat64(v)
	case reflect.Complex64:
		bytes = s.encodeComplex64(v)
	case reflect.Complex128:
		bytes = s.encodeComplex128(v)	
	case reflect.Uintptr:
		bytes = s.encodeUintptr(v)
	case reflect.UnsafePointer:
		bytes = s.encodeUnsafePointer(v)
	case reflect.Chan:
		bytes = s.encodeChan(v)
	case reflect.Func:
		bytes = s.encodeFunc(v)
	case reflect.Slice, reflect.Array:
		bytes = s.encodeList(v)
	case reflect.Map:
		bytes = s.encodeMap(v)
	case reflect.Struct:
		bytes = s.encodeStruct(v)
	case reflect.Interface:
		bytes, isReferencedValue = s.encodeInterface(v)
	case reflect.Pointer:
		bytes = s.encodePointer(v)
	default:
		panic(fmt.Errorf("unsupported type kind %q", v.Kind()))
	}
	return
}

func (s *Serializer) encodeNil() []byte {
	return []byte{meta_nil}
}

func (s *Serializer) encodeBool(v reflect.Value) []byte {
	if v.Bool() {
		return []byte{1}
	}
	return []byte{0}
}

func (s *Serializer) encodeString(v reflect.Value) []byte {
	return append(s.encodeCount(v.Len()), v.String()...)
}

func (s *Serializer) encodeUint8(v reflect.Value) []byte {
	return []byte{uint8(v.Uint())}
}

func (s *Serializer) encodeInt8(v reflect.Value) []byte {
	return []byte{uint8(toUint(v.Int()))}
}

func (s *Serializer) encodeUint16(v reflect.Value) []byte {
	return asBytesWithSize(v.Uint(), 2)
}

func (s *Serializer) encodeInt16(v reflect.Value) []byte {
	return asBytesWithSize(toUint(v.Int()), 2)
}

func (s *Serializer) encodeUint32(v reflect.Value) []byte {
	return asBytesWithSize(v.Uint(), 3)
}

func (s *Serializer) encodeInt32(v reflect.Value) []byte {
	return asBytesWithSize(toUint(v.Int()), 3)
}

func (s *Serializer) encodeUint64(v reflect.Value) []byte {
	return asBytesWithSize(v.Uint(), 4)
}

func (s *Serializer) encodeInt64(v reflect.Value) []byte {
	return asBytesWithSize(toUint(v.Int()), 4)
}

func (s *Serializer) encodeUint(v reflect.Value) []byte {
	return s.encodeUint64(v)
}

func (s *Serializer) encodeInt(v reflect.Value) []byte {
	return s.encodeInt64(v)
}

func (s *Serializer) encodeFloat32(v reflect.Value) []byte {
	return asBytesWithSize(uint64(bits.ReverseBytes32(math.Float32bits(float32(v.Float())))), 3)
}

func (s *Serializer) encodeFloat64(v reflect.Value) []byte {
	return asBytesWithSize(bits.ReverseBytes64(math.Float64bits(v.Float())), 4)
}

func (s *Serializer) encodeComplex64(v reflect.Value) []byte {
	c := v.Complex()
	r := s.encodeFloat32(reflect.ValueOf(float32(real(c))))
	i := s.encodeFloat32(reflect.ValueOf(float32(imag(c))))
	return append(r, i...)
}

func (s *Serializer) encodeComplex128(v reflect.Value) []byte {
	c := v.Complex()
	r := s.encodeFloat64(reflect.ValueOf(real(c)))
	i := s.encodeFloat64(reflect.ValueOf(imag(c)))
	return append(r, i...)
}

func (s *Serializer) encodeUintptr(v reflect.Value) []byte {
	return s.encodeUint64(v)
}

func (s *Serializer) encodeUnsafePointer(v reflect.Value) []byte {
	return asBytesWithSize(uint64(v.Pointer()), 4)
}

func (s *Serializer) encodeChan(v reflect.Value) []byte {
	b := []byte{byte(v.Type().ChanDir())}
	if v.IsNil() {
		b[0] |= meta_nil
	} else {
		b = append(b, s.encodeCount(v.Cap())...)
	}
	return b
}

func (s *Serializer) encodeFunc(v reflect.Value) []byte {
	if v.IsNil() {
		return []byte{meta_nil}
	}
	return []byte{0}
}

func (s *Serializer) encodeList(v reflect.Value) []byte {
	var b []byte
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			return []byte{meta_nil}
		}
		b = []byte{0}
	} else {
		b = []byte{meta_fixed}
	}
	length := v.Len()
	values := []byte{}
	refs := []byte{}
	for i := 0; i < length; i++ {
		value, isRefVal := s.encodeValue(v.Index(i))
		if isRefVal {
			value = value[1:]
			refs = append(refs, s.encodeCount(i)...)
		}
		values = append(values, value...)
	}
	b = append(b, s.encodeCount(length)...)
	b = append(b, s.encodeCount(len(refs))...)
	b = append(b, refs...)
	b = append(b, values...)
	return b
}

func (s *Serializer) encodeMap(v reflect.Value) []byte {
	if v.IsNil() {
		return []byte{meta_nil}
	}
	values := []byte{}
	refs := []byte{}
	iter := v.MapRange()
	i := 0
	for iter.Next() {
		key, keyRefVal := s.encodeValue(iter.Key())
		value, valueRefVal := s.encodeValue(iter.Value())
		f := byte(0)
		if valueRefVal {
			f |= 1
			value = value[1:]
		}
		if keyRefVal {
			f |= 2
			key = key[1:]
		}
		if f != 0 {
			refs = append(refs, f)
			refs = append(refs, s.encodeCount(i)...)
		}
		values = append(values, key...)
		values = append(values, value...)
		i++
	}
	b := append([]byte{0}, s.encodeCount(v.Len())...)
	b = append(b, s.encodeCount(len(refs))...)
	b = append(b, refs...)
	b = append(b, values...)
	return b
}

func (s *Serializer) encodeStruct(v reflect.Value) []byte {
	var fields []byte
	var i, fieldCount int
	switch s.structCodingMode {
	case StructCodingModeIndex:
		for i, fieldCount = 0, v.NumField(); i < fieldCount; i++ {
			fields = append(fields, s.encodeCount(i)...)
			field, _ := s.encode(v.Field(i))
			fields = append(fields, field...)
		}
	case StructCodingModeName:
		t := v.Type()
		for i, fieldCount = 0, v.NumField(); i < fieldCount; i++ {
			fieldName := t.Field(i).Name
			fields = append(fields, s.encodeString(reflect.ValueOf(fieldName))...)
			if fieldName == "_" {
				fields = append(fields, s.encodeCount(i)...)
			}
			field, _ := s.encode(v.Field(i))
			fields = append(fields, field...)
		}
	default:
		for i, fieldCount = 0, v.NumField(); i < fieldCount; i++ {
			field, _ := s.encode(v.Field(i))
			fields = append(fields, field...)
		}
	}
	return append(s.encodeCount(fieldCount), fields...)
}

func (s *Serializer) encodeInterface(v reflect.Value) ([]byte, bool) {
	return s.encode(v.Elem())
}

func (s *Serializer) encodePointer(v reflect.Value) []byte {
	if v.IsNil() {
		return []byte{meta_nil}
	}
	value, isRefVal := s.encodeValue(v.Elem())
	if isRefVal {
		value[0] = meta_prf
		return value
	}
	return append([]byte{0}, value...)
}

func (s *Serializer) encodeReference(cnt uint32) []byte {
	return append([]byte{meta_ref}, asBytesWithSize(uint64(cnt), 3)...)
}

func (s *Serializer) encodeSerializable(v reflect.Value) []byte {
	if isNil(v) {
		return s.encodeNil()
	}
	var b []byte
	if v.Kind() == reflect.Pointer {
		el := v.Elem()
		cnt, ok := s.refs[el]
		if ok {
			return append([]byte{1}, s.encodeReference(cnt)...)
		}
		b = []byte{0}
	}
	body := v.MethodByName("Serialize").Call(nil)[0].Interface().([]byte)
	return append(b, body...)
}

func (s *Serializer) encodeCount(cnt int) []byte {
	return asBytesWithSize(uint64(cnt), 4)
}

/*func (s *Serializer) valueAddress(v reflect.Value) reflect.Value {
	if !v.CanAddr() {
		return ""
	}
	id := s.typeRegistry.typeIdByValue(v)
	b := asMinBytes(uint64(id))
	b = append(b, asMinBytes(uint64(v.UnsafeAddr()))...)
	return *(*string)(unsafe.Pointer(&b))
}*/

func Serialize(value any, options ...any) []byte {
	return NewSerializer().
		WithOptions(options).
		Encode(value)
}
