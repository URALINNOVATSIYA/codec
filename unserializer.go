package codec

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/bits"
	"reflect"
	"unsafe"
)

type Unserializer struct {
	refs map[uint32]reflect.Value
	rels map[uint32]uint32
	cnt  uint32
	data []byte
	pos  int
	size int
}

func NewUnserializer() *Unserializer {
	return &Unserializer{}
}

func (u *Unserializer) Decode(data []byte) (value any, err error) {
	if data == nil {
		return nil, io.ErrUnexpectedEOF
	}
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(e.(string))
		}
	}()
	u.refs = make(map[uint32]reflect.Value)
	u.rels = make(map[uint32]uint32)
	u.cnt = 0
	u.data = data
	u.pos = 1 // skip version for now
	u.size = len(data)
	var v reflect.Value
	v, err = u.decode(v)
	if err == nil && v.IsValid() {
		value = v.Interface()
	}
	return value, err
}

func (u *Unserializer) decode(value reflect.Value) (v reflect.Value, err error) {
	t := u.data[u.pos]
	if t&mask == tType && t&custom == 0 {
		t = u.data[u.pos+int(t&0b111)+2]
	}
	cnt := u.cnt
	proceed := true
	switch t & mask {
	case tType:
		v, err = u.decodeSerializable()
	case tNil:
		v = u.decodeNil()
	case tBool:
		v, err = u.decodeBool()
	case tInt8:
		v, err = u.decodeFixedInt()
	case tInt:
		v, err = u.decodeInt()
	case tFloat:
		v, err = u.decodeFloat()
	case tComplex:
		v, err = u.decodeComplex()
	case tString:
		v, err = u.decodeString()
	case tUintptr:
		v, err = u.decodeUintptr()
	case tChan:
		if t&tFunc == tFunc {
			v, err = u.decodeFunc()
		} else {
			v, err = u.decodeChan()
		}
	case tList:
		v, err = u.decodeList()
		proceed = false
	case tMap:
		v, err = u.decodeMap()
		proceed = false
	case tStruct:
		v, err = u.decodeStruct()
		proceed = false
	case tPointer:
		v, err = u.decodePointer()
		proceed = false
	case tRef:
		v, err = u.decodeReference(value)
		proceed = false
	case tInterface:
		v, err = u.decodeInterface()
		proceed = false
	default:
		return value, fmt.Errorf("unrecognized type %b", t)
	}
	if err != nil {
		return v, err
	}
	if value.IsValid() {
		value.Set(v)
	} else {
		value = v
	}
	if proceed {
		u.refs[u.cnt] = value
		u.cnt++
	} else {
		u.refs[cnt] = value
	}
	return value, nil
}

func (u *Unserializer) decodeSerializable() (reflect.Value, error) {
	v, l, _, err := u.decodeTypeWithLength()
	if err != nil {
		return v, err
	}
	length := int(l)
	u.pos += length
	if u.pos > u.size {
		return v, io.ErrUnexpectedEOF
	}
	if !isSerializable(v) {
		return v, fmt.Errorf("struct %q does not implement Serializable interface", v.Type().Name())
	}
	s := v.MethodByName("Unserialize").Call([]reflect.Value{reflect.ValueOf(u.data[u.pos-length : u.pos])})
	e := s[1].Interface()
	if err != nil {
		return v, e.(error)
	}
	return reflect.ValueOf(s[0].Interface()).Convert(v.Type()), nil
}

func (u *Unserializer) decodeNil() reflect.Value {
	return reflect.Value{}
}

func (u *Unserializer) decodeBool() (reflect.Value, error) {
	v, err := u.decodeType()
	isTrue := (u.data[u.pos-1] & tru) != 0
	if err != nil {
		return v, err
	}
	if isTrue {
		changeValue(v, true)
	} else {
		changeValue(v, false)
	}
	return v, nil
}

func (u *Unserializer) decodeString() (reflect.Value, error) {
	v, length, _, err := u.decodeTypeWithLength()
	if err != nil {
		return v, err
	}
	if length > 0 {
		start := u.pos
		u.pos += int(length)
		if u.pos > u.size {
			return v, io.ErrUnexpectedEOF
		}
		str := u.data[start:u.pos]
		changeValue(v, *(*string)(unsafe.Pointer(&str)))
	}
	return v, nil
}

func (u *Unserializer) decodeFixedInt() (reflect.Value, error) {
	v, err := u.decodeType()
	if err != nil {
		return v, err
	}
	t := u.data[u.pos-1]
	bitSize := 8 << (t & 0b11)
	hasMeta := t&meta != 0
	var size int
	si := u.pos
	sb := u.data[si]
	if hasMeta {
		if bitSize == 64 {
			size = int((u.data[si] & 0b11100000) >> 5)
			u.data[si] &= 0b00011111
		} else if bitSize == 32 {
			size = int((u.data[si] & 0b11000000) >> 6)
			u.data[si] &= 0b00111111
		} else if bitSize == 16 {
			if hasMeta {
				size = 1
			} else {
				size = 2
			}
		} else {
			size = 1
		}
	} else {
		size = bitSize >> 3
	}
	i, err := u.readUint(size)
	u.data[si] = sb // fix the changed first byte
	if err != nil {
		return v, err
	}
	if t&signed == 0 {
		switch bitSize {
		case 64:
			changeValue(v, i)
		case 32:
			changeValue(v, uint32(i))
		case 16:
			changeValue(v, uint16(i))
		default:
			changeValue(v, uint8(i))
		}
	} else {
		neg := i&1 != 0
		i >>= 1
		switch bitSize {
		case 64:
			if neg {
				changeValue(v, ^int64(i))
			} else {
				changeValue(v, int64(i))
			}
		case 32:
			if neg {
				changeValue(v, ^int32(i))
			} else {
				changeValue(v, int32(i))
			}
		case 16:
			if neg {
				changeValue(v, ^int16(i))
			} else {
				changeValue(v, int16(i))
			}
		default:
			if neg {
				changeValue(v, ^int8(i))
			} else {
				changeValue(v, int8(i))
			}
		}
	}
	return v, nil
}

func (u *Unserializer) decodeInt() (reflect.Value, error) {
	v, i, t, err := u.decodeTypeWithLength()
	if err != nil {
		return v, err
	}
	if t&signed == 0 {
		changeValue(v, i)
	} else if i&1 != 0 {
		changeValue(v, ^int(i>>1))
	} else {
		changeValue(v, int(i>>1))
	}
	return v, nil
}

func (u *Unserializer) decodeFloat() (reflect.Value, error) {
	v, f, t, err := u.decodeTypeWithLength()
	if err != nil {
		return v, err
	}
	if t&wide != 0 {
		changeValue(v, math.Float64frombits(bits.ReverseBytes64(f)))
	} else {
		changeValue(v, math.Float32frombits(bits.ReverseBytes32(uint32(f))))
	}
	return v, nil
}

func (u *Unserializer) decodeComplex() (v reflect.Value, err error) {
	v, err = u.decodeType()
	if err != nil {
		return v, err
	}
	t := u.data[u.pos-1]
	var r, i reflect.Value
	if r, err = u.decodeFloat(); err != nil {
		return r, err
	}
	if i, err = u.decodeFloat(); err != nil {
		return i, err
	}
	if t&wide != 0 {
		changeValue(v, complex(r.Interface().(float64), i.Interface().(float64)))
	} else {
		changeValue(v, complex(r.Interface().(float32), i.Interface().(float32)))
	}
	return v, nil
}

func (u *Unserializer) decodeUintptr() (reflect.Value, error) {
	v, ptr, t, err := u.decodeTypeWithLength()
	if err != nil {
		return v, err
	}
	if t&raw != 0 {
		changeValue(v, unsafe.Pointer(uintptr(ptr)))
	} else {
		changeValue(v, uintptr(ptr))
	}
	return v, nil
}

func (u *Unserializer) decodeChan() (reflect.Value, error) {
	v, err := u.decodeType()
	if err != nil {
		return v, err
	}
	size := int(u.data[u.pos]&0b111) + 1
	u.pos++
	buffSize, err := u.readUint(size)
	if err != nil {
		return v, err
	}
	t := v.Type()
	if t.ChanDir() == reflect.BothDir {
		return reflect.MakeChan(t, int(buffSize)), nil
	}
	el := t.Elem()
	biDirChan := reflect.ChanOf(reflect.BothDir, el)
	unDirChan := reflect.ChanOf(t.ChanDir(), el)
	v = reflect.MakeChan(biDirChan, int(buffSize))
	v = v.Convert(unDirChan).Convert(t)
	return v, nil
}

func (u *Unserializer) decodeFunc() (reflect.Value, error) {
	return u.decodeType()
}

func (u *Unserializer) decodeList() (reflect.Value, error) {
	v, l, t, err := u.decodeTypeWithLength()
	if err != nil {
		return v, err
	}
	length := int(l)
	if t&fixed == 0 {
		v = reflect.MakeSlice(v.Type(), length, length)
	}
	u.refs[u.cnt] = v
	u.cnt++
	for i := 0; i < length; i++ {
		if _, err = u.decode(v.Index(i)); err != nil {
			return reflect.Value{}, err
		}
	}
	return v, nil
}

func (u *Unserializer) decodeMap() (reflect.Value, error) {
	mp, l, _, err := u.decodeTypeWithLength()
	if err != nil {
		return mp, err
	}
	length := int(l)
	mp = reflect.MakeMapWithSize(mp.Type(), length)
	u.refs[u.cnt] = mp
	u.cnt++
	var k, v reflect.Value
	for i := 0; i < length; i++ {
		if k, err = u.decode(reflect.Value{}); err != nil {
			return v, err
		}
		if v, err = u.decode(reflect.Value{}); err != nil {
			return v, err
		}
		mp.SetMapIndex(k, v)
	}
	return mp, nil
}

func (u *Unserializer) decodeStruct() (reflect.Value, error) {
	v, l, _, err := u.decodeTypeWithLength()
	if err != nil {
		return v, err
	}
	u.refs[u.cnt] = v
	u.cnt++
	length := int(l)
	for i := 0; i < length; i++ {
		f := v.Field(i)
		f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
		_, err = u.decode(f)
		if err != nil {
			return v, err
		}
	}
	return v, nil
}

func (u *Unserializer) decodePointer() (reflect.Value, error) {
	v, err := u.decodeType()
	if err != nil {
		return v, err
	}
	cnt := u.cnt
	u.refs[cnt] = v
	u.cnt++
	u.rels[u.cnt] = cnt
	el, err := u.decode(reflect.Value{})
	if err != nil {
		return v, err
	}
	if v.IsValid() {
		if el.IsValid() {
			changeValue(v, u.pointerTo(el).Interface())
		}
	} else {
		v = u.pointerTo(el)
	}
	u.refs[cnt] = v
	return v, nil
}

func (u *Unserializer) decodeReference(value reflect.Value) (v reflect.Value, err error) {
	t := u.data[u.pos]
	if t&val != 0 { // forward references
		return u.decodePointedValue(value)
	}
	// back references
	var cnt uint32
	if v, cnt, err = u.extractReferenceValue(); err != nil {
		return v, err
	}
	v = u.pointerTo(v)
	u.refs[u.cnt] = v
	u.rels[cnt] = u.cnt
	u.cnt++
	return v, nil
}

func (u *Unserializer) decodePointedValue(value reflect.Value) (v reflect.Value, err error) {
	var cnt uint32
	if v, cnt, err = u.extractReferenceValue(); err != nil {
		return v, err
	}
	if value.IsValid() {
		value.Set(v)
		v = value
	}
	relCnt, exists := u.rels[cnt]
	if exists {
		r := u.refs[relCnt]
		r.Set(u.pointerTo(v))
	}
	u.refs[cnt] = v
	return v, nil
}

func (u *Unserializer) extractReferenceValue() (reflect.Value, uint32, error) {
	size := (u.data[u.pos] & 0b111) + 1
	u.pos++
	cnt, err := u.readUint(int(size))
	if err != nil {
		return reflect.Value{}, 0, err
	}
	c := uint32(cnt)
	v, ok := u.refs[c]
	if !ok {
		return v, 0, fmt.Errorf("invalid reference #%d", c)
	}
	return v, c, nil
}

func (u *Unserializer) pointerTo(v reflect.Value) reflect.Value {
	var p reflect.Value
	if v.IsValid() {
		if v.CanAddr() {
			p = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr()))
		} else {
			p = reflect.New(v.Type())
			p.Elem().Set(v)
		}
	} else {
		p = reflect.ValueOf((*any)(nil))
	}
	return p
}

func (u *Unserializer) decodeInterface() (reflect.Value, error) {
	v, err := u.decodeType()
	if err != nil {
		return v, err
	}
	u.refs[u.cnt] = v
	return u.decode(v)
}

func (u *Unserializer) decodeType() (reflect.Value, error) {
	typeSignature, err := u.readTypeSignature()
	if len(typeSignature) == 1 && typeSignature[0] == tPointer {
		return reflect.Value{}, err
	}
	if err != nil {
		return reflect.Value{}, err
	}
	return tChecker.valueOf(typeSignature)
}

func (u *Unserializer) decodeTypeWithLength() (reflect.Value, uint64, byte, error) {
	v, err := u.decodeType()
	if err != nil {
		return v, 0, 0, err
	}
	t := u.data[u.pos-1]
	size := int((t & 0b111) + 1)
	length, err := u.readLength(size)
	if err != nil {
		return v, 0, 0, err
	}
	return v, length, t, nil
}

func (u *Unserializer) readLength(lengthSize int) (uint64, error) {
	length, err := u.readUint(lengthSize)
	if lengthSize > 4 && math.MaxUint == math.MaxUint32 {
		return 0, errors.New("int type has 32 bit size but serialized value has 64 bit size")
	}
	if err != nil {
		return 0, err
	}
	return length, nil
}

func (u *Unserializer) readUint(size int) (uint64, error) {
	var i uint64
	for ; size > 0; size-- {
		if u.pos >= u.size {
			return 0, io.ErrUnexpectedEOF
		}
		i = i<<8 | uint64(u.data[u.pos])
		u.pos++
	}
	return i, nil
}

func (u *Unserializer) readTypeSignature() ([]byte, error) {
	if u.pos >= u.size {
		return nil, io.ErrUnexpectedEOF
	}
	t := u.data[u.pos]
	u.pos++
	switch t & mask {
	case tType:
		size := int((t & 0b111) + 1)
		u.pos += size
		if u.pos > u.size {
			return nil, io.ErrUnexpectedEOF
		}
		b := append([]byte{}, u.data[u.pos-size-1:u.pos]...)
		ut, err := u.readTypeSignature()
		if err != nil {
			return nil, err
		}
		b[0] &= mask
		return append(b, ut...), nil
	case tList:
		return []byte{t & (mask | fixed)}, nil
	case tInt8, tInt16, tInt32, tInt64:
		return []byte{t & (mask | 0b11 | signed)}, nil
	case tInt:
		return []byte{t & (mask | signed)}, nil
	case tFloat, tComplex:
		return []byte{t & (mask | wide)}, nil
	case tUintptr:
		return []byte{t & (mask | raw)}, nil
	case tChan:
		if t&tFunc == tFunc {
			return []byte{tFunc}, nil
		}
		return []byte{t & (mask | 0b11)}, nil
	default:
		return []byte{t & mask}, nil
	}
}

func Unserialize(bytes []byte) (any, error) {
	return NewUnserializer().Decode(bytes)
}
