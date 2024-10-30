package codec

import (
	"reflect"
	"unsafe"
)

const maskFlagRO = ^uintptr(96)

var flagOffset uintptr
var hasFlagField bool

var serializableInterfaceType = reflect.TypeOf((*Serializable)(nil)).Elem()

func determineReflectValueFlagOffset() {
	if field, ok := reflect.TypeOf(reflect.Value{}).FieldByName("flag"); ok {
		flagOffset = field.Offset
		hasFlagField = true
	} else {
		hasFlagField = false
	}
}

func makeExportable(v reflect.Value) reflect.Value {
	if hasFlagField {
		flag := (*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(&v)) + flagOffset))
		*flag &= maskFlagRO
	}
	return v
}

func setValue(oldValue reflect.Value, newValue reflect.Value) {
	oldValue.Set(makeExportable(newValue))
}

func changeValue(oldValue reflect.Value, newValue any) {
	oldValue.Set(reflect.ValueOf(newValue).Convert(oldValue.Type()))
}

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

func zeroValueOf(t reflect.Type) reflect.Value {
	if t == nil {
		return reflect.Value{}
	}
	return reflect.New(t).Elem()
}

func ptrTo(t reflect.Type, v reflect.Value) reflect.Value {
	//if v.CanAddr() {
	//return reflect.NewAt(t, v.Addr().UnsafePointer())
	//}
	p := reflect.New(t)
	p.Elem().Set(v)
	return p
}

func ptr(v reflect.Value) unsafe.Pointer {
	rv := reflect.ValueOf(v)
	f := makeExportable(rv.FieldByName("ptr"))
	return f.Interface().(unsafe.Pointer)
}
