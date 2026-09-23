package codec

import (
	"fmt"
	"io"
	"math"
	"math/bits"
	"reflect"
	"unsafe"

	"github.com/URALINNOVATSIYA/reflex"
)

type sliceExp struct {
	id int
	i  int
	j  int
	k  int
}

type Unserializer struct {
	typeRegistry *TypeRegistry
	id           int
	pos          int
	size         int
	data         []byte
	values       []reflect.Value
	slices       map[int][]sliceExp
	maps         map[int][]int
}

func NewUnserializer() *Unserializer {
	return &Unserializer{
		typeRegistry: GetDefaultTypeRegistry(),
		slices:       make(map[int][]sliceExp),
		maps:         make(map[int][]int),
	}
}

func (u *Unserializer) WithOptions(options []any) *Unserializer {
	for _, option := range options {
		if v, ok := option.(*TypeRegistry); ok {
			u.WithTypeRegistry(v)
			continue
		}
		panic(fmt.Errorf("invalid option type %T", option))
	}
	return u
}

func (u *Unserializer) WithTypeRegistry(registry *TypeRegistry) *Unserializer {
	u.typeRegistry = registry
	return u
}

func (u *Unserializer) clear() {
	u.id = 0
	u.pos = 1 // skip version for now
	u.values = nil
	clear(u.slices)
	clear(u.maps)
}

func (u *Unserializer) Decode(data []byte) (value any, err error) {
	if data == nil {
		return nil, io.ErrUnexpectedEOF
	}
	//defer func() {
	//	if e := recover(); e != nil {
	//		err = fmt.Errorf("%s", e)
	//	}
	//}()
	u.clear()
	u.data = data
	u.size = len(data)
	if v := u.decode(); v.IsValid() {
		return v.Interface(), nil
	}
	return value, err
}

func (u *Unserializer) decode() reflect.Value {
	for u.topIsNotRef() {
		u.decodeNode()
	}
	u.restoreSlices()
	u.restorePointers()
	u.restoreMaps()
	return u.values[0]
}

func (u *Unserializer) decodeId() int {
	return int(u.decodeCount(4))
}

func (u *Unserializer) decodeLength() int {
	return int(u.decodeCount(4))
}

func (u *Unserializer) decodeType() reflect.Type {
	return u.typeRegistry.typeById(int(u.decodeCount(3)))
}

func (u *Unserializer) decodeNode() reflect.Value {
	t := u.decodeType()
	return u.decodeValue(t, reflex.Zero(t))
}

func (u *Unserializer) decodeValue(t reflect.Type, v reflect.Value) reflect.Value {
	if isSerializableType(t) {
		if u.topIsRef() {
			return u.decodeReference()
		}
		u.values = append(u.values, v)
		u.decodeSerializable(t, v)
		return v
	}
	kind := v.Kind()
	switch kind {
	case reflect.Invalid:
		u.decodeNil()
	case reflect.Bool:
		u.decodeBool(v)
	case reflect.Uint8:
		u.decodeUint8(v)
	case reflect.Int8:
		u.decodeInt8(v)
	case reflect.Uint16:
		u.decodeUint16(v)
	case reflect.Int16:
		u.decodeInt16(v)
	case reflect.Uint32:
		u.decodeUint32(v)
	case reflect.Int32:
		u.decodeInt32(v)
	case reflect.Uint64:
		u.decodeUint64(v)
	case reflect.Int64:
		u.decodeInt64(v)
	case reflect.Uint:
		u.decodeUint(v)
	case reflect.Int:
		u.decodeInt(v)
	case reflect.Float32:
		u.decodeFloat32(v)
	case reflect.Float64:
		u.decodeFloat64(v)
	case reflect.Complex64:
		u.decodeComplex64(v)
	case reflect.Complex128:
		u.decodeComplex128(v)
	case reflect.Uintptr:
		u.decodeUintptr(v)
	case reflect.UnsafePointer:
		u.decodeUnsafePointer(v)
	default:
		if u.topIsRef() {
			return u.decodeReference()
		}
		u.values = append(u.values, v)
		switch kind {
		case reflect.String:
			u.decodeString(v)
		case reflect.Chan:
			u.decodeChan(v)
		case reflect.Func:
			u.decodeFunc(v)
		case reflect.Array:
			u.decodeArray(t, v)
		case reflect.Slice:
			u.decodeSlice(t, v)
		case reflect.Map:
			u.decodeMap(t, v)
		case reflect.Struct:
			u.decodeStruct(v)
		case reflect.Interface:
			u.decodeInterface(v)
		case reflect.Pointer:
			u.decodePointer(t, v)
		}
		return v
	}
	u.values = append(u.values, v)
	return v
}

func (u *Unserializer) decodeSerializable(t reflect.Type, v reflect.Value) {
	if u.readByte() == meta_nil {
		return
	}
	res := v.MethodByName("Unserialize").Call([]reflect.Value{reflect.ValueOf(u.readBytes(u.decodeLength()))})
	err := res[1]
	if !err.IsNil() {
		panic(err.Interface())
	}
	v.Set(res[0].Elem().Convert(t))
}

func (u *Unserializer) decodeNil() {
	u.readByte()
}

func (u *Unserializer) decodeBool(v reflect.Value) {
	v.SetBool(u.readByte() == meta_tru)
}

func (u *Unserializer) decodeString(v reflect.Value) {
	v.SetString(string(u.readBytes(u.decodeLength())))
}

func (u *Unserializer) decodeUint8(v reflect.Value) {
	v.SetUint(uint64(u.readByte()))
}

func (u *Unserializer) decodeInt8(v reflect.Value) {
	v.SetInt(u2i(uint64(u.readByte())))
}

func (u *Unserializer) decodeUint16(v reflect.Value) {
	v.SetUint(u.decodeCount(2))
}

func (u *Unserializer) decodeInt16(v reflect.Value) {
	v.SetInt(u2i(u.decodeCount(2)))
}

func (u *Unserializer) decodeUint32(v reflect.Value) {
	v.SetUint(u.decodeCount(3))
}

func (u *Unserializer) decodeInt32(v reflect.Value) {
	v.SetInt(u2i(u.decodeCount(3)))
}

func (u *Unserializer) decodeUint64(v reflect.Value) {
	v.SetUint(u.decodeCount(4))
}

func (u *Unserializer) decodeInt64(v reflect.Value) {
	v.SetInt(u2i(u.decodeCount(4)))
}

func (u *Unserializer) decodeUint(v reflect.Value) {
	u.decodeUint64(v)
}

func (u *Unserializer) decodeInt(v reflect.Value) {
	u.decodeInt64(v)
}

func (u *Unserializer) decodeFloat32(v reflect.Value) {
	v.SetFloat(float64(u.readFloat32()))
}

func (u *Unserializer) readFloat32() float32 {
	return math.Float32frombits(bits.ReverseBytes32(uint32(u.decodeCount(3))))
}

func (u *Unserializer) decodeFloat64(v reflect.Value) {
	v.SetFloat(u.readFloat64())
}

func (u *Unserializer) readFloat64() float64 {
	return math.Float64frombits(bits.ReverseBytes64(u.decodeCount(4)))
}

func (u *Unserializer) decodeComplex64(v reflect.Value) {
	r := u.readFloat32()
	i := u.readFloat32()
	v.SetComplex(complex128(complex(r, i)))
}

func (u *Unserializer) decodeComplex128(v reflect.Value) {
	r := u.readFloat64()
	i := u.readFloat64()
	v.SetComplex(complex(r, i))
}

func (u *Unserializer) decodeUintptr(v reflect.Value) {
	u.decodeUint64(v)
}

func (u *Unserializer) decodeUnsafePointer(v reflect.Value) {
	v.SetPointer(unsafe.Pointer(uintptr(u.decodeCount(4))))
}

func (u *Unserializer) decodeChan(v reflect.Value) {
	if u.readByte() == meta_nil {
		return
	}
	cap := u.decodeLength()
	t := v.Type()
	if t.ChanDir() == reflect.BothDir {
		v.Set(reflect.MakeChan(t, cap))
	} else {
		el := t.Elem()
		biDirChan := reflect.ChanOf(reflect.BothDir, el)
		unDirChan := reflect.ChanOf(t.ChanDir(), el)
		ch := reflect.MakeChan(biDirChan, cap)
		v.Set(ch.Convert(unDirChan).Convert(t))
	}
}

func (u *Unserializer) decodeFunc(v reflect.Value) {
	if u.readByte() == meta_nonil {
		f := u.typeRegistry.funcById(int(u.decodeCount(3)))
		v.Set(reflex.MakeExported(f))
	}
}

func (u *Unserializer) decodeMap(t reflect.Type, v reflect.Value) {
	if u.readByte() == meta_nil {
		return
	}
	size := u.decodeLength()
	v.Set(reflect.MakeMapWithSize(t, size))
	keyType := t.Key()
	keyKind := keyType.Kind()
	valueType := t.Elem()
	valueKind := valueType.Kind()
	id := len(u.values) - 1
	for range size {
		keyId := len(u.values)
		key := u.decodeValue(keyType, reflex.Zero(keyType))
		valueId := len(u.values)
		value := u.decodeValue(valueType, reflex.Zero(valueType))
		if keyKind != reflect.Pointer && keyKind != reflect.Interface &&
			valueKind != reflect.Pointer && valueKind != reflect.Interface {
			v.SetMapIndex(key, value)
		} else {
			u.maps[id] = append(u.maps[id], keyId, valueId)
		}
	}
}

func (u *Unserializer) decodeSlice(t reflect.Type, v reflect.Value) {
	top := u.readByte()
	if top&meta_nil != 0 {
		return
	}
	if top&meta_sexp != 0 {
		parentId := int(u2i(u.decodeCount(4)))
		i := u.decodeLength()
		j := u.decodeLength()
		k := u.decodeLength()
		id := len(u.values) - 1
		if u.values[id-1].Kind() == reflect.Interface {
			id--
		}
		u.slices[parentId] = append(u.slices[parentId], sliceExp{
			id: id,
			i:  i,
			j:  j,
			k:  k,
		})
		return
	}
	length := u.decodeLength()
	capacity := u.decodeLength()
	v.Set(reflect.MakeSlice(t, length, capacity))
	u.decodeArray(t, v)
}

func (u *Unserializer) decodeArray(t reflect.Type, v reflect.Value) {
	elemType := t.Elem()
	for i := range t.Len() {
		elem := v.Index(i)
		u.decodeContainer(elemType, elem)
	}
}

func (u *Unserializer) decodeStruct(v reflect.Value) {
	if u.readByte()&meta_tags == 0 {
		for i, count := 0, v.NumField(); i < count; i++ {
			field := v.Field(i)
			u.decodeContainer(field.Type(), field)
		}
		return
	}
	fieldCount := v.NumField()
	tags := u.typeRegistry.tagsByValue(v)
	for range u.decodeLength() {
		i := tags.IndexById(u.decodeId())
		if i < 0 || i >= fieldCount {
			continue
		}
		field := v.Field(i)
		u.decodeContainer(field.Type(), field)
	}
}

func (u *Unserializer) decodeInterface(v reflect.Value) {
	elem := u.decodeNode()
	if elem.IsValid() {
		v.Set(elem)
	}
}

func (u *Unserializer) decodePointer(t reflect.Type, v reflect.Value) {
	if u.readByte() == meta_nil {
		return
	}
	elemType := t.Elem()
	elemValue := reflex.Zero(elemType)
	v.Set(reflex.PtrAt(elemType, elemValue))
}

func (u *Unserializer) decodeContainer(containerType reflect.Type, containerValue reflect.Value) reflect.Value {
	containerValue = reflex.PtrAt(containerType, containerValue).Elem()
	u.values = append(u.values, containerValue)
	v := u.decodeValue(containerType, containerValue)
	if v.IsValid() {
		containerValue.Set(v)
	}
	return containerValue
}

func (u *Unserializer) decodeReference() reflect.Value {
	_ = u.readByte()
	return u.values[u.decodeId()]
}

func (u *Unserializer) decodeCount(sizeBits int) uint64 {
	cnt, length := bs2u(u.data[u.pos:], sizeBits)
	if length <= 0 {
		panic(io.ErrUnexpectedEOF)
	}
	u.pos += length
	return cnt
}

func (u *Unserializer) restoreSlices() {
	for parentId, exps := range u.slices {
		p := u.values[parentId]
		for _, exp := range exps {
			s := p.Slice3(exp.i, exp.j, exp.k)
			u.values[exp.id].Set(s)
		}
	}
}

func (u *Unserializer) restorePointers() {
	for u.pos < u.size {
		_ = u.readByte()
		v := u.values[u.decodeId()]
		for u.topIsNotRef() {
			ptr := u.values[u.decodeId()]
			ptr.Set(reflex.PtrAt(v.Type(), v))
		}
	}
}

func (u *Unserializer) restoreMaps() {
	for mapId, items := range u.maps {
		m := u.values[mapId]
		for i, max := 0, len(items); i < max; i += 2 {
			m.SetMapIndex(u.values[items[i]], u.values[items[i+1]])
		}
	}
}

func (u *Unserializer) top() byte {
	return u.data[u.pos]
}

func (u *Unserializer) topIsContainerRef() bool {
	return u.pos+1 < u.size && u.data[u.pos] == meta_ref && u.data[u.pos+1] == meta_ref
}

func (u *Unserializer) topIsRef() bool {
	return u.pos < u.size && u.data[u.pos] == meta_ref
}

func (u *Unserializer) topIsNotRef() bool {
	return u.pos < u.size && u.data[u.pos] != meta_ref
}

func (u *Unserializer) readByte() byte {
	u.pos++
	if u.pos > u.size {
		panic(io.ErrUnexpectedEOF)
	}
	return u.data[u.pos-1]
}

func (u *Unserializer) readBytes(count int) []byte {
	u.pos += count
	if u.pos > u.size {
		panic(io.ErrUnexpectedEOF)
	}
	return u.data[u.pos-count : u.pos]
}

func Unserialize(data []byte, options ...any) (any, error) {
	return NewUnserializer().
		WithOptions(options).
		Decode(data)
}
