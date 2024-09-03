package codec

import (
	"fmt"
	"reflect"
)

type Serializer struct {
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
	return append([]byte{version}, s.encode(reflect.ValueOf(value))...)
}

func (s *Serializer) encode(v reflect.Value) []byte {
	return append(s.encodeType(v), s.encodeValue(v)...)
}

func (s *Serializer) encodeType(v reflect.Value) []byte {
	return s.typeRegistry.encodedType(v)
}

func (s *Serializer) encodeValue(v reflect.Value) []byte {
	switch v.Kind() {
	case reflect.Invalid: // nil
		return nil
	case reflect.Bool:
		return s.encodeBool(v)
	case reflect.String:
		return s.encodeString(v)	
	default:
		panic(fmt.Errorf("unsupported type kind %q", v.Kind()))
	}
}

func (s *Serializer) encodeBool(v reflect.Value) []byte {
	if v.Bool() {
		return []byte{1}
	}
	return []byte{0}
}

func (s *Serializer) encodeString(v reflect.Value) []byte {
	return append(s.encodeLength(v.Len()), v.String()...)
}

func (s *Serializer) encodeLength(length int) []byte {
	size := minByteSizeWithMeta(uint64(length), 3)
	return asBytesOfSize(uint64(length) | uint64((size-1) << (8 * size - 3)), size)
}

func Serialize(value any, options ...any) []byte {
	return NewSerializer().
		WithOptions(options).
		Encode(value)
}
