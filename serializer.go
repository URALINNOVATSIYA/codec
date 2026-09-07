package codec

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"

	"github.com/URALINNOVATSIYA/reflex"
)

const (
	meta_ref   byte = 0b0000_0000 // pseudo type for referenced values
	meta_fls   byte = 0b0000_0001 // boolean false
	meta_tru   byte = 0b0000_0011 // boolean true
	meta_nil   byte = 0b0001_0000 // determines whether underlying value is nil
	meta_nonil byte = 0b0010_0000 // determines whether underlying value is not nil
	meta_strc  byte = 0b0100_0000 // mark of structs
)

type Serializer struct {
	typeRegistry   *TypeRegistry
	values         []Value
	containers     map[Addr]int
	addresses      map[Addr]int
	mustEncodeType bool
}

func NewSerializer() *Serializer {
	return &Serializer{
		typeRegistry: GetDefaultTypeRegistry(),
	}
}

func (s *Serializer) WithOptions(options []any) *Serializer {
	for _, option := range options {
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

func (s *Serializer) Encode(v any) []byte {
	s.clear()
	b := append([]byte{version}, s.encode(reflect.ValueOf(v))...)
	s.clear()
	return b
}

func (s *Serializer) clear() {
	s.values = nil
	s.containers = make(map[Addr]int)
	s.addresses = make(map[Addr]int)
}

func (s *Serializer) encode(v reflect.Value) []byte {
	s.traverse(v)
	b := s.encodeValues()
	return b
}

func (s *Serializer) addSimpleValue(v reflect.Value) {
	s.values = append(s.values, Value{Ref: -1, V: v})
}

func (s *Serializer) addReferenceValue(v reflect.Value) bool {
	addr := Address(v)
	if id, exists := s.addresses[addr]; exists {
		s.values = append(s.values, Value{Ref: id, V: s.values[id].V})
		return true
	}
	s.addresses[addr] = len(s.values)
	s.values = append(s.values, Value{Ref: -1, V: v})
	return false
}

func (s *Serializer) addContainer(v reflect.Value) bool {
	addr := Addr{
		reflex.PtrOf(v),
		reflex.NameOf(v.Type()),
	}
	id, exists := s.containers[addr]
	s.containers[addr] = len(s.values)
	s.values = append(s.values, Value{Ref: -2, V: v})
	if !exists {
		return false
	}
	containerId := len(s.values) - 1
	s.values[id].Ref = containerId
	for i, value := range s.values {
		if value.Ref == id {
			s.values[i].Ref = containerId
		}
	}
	return true
}

func (s *Serializer) traverse(v reflect.Value) {
	switch v.Kind() {
	case reflect.Chan:
		fallthrough
	case reflect.Func:
		fallthrough
	case reflect.String:
		if s.addReferenceValue(v) {
			return
		}
	//case reflect.Slice, reflect.Array:
	//	s.traverseList(v, nodeId)
	case reflect.Map:
		s.traverseMap(v)
	case reflect.Struct:
		s.traverseStruct(v)
	case reflect.Interface:
		s.traverseInterface(v)
	case reflect.Pointer:
		s.traversePointer(v)
	default:
		s.addSimpleValue(v)
	}
}

/*func (s *Serializer) traverseList(v reflect.Value) {
	max := v.Len()
	elemId := s.id
	s.id += max
	for i := range max {
		elem := v.Index(i)
		if s.registerContainer(elem, elemId, nodeId) {
			s.traverse(elem, elemId)
		}
		elemId++
	}
}*/

func (s *Serializer) traverseMap(v reflect.Value) {
	if s.addReferenceValue(v) {
		return
	}
	iter := v.MapRange()
	for iter.Next() {
		s.traverse(iter.Key())
		s.traverse(iter.Value())
	}
}

func (s *Serializer) traverseStruct(v reflect.Value) {
	if s.addReferenceValue(v) {
		return
	}
	for _, field := range v.Fields() {
		if !s.addContainer(field) {
			s.traverse(field)
		}
	}
}

func (s *Serializer) traverseInterface(v reflect.Value) {
	s.addSimpleValue(v)
	s.traverse(v.Elem())
}

func (s *Serializer) traversePointer(v reflect.Value) {
	if s.addReferenceValue(v) {
		return
	}
	if v.IsNil() {
		return
	}
	elem := v.Elem()
	addr := Addr{
		Ptr:  reflex.DirPtrOf(v),
		Type: reflex.NameOf(elem.Type()),
	}
	if id, exists := s.containers[addr]; exists {
		s.values = append(s.values, Value{Ref: id})
		return
	}
	s.containers[addr] = len(s.values)
	s.traverse(elem)
}

func (s *Serializer) encodeValues() []byte {
	var b []byte
	s.mustEncodeType = true
	for id, value := range s.values {
		if value.Ref == -2 {
			continue
		}
		if s.mustEncodeType {
			b = append(b, s.encodeType(value.V)...)
			s.mustEncodeType = false
		}
		if value.Ref >= 0 {
			b = append(b, s.encodeReference(value.Ref)...)
			if value.V.Kind() == reflect.Interface && value.Ref > id {
				s.mustEncodeType = true
			}
		} else {
			b = append(b, s.encodeValue(value.V)...)
		}
	}
	return b
}

func (s *Serializer) encodeType(v reflect.Value) []byte {
	return u2bs(uint64(s.typeRegistry.typeIdByValue(v)), 3)
}

func (s *Serializer) encodeValue(v reflect.Value) []byte {
	switch v.Kind() {
	case reflect.Invalid:
		return s.encodeNil()
	case reflect.Bool:
		return s.encodeBool(v)
	case reflect.String:
		return s.encodeString(v)
	case reflect.Uint8:
		return s.encodeUint8(v)
	case reflect.Int8:
		return s.encodeInt8(v)
	case reflect.Uint16:
		return s.encodeUint16(v)
	case reflect.Int16:
		return s.encodeInt16(v)
	case reflect.Uint32:
		return s.encodeUint32(v)
	case reflect.Int32:
		return s.encodeInt32(v)
	case reflect.Uint64:
		return s.encodeUint64(v)
	case reflect.Int64:
		return s.encodeInt(v)
	case reflect.Uint:
		return s.encodeUint(v)
	case reflect.Int:
		return s.encodeInt(v)
	case reflect.Float32:
		return s.encodeFloat32(v)
	case reflect.Float64:
		return s.encodeFloat64(v)
	case reflect.Complex64:
		return s.encodeComplex64(v)
	case reflect.Complex128:
		return s.encodeComplex128(v)
	case reflect.Uintptr:
		return s.encodeUintptr(v)
	case reflect.UnsafePointer:
		return s.encodeUnsafePointer(v)
	case reflect.Chan:
		return s.encodeChan(v)
	case reflect.Func:
		return s.encodeFunc(v)
	case reflect.Array:
		//return s.encodeArray(v)
	case reflect.Slice:
		//return s.encodeSlice(v)
	case reflect.Map:
		return s.encodeMap(v)
	case reflect.Struct:
		return s.encodeStruct(v)
	case reflect.Interface:
		return s.encodeInterface()
	case reflect.Pointer:
		return s.encodePointer(v)
	}
	panic("unrecognized value kind")
}

func (s *Serializer) encodeNil() []byte {
	return []byte{meta_nil}
}

func (s *Serializer) encodeBool(v reflect.Value) []byte {
	if v.Bool() {
		return []byte{meta_tru}
	}
	return []byte{meta_fls}
}

func (s *Serializer) encodeString(v reflect.Value) []byte {
	return append(c2b(v.Len()), v.String()...)
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
	return append([]byte{meta_nonil}, c2b(v.Cap())...)
}

func (s *Serializer) encodeFunc(v reflect.Value) []byte {
	if v.IsNil() {
		return []byte{meta_nil}
	}
	return []byte{meta_nonil}
}

/*func (s *Serializer) encodeArray(nodeId int) []byte {
	b := []byte{meta_aggr}
	for _, cntrId := range s.graph.Children(nodeId) {
		s.graph.Visit(cntrId)
		b = append(b, s.encodeContainer(cntrId)...)
	}
	return b
}*/

func (s *Serializer) encodeSlice(nodeId int) []byte {
	return nil
}

func (s *Serializer) encodeMap(v reflect.Value) []byte {
	if v.IsNil() {
		return []byte{meta_nil}
	}
	return append([]byte{meta_nonil}, c2b(v.Len())...)
}

func (s *Serializer) encodeStruct(v reflect.Value) []byte {
	/*b := []byte{meta_aggr}
	for _, fieldId := range s.graph.Children(nodeId) {
		s.graph.Visit(fieldId)
		b = append(b, s.encodeContainer(fieldId)...)
	}*/
	return []byte{meta_strc}
}

func (s *Serializer) encodeInterface() []byte {
	s.mustEncodeType = true
	return nil
}

func (s *Serializer) encodePointer(v reflect.Value) []byte {
	if v.IsNil() {
		return []byte{meta_nil}
	}
	return []byte{meta_nonil}
}

func (s *Serializer) encodeReference(id int) []byte {
	return append([]byte{meta_ref}, c2b(id)...)
}

func (s *Serializer) encodeSerializable(v reflect.Value) []byte {
	/*if isNil(v) {
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
	return append(b, body...)*/
	return nil
}

func Serialize(value any, options ...any) []byte {
	return NewSerializer().
		WithOptions(options).
		Encode(value)
}
