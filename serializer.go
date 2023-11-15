package codec

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"unsafe"
)

type Serializer struct {
	refs map[string]uint32
	cnt  uint32
}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s *Serializer) Encode(value any) []byte {
	s.refs = make(map[string]uint32, 8)
	s.cnt = 0
	return append([]byte{version}, s.encode(reflect.ValueOf(value))...)
}

func (s *Serializer) encode(v reflect.Value) []byte {
	var bytes []byte
	if bytes = s.storeAddress(v); bytes != nil {
		return bytes
	}
	if isSerializable(v) {
		return s.encodeSerializable(v)
	}
	switch v.Kind() {
	case reflect.Invalid:
		bytes = s.encodeNil(v)
	case reflect.Bool:
		bytes = s.encodeBool(v)
	case reflect.String:
		bytes = s.encodeString(v)
	case reflect.Int, reflect.Uint:
		bytes = s.encodeInt(v)
	case reflect.Int8, reflect.Uint8:
		bytes = s.encodeInt8(v)
	case reflect.Int16, reflect.Uint16:
		bytes = s.encodeInt16(v)
	case reflect.Int32, reflect.Uint32:
		bytes = s.encodeInt32(v)
	case reflect.Int64, reflect.Uint64:
		bytes = s.encodeInt64(v)
	case reflect.Float32, reflect.Float64:
		bytes = s.encodeFloat(v)
	case reflect.Complex64, reflect.Complex128:
		bytes = s.encodeComplex(v)
	case reflect.Uintptr:
		bytes = s.encodeUintptr(v)
	case reflect.UnsafePointer:
		bytes = s.encodeUnsafePointer(v)
	case reflect.Pointer:
		bytes = s.encodePointer(v)
	case reflect.Array:
		bytes = s.encodeArray(v)
	case reflect.Slice:
		bytes = s.encodeList(v)
	case reflect.Map:
		bytes = s.encodeMap(v)
	case reflect.Struct:
		bytes = s.encodeStruct(v)
	case reflect.Chan:
		bytes = s.encodeChan(v)
	case reflect.Func:
		bytes = s.encodeFunc(v)
	case reflect.Interface:
		bytes = s.encodeInterface(v)
	default:
		panic(fmt.Sprintf("unsupported type kind %q", v.Kind()))
	}
	return bytes
}

func (s *Serializer) storeAddress(v reflect.Value) []byte {
	if v.CanAddr() {
		b := asMinBytes(uint64(v.UnsafeAddr()))
		b = append(b, asMinBytes(uint64(tChecker.typeId(v.Type())))...)
		key := *(*string)(unsafe.Pointer(&b))
		cnt, ok := s.refs[key]
		if ok {
			return s.encodeReferenceValue(cnt)
		}
		s.refs[key] = s.cnt
	}
	s.cnt++
	return nil
}

func (s *Serializer) encodeReferenceValue(cnt uint32) []byte {
	return s.encodeUint([]byte{tRef | val}, uint64(cnt))
}

func (s *Serializer) encodeSerializable(v reflect.Value) []byte {
	body := v.MethodByName("Serialize").Call(nil)[0].Interface().([]byte)
	b := s.encodeTypeSignatureWithLength(v, false, len(body))
	b = append(b, body...)
	b[0] |= custom
	return b
}

func (s *Serializer) encodeNil(v reflect.Value) []byte {
	return tChecker.typeSignatureOf(v)
}

func (s *Serializer) encodeBool(v reflect.Value) []byte {
	t := tChecker.typeSignatureOf(v)
	if v.Bool() {
		t[len(t)-1] |= tru
	}
	return t
}

func (s *Serializer) encodeString(v reflect.Value) []byte {
	str := v.String()
	return append(s.encodeTypeSignatureWithLength(v, false, len(str)), str...)
}

func (s *Serializer) encodeInt(v reflect.Value) []byte {
	var i uint64
	t := tChecker.typeSignatureOf(v)
	if v.Kind() == reflect.Int {
		i = toUint(v.Int())
		t[len(t)-1] |= signed
	} else {
		i = v.Uint()
	}
	return s.encodeUint(t, i)
}

func (s *Serializer) encodeUint(typeSignature []byte, i uint64) []byte {
	byteSize, integerBytes := toMinBytes(i)
	b := make([]byte, 0, byteSize+len(typeSignature))
	b = append(b, typeSignature...)
	b = append(b, integerBytes...)
	b[len(typeSignature)-1] |= byte(byteSize - 1)
	return b
}

func (s *Serializer) encodeInt8(v reflect.Value) []byte {
	return s.encodeFixedInt(v, v.Kind() == reflect.Int8, 1, 0)
}

func (s *Serializer) encodeInt16(v reflect.Value) []byte {
	return s.encodeFixedInt(v, v.Kind() == reflect.Int16, 2, 0)
}

func (s *Serializer) encodeInt32(v reflect.Value) []byte {
	return s.encodeFixedInt(v, v.Kind() == reflect.Int32, 4, 2)
}

func (s *Serializer) encodeInt64(v reflect.Value) []byte {
	return s.encodeFixedInt(v, v.Kind() == reflect.Int64, 8, 3)
}

func (s *Serializer) encodeFixedInt(v reflect.Value, neg bool, size int, metaSize int) []byte {
	var i uint64
	t := tChecker.typeSignatureOf(v)
	typeLength := len(t)
	if neg {
		i = toUint(v.Int())
	} else {
		i = v.Uint()
	}
	byteSize := minByteSizeWithMeta(i, metaSize)
	hasMeta := byteSize < size
	if byteSize >= size {
		byteSize = size
	}
	b := make([]byte, 0, byteSize+typeLength)
	b = append(b, t...)
	b = append(b, toBytes(i, byteSize)...)
	if hasMeta {
		b[typeLength-1] |= meta
		b[typeLength] |= byte(byteSize) << (8 - metaSize)
	}
	return b
}

func (s *Serializer) encodeFloat(v reflect.Value) []byte {
	t := tChecker.typeSignatureOf(v)
	var sizeBytes []byte
	var byteSize int
	typeLength := len(t)
	if v.Kind() == reflect.Float64 {
		t[typeLength-1] |= wide
		byteSize, sizeBytes = toMinBytes(bits.ReverseBytes64(math.Float64bits(v.Float())))
	} else {
		byteSize, sizeBytes = toMinBytes(uint64(bits.ReverseBytes32(math.Float32bits(float32(v.Float())))))
	}
	b := make([]byte, 0, byteSize+typeLength)
	b = append(b, t...)
	b = append(b, sizeBytes...)
	b[typeLength-1] |= byte(byteSize - 1)
	return b
}

func (s *Serializer) encodeComplex(v reflect.Value) []byte {
	t := tChecker.typeSignatureOf(v)
	c := v.Complex()
	var r, i []byte
	if v.Kind() == reflect.Complex128 {
		t[len(t)-1] |= wide
		r = s.encodeFloat(reflect.ValueOf(real(c)))
		i = s.encodeFloat(reflect.ValueOf(imag(c)))
	} else {
		r = s.encodeFloat(reflect.ValueOf(float32(real(c))))
		i = s.encodeFloat(reflect.ValueOf(float32(imag(c))))
	}
	return append(append(t, r...), i...)
}

func (s *Serializer) encodeUintptr(v reflect.Value) []byte {
	return s.encodeUint(tChecker.typeSignatureOf(v), v.Uint())
}

func (s *Serializer) encodeUnsafePointer(v reflect.Value) []byte {
	return s.encodeUint(tChecker.typeSignatureOf(v), uint64(v.Pointer()))
}

func (s *Serializer) encodePointer(v reflect.Value) []byte {
	_, b := toMinBytes(uint64(v.Pointer()))
	el := v.Elem()
	if el.IsValid() {
		_, idBytes := toMinBytes(uint64(tChecker.typeId(el.Type())))
		b = append(b, idBytes...)
	}
	cnt, ok := s.refs[*(*string)(unsafe.Pointer(&b))]
	if ok {
		return s.encodeReference(cnt)
	}
	return append(tChecker.typeSignatureOf(v), s.encode(v.Elem())...)
}

func (s *Serializer) encodeReference(cnt uint32) []byte {
	return s.encodeUint([]byte{tRef}, uint64(cnt))
}

func (s *Serializer) encodeArray(v reflect.Value) []byte {
	length := v.Len()
	b := s.encodeTypeSignatureWithLength(v, false, length)
	for i := 0; i < length; i++ {
		b = append(b, s.encode(v.Index(i))...)
	}
	return b
}

func (s *Serializer) encodeList(v reflect.Value) []byte {
	length := v.Len()
	b := s.encodeTypeSignatureWithLength(v, true, length)
	for i := 0; i < length; i++ {
		b = append(b, s.encode(v.Index(i))...)
	}
	return b
}

func (s *Serializer) encodeMap(v reflect.Value) []byte {
	length := v.Len()
	b := s.encodeTypeSignatureWithLength(v, true, length)
	iter := v.MapRange()
	for iter.Next() {
		b = append(b, s.encode(iter.Key())...)
		b = append(b, s.encode(iter.Value())...)
	}
	return b
}

func (s *Serializer) encodeTypeSignatureWithLength(v reflect.Value, canBeNil bool, length int) []byte {
	typeSignature := tChecker.typeSignatureOf(v)
	if canBeNil && v.IsNil() {
		typeSignature[0] |= null
		return typeSignature
	}
	byteSize, sizeBytes := toMinBytes(uint64(length))
	b := make([]byte, 0, length+byteSize+len(typeSignature))
	b = append(b, typeSignature...)
	b = append(b, sizeBytes...)
	b[len(typeSignature)-1] |= byte(byteSize - 1)
	return b
}

func (s *Serializer) encodeStruct(v reflect.Value) []byte {
	var f []byte
	var i, fieldCount int
	switch GetStructCodingMode() {
	case StructCodingModeIndex:
		for i, fieldCount = 0, v.NumField(); i < fieldCount; i++ {
			f = append(f, s.encodeInt(reflect.ValueOf(uint(i)))...)
			f = append(f, s.encode(v.Field(i))...)
		}
	case StructCodingModeName:
		t := v.Type()
		for i, fieldCount = 0, v.NumField(); i < fieldCount; i++ {
			f = append(f, s.encodeString(reflect.ValueOf(t.Field(i).Name))...)
			f = append(f, s.encode(v.Field(i))...)
		}
	default:
		for i, fieldCount = 0, v.NumField(); i < fieldCount; i++ {
			f = append(f, s.encode(v.Field(i))...)
		}
	}
	return append(s.encodeTypeSignatureWithLength(v, false, fieldCount), f...)
}

func (s *Serializer) encodeChan(v reflect.Value) []byte {
	b := s.typeSignatureOf(v, true)
	b[len(b)-1] |= byte(v.Type().ChanDir())
	if !v.IsNil() {
		b = append(b, s.encodeUint([]byte{tInt}, uint64(v.Cap()))...)
	}
	return b
}

func (s *Serializer) encodeFunc(v reflect.Value) []byte {
	return s.typeSignatureOf(v, true)
}

func (s *Serializer) encodeInterface(v reflect.Value) []byte {
	b := s.typeSignatureOf(v, true)
	if v.IsNil() {
		return b
	}
	s.cnt--
	return append(b, s.encode(v.Elem())...)
}

func (s *Serializer) typeSignatureOf(v reflect.Value, canBeNil bool) []byte {
	b := tChecker.typeSignatureOf(v)
	if canBeNil && v.IsNil() {
		b[0] |= null
	}
	return b
}

func Serialize(value any) []byte {
	return NewSerializer().Encode(value)
}
