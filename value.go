package codec

import (
	"reflect"
	"unsafe"

	"github.com/URALINNOVATSIYA/reflex"
)

type Addr struct {
	Ptr  unsafe.Pointer
	Type string
}

func (a Addr) IsValid() bool {
	return a.Ptr != nil
}

func Address(v reflect.Value) Addr {
	if !v.IsValid() {
		return Addr{}
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Array:
		return Addr{
			Ptr:  reflex.PtrOf(v),
			Type: reflex.NameOf(v.Type()),
		}
	case reflect.String, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Pointer:
		return Addr{
			Ptr:  reflex.DirPtrOf(v),
			Type: reflex.NameOf(v.Type()),
		}
	}
	return Addr{}
}

type Cell struct {
	Id  int
	Ref int
}

type Value struct {
	Ref int
	V   reflect.Value
}

func (v Value) Name() string {
	if !v.V.IsValid() {
		return "<nil>"
	}
	return reflex.NameOf(v.V.Type())
}
