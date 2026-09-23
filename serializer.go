package codec

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"slices"

	"github.com/URALINNOVATSIYA/reflex"
)

const (
	version    byte = 1           // serializer version
	meta_ref   byte = 0b0000_0000 // pseudo type for referenced values
	meta_fls   byte = 0b0000_0001 // boolean false
	meta_tru   byte = 0b0000_0011 // boolean true
	meta_nil   byte = 0b0001_0000 // determines whether underlying value is nil
	meta_nonil byte = 0b0010_0000 // determines whether underlying value is not nil
	meta_strc  byte = 0b0100_0000 // mark of structs
	meta_tags  byte = 0b0000_0001 // mark of structs that has codec tags
	meta_slice byte = 0b1000_0000 // mark of structs
	meta_sexp  byte = 0b0000_0001 // mark of a slice that created as a slice expression
)

type containerKey struct {
	elemType reflect.Type
	ptr      uintptr
}

type Serializer struct {
	typeRegistry *TypeRegistry
	values       []reflect.Value
	slices       *reflex.SliceMap
	containers   map[containerKey]int
	addresses    map[reflex.Addr]int
	ptrValues    map[reflex.Addr]int
	parents      map[int]int
	childs       map[int][]int
	ptrs         map[int][]int
	refs         map[int]int
	imap         map[int]int
	idx          int
}

func NewSerializer() *Serializer {
	return &Serializer{
		typeRegistry: GetDefaultTypeRegistry(),
		slices:       reflex.NewSliceMap(),
		containers:   make(map[containerKey]int),
		addresses:    make(map[reflex.Addr]int),
		ptrValues:    make(map[reflex.Addr]int),
		parents:      make(map[int]int),
		childs:       make(map[int][]int),
		ptrs:         make(map[int][]int),
		refs:         make(map[int]int),
		imap:         make(map[int]int),
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

func (s *Serializer) clear() {
	s.idx = 0
	s.values = nil
	s.slices.Clear()
	clear(s.containers)
	clear(s.addresses)
	clear(s.ptrValues)
	clear(s.parents)
	clear(s.childs)
	clear(s.ptrs)
	clear(s.refs)
	clear(s.imap)
}

func (s *Serializer) Encode(v any) []byte {
	b := s.encode(reflect.ValueOf(v))
	s.clear()
	return b
}

func (s *Serializer) encode(v reflect.Value) []byte {
	s.traverse(v, -1, -1)
	s.traverseSlices()
	b := []byte{version}
	b = append(b, s.encodeValues()...)
	b = append(b, s.encodePtrs()...)
	return b
}

func (s *Serializer) addNode(parentId int, v reflect.Value) int {
	id := len(s.values)
	s.values = append(s.values, v)
	s.parents[id] = parentId
	s.childs[parentId] = append(s.childs[parentId], id)
	return id
}

func (s *Serializer) addContainer(v reflect.Value, containerId int) bool {
	addr := reflex.Addr{
		Ptr:  reflex.PtrOf(v),
		Type: v.Type(),
	}
	id, exists := s.ptrValues[addr]
	s.ptrValues[addr] = containerId
	if !exists {
		return true
	}
	prevParentId := s.parents[id]
	s.childs[prevParentId] = slices.DeleteFunc(s.childs[prevParentId], func(i int) bool {
		return i == id
	})
	s.parents[id] = containerId
	s.childs[containerId] = []int{id}
	ptrs := s.ptrs[id]
	delete(s.ptrs, id)
	for _, pid := range ptrs {
		p := s.ptrs[pid]
		for i, pid := range p {
			if pid == id {
				p[i] = containerId
			}
		}
		s.ptrs[pid] = p
	}
	s.ptrs[containerId] = ptrs
	return false
}

func (s *Serializer) addReference(v reflect.Value, id int) int {
	addr := reflex.Address(v)
	if !addr.IsValid() {
		return -1
	}
	if ref, exists := s.addresses[addr]; exists {
		s.refs[id] = ref
		return ref
	}
	s.addresses[addr] = id
	return -1
}

func (s *Serializer) addListElem(ptr uintptr, elemType reflect.Type, containerId int) bool {
	key := containerKey{
		elemType: elemType,
		ptr:      ptr,
	}
	ref, exists := s.containers[key]
	if exists {
		s.refs[containerId] = ref
		return false
	}
	s.containers[key] = containerId
	return true
}

func (s *Serializer) addSlice(v reflect.Value, id int) (*reflex.Slice, bool) {
	slice, new := s.slices.Add(v, id)
	if new {
		return slice, true
	}
	s.refs[id] = slice.Id
	return slice, false
}

func (s *Serializer) traverse(v reflect.Value, parentId, parentContainerId int) {
	id := s.addNode(parentId, v)
	switch v.Kind() {
	case reflect.Chan:
		fallthrough
	case reflect.Func:
		fallthrough
	case reflect.String:
		s.addReference(v, id)
	case reflect.Slice:
		if !v.IsNil() {
			s.traverseList(v, id)
		}
	case reflect.Map:
		if s.addReference(v, id) < 0 {
			s.traverseMap(v, id)
		}
	case reflect.Interface:
		s.traverseInterface(v, id, parentContainerId)
	case reflect.Array:
		s.traverseList(v, id)
	case reflect.Struct:
		s.traverseStruct(v, id)
	case reflect.Pointer:
		s.traversePointer(v, id, parentContainerId)
	}
}

func (s *Serializer) traverseList(v reflect.Value, id int) {
	slice, new := s.addSlice(v, id)
	if !new {
		return
	}
	ptr := slice.Ptr
	elemsize := slice.ElemType.Size()
	for i := range v.Len() {
		elem := v.Index(i)
		containerId := s.addNode(id, elem)
		if s.addListElem(ptr, slice.ElemType, containerId) {
			if proceed := s.addContainer(elem, containerId); proceed {
				s.traverse(elem, containerId, containerId)
			}
		}
		ptr += elemsize
	}
}

func (s *Serializer) traverseMap(v reflect.Value, id int) {
	iter := v.MapRange()
	for iter.Next() {
		s.traverse(iter.Key(), id, -1)
		s.traverse(iter.Value(), id, -1)
	}
}

func (s *Serializer) traverseInterface(v reflect.Value, id, parentContainerId int) {
	if parentContainerId < 0 {
		parentContainerId = id
	}
	s.traverse(v.Elem(), id, parentContainerId)
}

func (s *Serializer) traverseStruct(v reflect.Value, id int) {
	tags := s.typeRegistry.tagsByValue(v)
	if len(tags) == 0 {
		for _, field := range v.Fields() {
			containerId := s.addNode(id, field)
			if proceed := s.addContainer(field, containerId); proceed {
				s.traverse(field, containerId, containerId)
			}
		}
		return
	}
	for sf, field := range v.Fields() {
		if tag, exists := tags[sf.Index[0]]; !exists || tag.Deprecated {
			continue
		}
		containerId := s.addNode(id, field)
		if proceed := s.addContainer(field, containerId); proceed {
			s.traverse(field, containerId, containerId)
		}
	}
}

func (s *Serializer) traversePointer(v reflect.Value, id, parentContainerId int) {
	if v.IsNil() {
		return
	}
	elem := v.Elem()
	addr := reflex.Addr{
		Ptr:  reflex.DirPtrOf(v),
		Type: elem.Type(),
	}
	if parentContainerId > 0 {
		id = parentContainerId
	}
	if containerId, exists := s.ptrValues[addr]; exists {
		s.ptrs[containerId] = append(s.ptrs[containerId], id)
		return
	}
	switch elem.Kind() {
	case reflect.String, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
		addr = reflex.Address(elem)
		if ref, exists := s.addresses[addr]; exists {
			s.ptrs[ref] = append(s.ptrs[ref], id)
			return
		}
	}
	nextId := len(s.values)
	s.ptrs[nextId] = append(s.ptrs[nextId], id)
	s.ptrValues[addr] = nextId
	s.traverse(elem, -1, -1)
}

func (s *Serializer) traverseSlices() {
	for _, p := range s.slices.Parents() {
		pchilds := s.childs[p.Id]
		elemSize := p.ElemType.Size()
		for _, child := range p.Childs {
			i := int(child.Ptr-p.Ptr) / int(elemSize)
			for j, containerId := range s.childs[child.Id] {
				delete(s.parents, containerId)
				if _, exists := s.refs[containerId]; exists {
					delete(s.refs, containerId)
					continue
				}
				pid := pchilds[i+j]
				delete(s.refs, pid)
				delete(s.parents, pid)
				pchilds[i+j] = containerId
				s.parents[containerId] = p.Id
			}
			delete(s.childs, child.Id)
		}
	}
}

func (s *Serializer) addMapping(id int) {
	s.imap[id] = s.idx
	s.idx++
}

func (s *Serializer) encodeValues() []byte {
	var b []byte
	for _, id := range s.childs[-1] {
		b = append(b, s.encodeNode(id)...)
	}
	return b
}

func (s *Serializer) encodeNode(id int) []byte {
	b := s.encodeType(s.values[id])
	return append(b, s.encodeValue(id)...)
}

func (s *Serializer) encodeType(v reflect.Value) []byte {
	return u2bs(uint64(s.typeRegistry.typeIdByValue(v)), 3)
}

func (s *Serializer) encodeValue(id int) []byte {
	s.addMapping(id)
	v := s.values[id]
	if isSerializableValue(v) {
		return s.encodeSerializable(v, id)
	}
	switch v.Kind() {
	case reflect.Invalid:
		return s.encodeNil()
	case reflect.Bool:
		return s.encodeBool(v)
	case reflect.String:
		return s.encodeString(v, id)
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
		return s.encodeChan(v, id)
	case reflect.Func:
		return s.encodeFunc(v, id)
	case reflect.Array:
		return s.encodeArray(id)
	case reflect.Slice:
		return s.encodeSlice(v, id)
	case reflect.Map:
		return s.encodeMap(v, id)
	case reflect.Struct:
		return s.encodeStruct(v, id)
	case reflect.Interface:
		return s.encodeInterface(id)
	case reflect.Pointer:
		return s.encodePointer(v)
	}
	panic("unrecognized value kind")
}

func (s *Serializer) encodeSerializable(v reflect.Value, id int) []byte {
	switch v.Kind() {
	case reflect.Interface,
		reflect.Map, reflect.Slice,
		reflect.Pointer, reflect.UnsafePointer,
		reflect.Chan, reflect.Func:
		if v.IsNil() {
			return s.encodeNil()
		}
		if ref := s.addReference(v, id); ref >= 0 {
			return s.encodeReference(ref)
		}
	}
	body := v.MethodByName("Serialize").Call(nil)[0].Interface().([]byte)
	b := append([]byte{meta_nonil}, c2b(len(body))...)
	return append(b, body...)
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

func (s *Serializer) encodeString(v reflect.Value, id int) []byte {
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(ref)
	}
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

func (s *Serializer) encodeChan(v reflect.Value, id int) []byte {
	if v.IsNil() {
		return s.encodeNil()
	}
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(ref)
	}
	return append([]byte{meta_nonil}, c2b(v.Cap())...)
}

func (s *Serializer) encodeFunc(v reflect.Value, id int) []byte {
	if v.IsNil() {
		return s.encodeNil()
	}
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(ref)
	}
	return append([]byte{meta_nonil}, u2bs(uint64(s.typeRegistry.funcIdByValue(v)), 3)...)
}

func (s *Serializer) encodeArray(id int) []byte {
	var b []byte
	for _, containerId := range s.childs[id] {
		s.addMapping(containerId)
		b = append(b, s.encodeValue(s.childs[containerId][0])...)
	}
	return b
}

func (s *Serializer) encodeSlice(v reflect.Value, id int) []byte {
	if v.IsNil() {
		return []byte{meta_slice | meta_nil}
	}
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(ref)
	}
	b := []byte{meta_slice}
	slice := s.slices.Get(id)
	if slice.Parent != nil {
		b[0] |= meta_sexp
		p := slice.Parent
		b = append(b, i2b(p.Id)...)
		i, j, k := p.SliceOf(slice)
		b = append(b, c2b(i)...)
		b = append(b, c2b(j)...)
		b = append(b, c2b(k)...)
		return b
	}
	b = append(b, c2b(slice.Len())...)
	b = append(b, c2b(slice.Cap())...)
	for _, containerId := range s.childs[id] {
		s.addMapping(containerId)
		b = append(b, s.encodeValue(s.childs[containerId][0])...)
	}
	return b
}

func (s *Serializer) encodeMap(v reflect.Value, id int) []byte {
	if v.IsNil() {
		return s.encodeNil()
	}
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(ref)
	}
	b := append([]byte{meta_nonil}, c2b(v.Len())...)
	for _, childId := range s.childs[id] {
		b = append(b, s.encodeValue(childId)...)
	}
	return b
}

func (s *Serializer) encodeStruct(v reflect.Value, id int) []byte {
	b := []byte{meta_strc}
	tags := s.typeRegistry.tagsByValue(v)
	if len(tags) == 0 {
		for _, containerId := range s.childs[id] {
			s.addMapping(containerId)
			b = append(b, s.encodeValue(s.childs[containerId][0])...)
		}
		return b
	}
	b[0] |= meta_tags
	b = append(b, c2b(len(tags))...)
	for i, containerId := range s.childs[id] {
		s.addMapping(containerId)
		b = append(b, c2b(tags[i].Id)...)
		b = append(b, s.encodeValue(s.childs[containerId][0])...)
	}
	return b
}

func (s *Serializer) encodeInterface(id int) []byte {
	return s.encodeNode(s.childs[id][0])
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

func (s *Serializer) encodePtrs() []byte {
	var b []byte
	for ptrValueId, ptrIds := range s.ptrs {
		b = append(b, meta_ref)
		b = append(b, c2b(s.imap[ptrValueId])...)
		for _, ptrId := range ptrIds {
			b = append(b, c2b(s.imap[ptrId])...)
		}
	}
	return b
}

func Serialize(value any, options ...any) []byte {
	return NewSerializer().
		WithOptions(options).
		Encode(value)
}
