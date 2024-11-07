package codec

import (
	"reflect"
)

var serializableInterfaceType = reflect.TypeOf((*Serializable)(nil)).Elem()

func isSerializable(v reflect.Value) bool {
	if v.IsValid() {
		t := v.Type()
		if isPointer(t) {
			if isCommonType(t.Elem()) {
				return false
			}
		} else if isCommonType(t) {
			return false
		}
		return t.Implements(serializableInterfaceType)
	}
	return false
}

func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Interface,
		reflect.Map, reflect.Slice,
		reflect.Pointer, reflect.UnsafePointer,
		reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

func isPointer(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer
}

func isSimplePointer(t reflect.Type) bool {
	return isPointer(t) && isCommonType(t)
}

func isCommonType(t reflect.Type) bool {
	name := t.Name()
	if name == "" {
		return true
	}
	if name == t.Kind().String() {
		return true
	}
	if t.Kind() == reflect.UnsafePointer && name == "Pointer" {
		return true
	}
	return false
}
