package codec

import (
	"reflect"
	"unsafe"

	"github.com/URALINNOVATSIYA/reflex"
)

type Addr struct {
	Ptr  unsafe.Pointer
	Type reflect.Type
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
			Type: v.Type(),
		}
	case reflect.String, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Pointer:
		return Addr{
			Ptr:  reflex.DirPtrOf(v),
			Type: v.Type(),
		}
	}
	return Addr{}
}
