package codec

import (
	"reflect"
	"unsafe"
)

const maskFlagRO = ^uintptr(96)

var flagOffset uintptr
var hasFlagField bool

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
