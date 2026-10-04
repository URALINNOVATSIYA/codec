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
	version    byte = 1           // current serializer version
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
}

func NewSerializer() *Serializer {
	return &Serializer{
		typeRegistry: GetDefaultTypeRegistry(),
		slices:       reflex.NewSliceMap(-2),
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
	s.slices.Clear()
	clear(s.containers)
	clear(s.addresses)
	clear(s.ptrValues)
	clear(s.parents)
	clear(s.childs)
	clear(s.ptrs)
	clear(s.refs)
	clear(s.imap)
	clear(s.values)
	s.values = s.values[:0]
}

func (s *Serializer) Encode(v any) []byte {
	b := s.encode(reflect.ValueOf(v))
	s.clear()
	return b
}

func (s *Serializer) encode(v reflect.Value) []byte {
	s.visit(v, -1, -1)
	s.visitSlices()
	s.renumberGraph(-1)
	b := make([]byte, 1, min(max(1, len(s.values)<<1), 4096))
	b[0] = version
	b = s.encodeValues(b)
	b = s.encodePtrs(b)
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

func (s *Serializer) addSlice(v reflect.Value, id int) (*reflex.Slice, bool) {
	slice, new := s.slices.Add(v, id)
	if new {
		return slice, true
	}
	if id == slice.Id {
		return slice, true
	}
	s.refs[id] = slice.Id
	return slice, false
}

func (s *Serializer) visit(v reflect.Value, parentId, parentContainerId int) {
	id := s.addNode(parentId, v)
	if isSerializableValue(v) && v.Kind() != reflect.Pointer {
		return
	}
	switch v.Kind() {
	case reflect.Chan:
		s.addReference(v, id)
	case reflect.Func:
		s.addReference(v, id)
	case reflect.String:
		s.addSlice(v, id)
	case reflect.Map:
		s.visitMap(v, id)
	case reflect.Interface:
		s.visitInterface(v, id, parentContainerId)
	case reflect.Pointer:
		s.visitPointer(v, id, parentContainerId)
	case reflect.Struct:
		s.visitStruct(v, id)
	case reflect.Array, reflect.Slice:
		s.visitList(v, id)
	}
}

func (s *Serializer) visitList(v reflect.Value, id int) {
	if v.Kind() == reflect.Slice && (v.IsNil() || v.Cap() == 0) {
		return
	}
	slice, new := s.addSlice(v, id)
	if !new {
		return
	}
	ptr := slice.Ptr
	elemsize := slice.ElemType.Size()
	for i := range v.Len() {
		elem := v.Index(i)
		containerId := s.addNode(id, elem)
		if elemsize == 0 {
			s.visit(elem, containerId, containerId)
			continue
		}
		if s.addListElem(ptr, slice.ElemType, containerId) {
			if proceed := s.addContainer(elem, containerId); proceed {
				s.visit(elem, containerId, containerId)
			}
		}
		ptr += elemsize
	}
}

func (s *Serializer) visitMap(v reflect.Value, id int) {
	if s.addReference(v, id) >= 0 {
		return
	}
	iter := v.MapRange()
	for iter.Next() {
		s.visit(iter.Key(), id, -1)
		s.visit(iter.Value(), id, -1)
	}
}

func (s *Serializer) visitInterface(v reflect.Value, id, parentContainerId int) {
	if parentContainerId < 0 {
		parentContainerId = id
	}
	s.visit(v.Elem(), id, parentContainerId)
}

func (s *Serializer) visitStruct(v reflect.Value, id int) {
	tags := s.typeRegistry.tagsByValue(v)
	if len(tags) == 0 {
		for _, field := range v.Fields() {
			containerId := s.addNode(id, field)
			if proceed := s.addContainer(field, containerId); proceed {
				s.visit(field, containerId, containerId)
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
			s.visit(field, containerId, containerId)
		}
	}
}

func (s *Serializer) visitPointer(v reflect.Value, id, parentContainerId int) {
	if v.IsNil() {
		return
	}
	elem := v.Elem()
	addr := reflex.Addr{
		Ptr:  reflex.PtrOf(elem),
		Type: elem.Type(),
	}
	if parentContainerId > 0 {
		id = parentContainerId
	}
	if ref, exists := s.ptrValues[addr]; exists {
		s.ptrs[ref] = append(s.ptrs[ref], id)
		return
	}
	nextId := len(s.values)
	s.ptrs[nextId] = append(s.ptrs[nextId], id)
	s.ptrValues[addr] = nextId
	s.visit(elem, -1, -1)
}

func (s *Serializer) visitSlices() {
	for _, p := range s.slices.Parents() {
		if p.Id < 0 {
			s.visit(p.V, -1, -1)
		}
		if p.V.Kind() == reflect.String {
			continue
		}
		if p.ElemSize() == 0 {
			continue
		}
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

func (s *Serializer) renumberGraph(parentId int) {
	for _, id := range s.childs[parentId] {
		s.imap[id] = len(s.imap)
		s.renumberGraph(id)
	}
}

func (s *Serializer) encodeValues(b []byte) []byte {
	for _, id := range s.childs[-1] {
		b = s.encodeNode(b, id)
	}
	return b
}

func (s *Serializer) encodeNode(b []byte, id int) []byte {
	b = u2bs(b, uint64(s.typeRegistry.typeIdByValue(s.values[id])), 3)
	return s.encodeValue(b, id)
}

func (s *Serializer) encodeValue(b []byte, id int) []byte {
	v := s.values[id]
	if isSerializableValue(v) {
		return s.encodeSerializable(b, v, id)
	}
	switch v.Kind() {
	case reflect.Invalid:
		return s.encodeNil(b)
	case reflect.Bool:
		return s.encodeBool(b, v)
	case reflect.Uint8:
		return s.encodeUint8(b, v)
	case reflect.Int8:
		return s.encodeInt8(b, v)
	case reflect.Uint16:
		return s.encodeUint16(b, v)
	case reflect.Int16:
		return s.encodeInt16(b, v)
	case reflect.Uint32:
		return s.encodeUint32(b, v)
	case reflect.Int32:
		return s.encodeInt32(b, v)
	case reflect.Uint64:
		return s.encodeUint64(b, v)
	case reflect.Int64:
		return s.encodeInt64(b, v)
	case reflect.Uint:
		return s.encodeUint(b, v)
	case reflect.Int:
		return s.encodeInt(b, v)
	case reflect.Float32:
		return s.encodeFloat32(b, v)
	case reflect.Float64:
		return s.encodeFloat64(b, v)
	case reflect.Complex64:
		return s.encodeComplex64(b, v)
	case reflect.Complex128:
		return s.encodeComplex128(b, v)
	case reflect.Uintptr:
		return s.encodeUintptr(b, v)
	case reflect.UnsafePointer:
		return s.encodeUnsafePointer(b, v)
	case reflect.Chan:
		return s.encodeChan(b, v, id)
	case reflect.Func:
		return s.encodeFunc(b, v, id)
	case reflect.String:
		return s.encodeString(b, v, id)
	case reflect.Array:
		return s.encodeArray(b, id)
	case reflect.Slice:
		return s.encodeSlice(b, v, id)
	case reflect.Map:
		return s.encodeMap(b, v, id)
	case reflect.Struct:
		return s.encodeStruct(b, v, id)
	case reflect.Interface:
		return s.encodeInterface(b, id)
	case reflect.Pointer:
		return s.encodePointer(b, v)
	}
	panic("unrecognized value kind")
}

func (s *Serializer) encodeSerializable(b []byte, v reflect.Value, id int) []byte {
	switch v.Kind() {
	case reflect.Interface,
		reflect.Map, reflect.Slice,
		reflect.Pointer, reflect.UnsafePointer,
		reflect.Chan, reflect.Func:
		if v.IsNil() {
			return s.encodeNil(b)
		}
		if ref := s.addReference(v, id); ref >= 0 {
			return s.encodeReference(b, ref)
		}
	}
	body := v.MethodByName("Serialize").Call(nil)[0].Interface().([]byte)
	b = append(b, meta_nonil)
	b = c2b(b, len(body))
	return append(b, body...)
}

func (s *Serializer) encodeNil(b []byte) []byte {
	return append(b, meta_nil)
}

func (s *Serializer) encodeBool(b []byte, v reflect.Value) []byte {
	if v.Bool() {
		return append(b, meta_tru)
	}
	return append(b, meta_fls)
}

func (s *Serializer) encodeUint8(b []byte, v reflect.Value) []byte {
	return append(b, uint8(v.Uint()))
}

func (s *Serializer) encodeInt8(b []byte, v reflect.Value) []byte {
	return append(b, uint8(i2u(v.Int())))
}

func (s *Serializer) encodeUint16(b []byte, v reflect.Value) []byte {
	return u2bs(b, v.Uint(), 2)
}

func (s *Serializer) encodeInt16(b []byte, v reflect.Value) []byte {
	return u2bs(b, i2u(v.Int()), 2)
}

func (s *Serializer) encodeUint32(b []byte, v reflect.Value) []byte {
	return u2bs(b, v.Uint(), 3)
}

func (s *Serializer) encodeInt32(b []byte, v reflect.Value) []byte {
	return u2bs(b, i2u(v.Int()), 3)
}

func (s *Serializer) encodeUint64(b []byte, v reflect.Value) []byte {
	return u2bs(b, v.Uint(), 4)
}

func (s *Serializer) encodeInt64(b []byte, v reflect.Value) []byte {
	return u2bs(b, i2u(v.Int()), 4)
}

func (s *Serializer) encodeUint(b []byte, v reflect.Value) []byte {
	return s.encodeUint64(b, v)
}

func (s *Serializer) encodeInt(b []byte, v reflect.Value) []byte {
	return s.encodeInt64(b, v)
}

func (s *Serializer) encodeFloat32(b []byte, v reflect.Value) []byte {
	return u2bs(b, uint64(bits.ReverseBytes32(math.Float32bits(float32(v.Float())))), 3)
}

func (s *Serializer) encodeFloat64(b []byte, v reflect.Value) []byte {
	return u2bs(b, bits.ReverseBytes64(math.Float64bits(v.Float())), 4)
}

func (s *Serializer) encodeComplex64(b []byte, v reflect.Value) []byte {
	c := v.Complex()
	b = s.encodeFloat32(b, reflect.ValueOf(float32(real(c))))
	return s.encodeFloat32(b, reflect.ValueOf(float32(imag(c))))
}

func (s *Serializer) encodeComplex128(b []byte, v reflect.Value) []byte {
	c := v.Complex()
	b = s.encodeFloat64(b, reflect.ValueOf(real(c)))
	return s.encodeFloat64(b, reflect.ValueOf(imag(c)))
}

func (s *Serializer) encodeUintptr(b []byte, v reflect.Value) []byte {
	return s.encodeUint64(b, v)
}

func (s *Serializer) encodeUnsafePointer(b []byte, v reflect.Value) []byte {
	return u2bs(b, uint64(v.Pointer()), 4)
}

func (s *Serializer) encodeChan(b []byte, v reflect.Value, id int) []byte {
	if v.IsNil() {
		return s.encodeNil(b)
	}
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(b, ref)
	}
	b = append(b, meta_nonil)
	return c2b(b, v.Cap())
}

func (s *Serializer) encodeFunc(b []byte, v reflect.Value, id int) []byte {
	if v.IsNil() {
		return s.encodeNil(b)
	}
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(b, ref)
	}
	b = append(b, meta_nonil)
	return u2bs(b, uint64(s.typeRegistry.funcIdByValue(v)), 3)
}

func (s *Serializer) encodeString(b []byte, v reflect.Value, id int) []byte {
	if ref, exists := s.refs[id]; exists {
		if s.slices.Get(ref).Parent == nil {
			return s.encodeReference(b, ref)
		}
		id = ref
	}
	slice := s.slices.Get(id)
	if slice.Parent == nil {
		return append(i2b(b, v.Len()), v.String()...)
	}
	p := slice.Parent
	b = i2b(b, -1)
	b = i2b(b, s.imap[p.Id])
	i, j, _ := p.SliceOf(slice)
	b = c2b(b, i)
	b = c2b(b, j)
	return b
}

func (s *Serializer) encodeSlice(b []byte, v reflect.Value, id int) []byte {
	if v.IsNil() {
		return append(b, meta_slice|meta_nil)
	}
	if v.Cap() == 0 {
		b = append(b, meta_slice)
		b = c2b(b, v.Len())
		return c2b(b, v.Cap())
	}
	if ref, exists := s.refs[id]; exists {
		if s.slices.Get(ref).Parent == nil {
			return s.encodeReference(b, ref)
		}
		id = ref
	}
	b = append(b, meta_slice)
	slice := s.slices.Get(id)
	if slice.Parent != nil {
		b[len(b)-1] |= meta_sexp
		p := slice.Parent
		b = i2b(b, s.imap[p.Id])
		i, j, k := p.SliceOf(slice)
		b = c2b(b, i)
		b = c2b(b, j)
		b = c2b(b, k)
		return b
	}
	b = c2b(b, v.Len())
	b = c2b(b, v.Cap())
	for _, containerId := range s.childs[id] {
		b = s.encodeValue(b, s.childs[containerId][0])
	}
	return b
}

func (s *Serializer) encodeMap(b []byte, v reflect.Value, id int) []byte {
	if v.IsNil() {
		return s.encodeNil(b)
	}
	if ref, exists := s.refs[id]; exists {
		return s.encodeReference(b, ref)
	}
	b = append(b, meta_nonil)
	b = c2b(b, v.Len())
	for _, childId := range s.childs[id] {
		b = s.encodeValue(b, childId)
	}
	return b
}

func (s *Serializer) encodeArray(b []byte, id int) []byte {
	for _, containerId := range s.childs[id] {
		b = s.encodeValue(b, s.childs[containerId][0])
	}
	return b
}

func (s *Serializer) encodeStruct(b []byte, v reflect.Value, id int) []byte {
	b = append(b, meta_strc)
	tags := s.typeRegistry.tagsByValue(v)
	if len(tags) == 0 {
		for _, containerId := range s.childs[id] {
			b = s.encodeValue(b, s.childs[containerId][0])
		}
		return b
	}
	b[len(b)-1] |= meta_tags
	b = c2b(b, len(s.childs[id]))
	childIndex := 0
	for fieldIndex := range v.NumField() {
		tag, exists := tags[fieldIndex]
		if !exists || tag.Deprecated {
			continue
		}
		containerId := s.childs[id][childIndex]
		b = c2b(b, tag.Id)
		b = s.encodeValue(b, s.childs[containerId][0])
		childIndex++
	}
	return b
}

func (s *Serializer) encodeInterface(b []byte, id int) []byte {
	return s.encodeNode(b, s.childs[id][0])
}

func (s *Serializer) encodePointer(b []byte, v reflect.Value) []byte {
	if v.IsNil() {
		return s.encodeNil(b)
	}
	return append(b, meta_nonil)
}

func (s *Serializer) encodeReference(b []byte, id int) []byte {
	b = append(b, meta_ref)
	return c2b(b, s.imap[id])
}

func (s *Serializer) encodePtrs(b []byte) []byte {
	for ptrValueId, ptrIds := range s.ptrs {
		b = append(b, meta_ref)
		b = c2b(b, s.imap[ptrValueId])
		for _, ptrId := range ptrIds {
			b = c2b(b, s.imap[ptrId])
		}
	}
	return b
}

func Serialize(value any, options ...any) []byte {
	return NewSerializer().
		WithOptions(options).
		Encode(value)
}
