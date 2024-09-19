package codec

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"
)

const (
	meta_ref   byte = 0b0000_0000 // pseudo type for referenced values
	meta_nil   byte = 0b0001_0000 // determines whether underlying value is nil
	meta_fixed byte = 0b0010_0000 // for lists that are arrays (i.e. have fixed size)
	meta_prf   byte = 0b0100_0000 //
)

type valueAddr struct {
	tp   string
	addr uintptr
}

func (a valueAddr) isEmpty() bool {
	return a.tp == "" && a.addr == 0
}

type Serializer struct {
	typeRegistry     *TypeRegistry
	structCodingMode byte
	cnt              int
	addr             map[valueAddr]int       
	vals             map[int]reflect.Value
	parents          map[int]map[int]struct{}
	childs           map[int]map[int]struct{}
	ptrs             map[int]int
	cycles           map[int]int
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

func (s *Serializer) Encode(v any) []byte {
	s.cnt = 0
	s.addr = make(map[valueAddr]int)
	s.vals = make(map[int]reflect.Value)
	s.parents = make(map[int]map[int]struct{})
	s.childs = make(map[int]map[int]struct{})
	bytes := s.encode(reflect.ValueOf(v))
	return append([]byte{version}, bytes...)
}

func (s *Serializer) encode(v reflect.Value) ([]byte) {
	s.traverse(-1, -1, v)

	return nil

	/*value, isReferencedValue := s.encodeValue(v)
	if isReferencedValue {
		return value, true
	}
	return append(s.encodeType(v), value...), false*/
}

func (s *Serializer) traverse(id, parentId int, v reflect.Value) {
	addr := s.address(v)
	if ident, exists := s.addr[addr]; exists {
		s.bindValues(ident, parentId)
		return
	}
	id = s.id(id)
	s.addr[addr] = id
	s.registerValue(v, addr.tp, id)
	s.bindValues(id, parentId)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		s.traverseList(v, id)
	case reflect.Map:
		//s.traverseMap(v)
	case reflect.Struct:
		//s.traverseStruct(v)
	case reflect.Interface:
		//s.traverseInterface(v)
	case reflect.Pointer:
		//s.traversePointer(v)
	}
	return
}

func (s *Serializer) id(id int) int {
	if id < 0 {
		id = s.cnt
		s.cnt++
		return id
	}
	return id
}

func (s *Serializer) address(v reflect.Value) valueAddr {
	if !v.IsValid() {
		return valueAddr{}
	}
	var addr uintptr
	switch v.Kind() {
	case reflect.Func:
		return valueAddr{
			tp: s.typeRegistry.funcName(v),
		}
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Chan:
		if v.IsNil() {
			return valueAddr{}
		}
		addr = v.Pointer()
	default:
		if !v.CanAddr() {
			return valueAddr{}
		}
		addr = v.Addr().Pointer()
	}
	return valueAddr{
		tp: s.typeRegistry.typeName(v.Type()),
		addr: addr,
	}
}

func (s *Serializer) registerValue(v reflect.Value, tp string, id int) {
	s.vals[id] = v
}

func (s *Serializer) bindValues(id, parentId int) {
	if childs := s.childs[parentId]; childs != nil {
		childs[id] = struct{}{}
	} else {
		s.childs[parentId] = map[int]struct{}{id: struct{}{}}
	}
	if parents := s.parents[id]; parents != nil {
		parents[parentId] = struct{}{}
	} else {
		s.parents[id] = map[int]struct{}{parentId: struct{}{}}
	}
	if len(s.parents[id]) > 1 {
		s.childs[-1][id] = struct{}{}
	}
}

func (s *Serializer) traverseList(v reflect.Value, id int) {
	length := v.Len()
	itemId := s.cnt
	s.cnt += length
	for i := 0; i < length; i++ {
		s.traverse(itemId, id, v.Index(i))
		itemId++
	}
}

func (s *Serializer) encodeType(v reflect.Value) []byte {
	id := s.typeRegistry.typeIdByValue(v)
	return u2bs(uint64(id), 3)
}

func (s *Serializer) encodeValue(v reflect.Value) (bytes []byte, isReferencedValue bool) {
	sign := s.valueSignature(v)
	if sign != "" {
		if cnt, ok := s.refs[sign]; ok {
			fmt.Printf("ref to %d\n", cnt)
			return s.encodeReference(cnt), true
		}
		s.refs[sign] = s.cnt
	}
	if v.IsValid() {
		fmt.Printf("cnt=%d: %s\n", s.cnt, s.typeRegistry.typeName(v.Type()))
	} else {
		fmt.Printf("cnt=%d: nil\n", s.cnt)
	}
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
	return []byte{uint8(i2u(v.Int()))}
}

func (s *Serializer) encodeUint16(v reflect.Value) []byte {
	return u2bs(v.Uint(), 2)
}

func (s *Serializer) encodeInt16(v reflect.Value) []byte {
	return u2bs(i2u(v.Int()), 2)
}

func (s *Serializer) encodeUint32(v reflect.Value) []byte {
	return u2bs(v.Uint(), 3)
}

func (s *Serializer) encodeInt32(v reflect.Value) []byte {
	return u2bs(i2u(v.Int()), 3)
}

func (s *Serializer) encodeUint64(v reflect.Value) []byte {
	return u2bs(v.Uint(), 4)
}

func (s *Serializer) encodeInt64(v reflect.Value) []byte {
	return u2bs(i2u(v.Int()), 4)
}

func (s *Serializer) encodeUint(v reflect.Value) []byte {
	return s.encodeUint64(v)
}

func (s *Serializer) encodeInt(v reflect.Value) []byte {
	return s.encodeInt64(v)
}

func (s *Serializer) encodeFloat32(v reflect.Value) []byte {
	return u2bs(uint64(bits.ReverseBytes32(math.Float32bits(float32(v.Float())))), 3)
}

func (s *Serializer) encodeFloat64(v reflect.Value) []byte {
	return u2bs(bits.ReverseBytes64(math.Float64bits(v.Float())), 4)
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
	return u2bs(uint64(v.Pointer()), 4)
}

func (s *Serializer) encodeChan(v reflect.Value) []byte {
	if v.IsNil() {
		return []byte{meta_nil}
	}
	return append([]byte{0}, s.encodeCount(v.Cap())...)
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
			f |= 0b01
			value = value[1:]
		}
		if keyRefVal {
			f |= 0b10
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
	switch s.structCodingMode {
	case StructCodingModeIndex:
		return s.encodeStructIndexMode(v)
	case StructCodingModeName:
		return s.encodeStructNameMode(v)
	default:
		return s.encodeStructDefaultMode(v)
	}
}

func (s *Serializer) encodeStructIndexMode(v reflect.Value) []byte {
	var fields []byte
	fieldCount := v.NumField()
	for i := 0; i < fieldCount; i++ {
		fields = append(fields, s.encodeCount(i)...)
		field, _ := s.encode(v.Field(i))
		fields = append(fields, field...)
	}
	return append(s.encodeCount(fieldCount), fields...)
}

func (s *Serializer) encodeStructNameMode(v reflect.Value) []byte {
	var fields []byte
	fieldCount := v.NumField()
	for i, t := 0, v.Type(); i < fieldCount; i++ {
		fieldName := t.Field(i).Name
		fields = append(fields, s.encodeString(reflect.ValueOf(fieldName))...)
		if fieldName == "_" {
			fields = append(fields, s.encodeCount(i)...)
		}
		field, _ := s.encode(v.Field(i))
		fields = append(fields, field...)
	}
	return append(s.encodeCount(fieldCount), fields...)
}

func (s *Serializer) encodeStructDefaultMode(v reflect.Value) []byte {
	var fields []byte
	fieldCount := v.NumField()
	for i := 0; i < fieldCount; i++ {
		field, _ := s.encode(v.Field(i))
		fields = append(fields, field...)
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
	return append([]byte{meta_ref}, u2bs(uint64(cnt), 3)...)
}

func (s *Serializer) encodeSerializable(v reflect.Value) []byte {
	if isNil(v) {
		return s.encodeNil()
	}
	var b []byte
	if v.Kind() == reflect.Pointer {
		cnt, ok := s.refs[s.valueSignature(v.Elem())]
		if ok {
			return append([]byte{1}, s.encodeReference(cnt)...)
		}
		b = []byte{0}
	}
	body := v.MethodByName("Serialize").Call(nil)[0].Interface().([]byte)
	return append(b, body...)
}

func (s *Serializer) encodeCount(cnt int) []byte {
	return u2bs(uint64(cnt), 4)
}

func Serialize(value any, options ...any) []byte {
	return NewSerializer().
		WithOptions(options).
		Encode(value)
}
