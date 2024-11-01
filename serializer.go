package codec

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"unsafe"

	"github.com/URALINNOVATSIYA/reflex"
)

const (
	meta_ref   byte = 0b0000_0000 // pseudo type for referenced values
	meta_fls   byte = 0b0000_0001 // boolean false
	meta_tru   byte = 0b0000_0011 // boolean true
	meta_nil   byte = 0b0001_0000 // determines whether underlying value is nil
	meta_nonil byte = 0b0010_0000 // determines whether underlying value is not nil
	meta_fixed byte = 0b0100_0000 // for lists that are arrays (i.e. have fixed size)
)

type valueAddr struct {
	ptr      unsafe.Pointer
	typeName string
}

type encodedPtr struct {
	from int
	to   int
}

func (a valueAddr) isEmpty() bool {
	return a.ptr == nil
}

type Serializer struct {
	typeRegistry     *TypeRegistry
	structCodingMode byte
	nextId           int
	valueCount       int
	containerAddrs   map[valueAddr]int
	valueAddrs       map[valueAddr]int
	values           map[int]reflect.Value
	nodes            map[int][]int
	nmap             map[int]struct{}
	ptrs             map[int][]encodedPtr
	cycles           map[int]int
	cycleDetector    map[int]int
	ptrChainCounter  int
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
	s.nextId = 0
	s.valueCount = 0
	s.containerAddrs = make(map[valueAddr]int)
	s.valueAddrs = make(map[valueAddr]int)
	s.values = make(map[int]reflect.Value)
	s.nodes = make(map[int][]int)
	s.nmap = make(map[int]struct{})
	s.ptrs = make(map[int][]encodedPtr)
	s.cycles = make(map[int]int)
	s.cycleDetector = make(map[int]int)
	s.ptrChainCounter = 0
	bytes := s.encode(reflect.ValueOf(v))
	return append([]byte{version}, bytes...)
}

func (s *Serializer) encode(v reflect.Value) []byte {
	s.traverse(-1, -1, v)
	b := s.encodeNodes(-1)
	return b
}

func (s *Serializer) id(id int) int {
	if id < 0 {
		id = s.nextId
		s.nextId++
		return id
	}
	return id
}

func (s *Serializer) address(v reflect.Value) valueAddr {
	if ptr := s.ptrOf(v); ptr != nil {
		return valueAddr{
			ptr,
			reflex.NameOf(v.Type()),
		}
	}
	return valueAddr{}
}

func (s *Serializer) ptrOf(v reflect.Value) unsafe.Pointer {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Array:
		return reflex.PtrOf(v)
	case reflect.String, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return reflex.DirPtrOf(v)
	default:
		return nil
	}
}

func (s *Serializer) registerContainer(v reflect.Value, id int) {
	addr := valueAddr{
		reflex.PtrOf(v),
		reflex.NameOf(v.Type()),
	}
	if _, exists := s.containerAddrs[addr]; exists {
		// renumber pointers
	}
	s.containerAddrs[addr] = id
}

func (s *Serializer) registerValue(v reflect.Value, id, parentId int) (int, bool) {
	addr := s.address(v)
	if !addr.isEmpty() {
		if ident, exists := s.valueAddrs[addr]; exists {
			s.bindValues(ident, parentId)
			return ident, false
		}
	}
	id = s.id(id)
	s.values[id] = v
	s.bindValues(id, parentId)
	if !addr.isEmpty() {
		s.valueAddrs[addr] = id
	}
	return id, true
}

func (s *Serializer) bindValues(id, parentId int) {
	s.nodes[parentId] = append(s.nodes[parentId], id)
}

func (s *Serializer) traverse(id, parentId int, v reflect.Value) {
	kind := v.Kind()
	if kind == reflect.Pointer {
		s.traversePointer(v, id, parentId)
		return
	}
	id, isNew := s.registerValue(v, id, parentId)
	if !isNew {
		return
	}
	switch kind {
	case reflect.Slice, reflect.Array:
		s.traverseList(v, id)
	case reflect.Map:
		s.traverseMap(v, id)
	case reflect.Struct:
		s.traverseStruct(v, id)
	case reflect.Interface:
		s.traverseInterface(v, id)
	}
}

func (s *Serializer) traverseList(v reflect.Value, id int) {
	length := v.Len()
	itemId := s.nextId
	s.nextId += length
	for i := 0; i < length; i++ {
		elem := v.Index(i)
		s.registerContainer(elem, itemId)
		s.traverse(-1, id, elem)
		itemId++
	}
}

func (s *Serializer) traverseMap(v reflect.Value, id int) {
	length := v.Len()
	itemId := s.nextId
	s.nextId += length << 1
	iter := v.MapRange()
	for iter.Next() {
		s.traverse(itemId, id, iter.Key())
		itemId++
		s.traverse(itemId, id, iter.Value())
		itemId++
	}
}

func (s *Serializer) traverseStruct(v reflect.Value, id int) {
	fieldId := s.nextId
	fieldCount := v.NumField()
	s.nextId += fieldCount
	for i := 0; i < fieldCount; i++ {
		field := v.Field(i)
		s.registerContainer(field, fieldId)
		s.traverse(-1, id, field)
		fieldId++
	}
}

func (s *Serializer) traverseInterface(v reflect.Value, id int) {
	s.traverse(-1, id, v.Elem())
}

func (s *Serializer) traversePointer(v reflect.Value, id, parentId int) {
	if v.IsNil() {
		id = s.id(id)
		s.values[id] = v
		s.bindValues(id, parentId)
		return
	}

	elem := v.Elem()
	//isPtrToPtr := elem.Kind() == reflect.Pointer

	/*addr := valueAddr{
		reflex.PtrOf(v),
		reflex.NameOf(elem.Type()),
	}*/
	/*nextId, exists := s.containerAddrs[addr]
	if exists {
		s.ptrs[s.ptrChainCounter] = append(s.ptrs[s.ptrChainCounter], encodedPtr{id, nextId})
		s.ptrChainCounter++
		return
	}*/

	/*if isPtrToPtr {
		return
	}*/

	s.ptrChainCounter++
	addr := valueAddr{
		reflex.DirPtrOf(v),
		reflex.NameOf(v.Type()),
	}
	if nextId, exists := s.valueAddrs[addr]; exists {
		s.bindValues(nextId, parentId)
		return
	}

	id = s.id(id)
	s.values[id] = v
	s.valueAddrs[addr] = id
	s.bindValues(id, parentId)
	s.traverse(-1, id, elem)

	/*addr = s.address(elem)
	if nextId, exists = s.valueAddrs[addr]; exists {
		if isPtrToPtr {
			if cnt, exists := s.cycleDetector[nextId]; exists && cnt == s.ptrChainCounter {
				s.cycles[id] = nextId
				s.ptrChainCounter++
			} else {
				s.cycleDetector[nextId] = s.ptrChainCounter
			}
		}
		return
	}

	nextId = s.id(-1)
	s.valueAddrs[addr] = nextId

	if isPtrToPtr {
		s.traverse(nextId, -1, elem)
	} else {
		s.ptrChainCounter++
		s.traverse(nextId, id, elem)
	}*/
}

func (s *Serializer) encodeNodes(parentId int) []byte {
	nodes := s.nodes[parentId]
	size := len(nodes)
	b := c2b(size)
	for i := size - 1; i >= 0; i-- {
		id := nodes[i]
		b = append(b, c2b(id)...)
		b = append(b, s.encodeNode(id)...)
	}
	return b
}

func (s *Serializer) encodeNode(id int) []byte {
	v := s.values[id]
	if b := s.encodeValue(v, id); b != nil {
		return append(s.encodeType(v), b...)
	}
	return s.encodeReference(id)
}

func (s *Serializer) encodeType(v reflect.Value) []byte {
	return u2bs(uint64(s.typeRegistry.typeIdByValue(v)), 3)
}

func (s *Serializer) encodeValue(v reflect.Value, id int) []byte {
	if _, exists := s.nmap[id]; exists {
		return nil
	}
	s.nmap[id] = struct{}{}
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
	case reflect.Slice, reflect.Array:
		//bytes = s.encodeList(v)
	case reflect.Map:
		//bytes = s.encodeMap(v)
	case reflect.Struct:
		return s.encodeStruct(v, id)
	case reflect.Interface:
		return s.encodeInterface(id)
	case reflect.Pointer:
		return s.encodePointer(id)
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

func (s *Serializer) encodeList(v reflect.Value) []byte {
	/*var b []byte
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
	return b*/
	return nil
}

func (s *Serializer) encodeMap(v reflect.Value) []byte {
	/*if v.IsNil() {
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
	b = append(b, values...)*/
	return nil
}

func (s *Serializer) encodeStruct(v reflect.Value, id int) []byte {
	switch s.structCodingMode {
	case StructCodingModeIndex:
		return s.encodeStructIndexMode(v)
	case StructCodingModeName:
		return s.encodeStructNameMode(v)
	default:
		return s.encodeStructDefaultMode(id)
	}
}

func (s *Serializer) encodeStructIndexMode(v reflect.Value) []byte {
	/*var fields []byte
	fieldCount := v.NumField()
	for i := 0; i < fieldCount; i++ {
		fields = append(fields, s.encodeCount(i)...)
		field, _ := s.encode(v.Field(i))
		fields = append(fields, field...)
	}
	return append(s.encodeCount(fieldCount), fields...)*/
	return nil
}

func (s *Serializer) encodeStructNameMode(v reflect.Value) []byte {
	/*var fields []byte
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
	return append(s.encodeCount(fieldCount), fields...)*/
	return nil
}

func (s *Serializer) encodeStructDefaultMode(id int) []byte {
	fields := s.nodes[id]
	size := len(fields)
	b := c2b(size)
	b = append(b, c2b(id+size+1)...)
	for _, fieldId := range fields {
		b = append(b, s.encodeNode(fieldId)...)
	}
	return b
}

func (s *Serializer) encodeInterface(id int) []byte {
	return s.encodeNode(s.nodes[id][0])
}

func (s *Serializer) encodePointer(id int) []byte {
	childs := s.nodes[id]
	if len(childs) == 0 {
		return []byte{meta_nil}
	}
	childId := childs[0]
	b := s.encodeValue(s.values[childId], childId)
	if b == nil {
		b = s.encodeReference(childId)
	}
	return append([]byte{meta_nonil}, b...)
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
