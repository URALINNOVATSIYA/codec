package codec

import (
	"reflect"
	"runtime"
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

func setValue(oldValue reflect.Value, newValue reflect.Value) {
	if hasFlagField {
		flag := (*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(&newValue)) + flagOffset))
		*flag &= maskFlagRO
	}
	oldValue.Set(newValue)
}

func changeValue(value reflect.Value, newValue any) {
	value.Set(reflect.ValueOf(newValue).Convert(value.Type()))
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

func funcName(f reflect.Value) string {
	if f.Kind() != reflect.Func {
		return ""
	}
	return runtime.FuncForPC(f.Pointer()).Name()
}
