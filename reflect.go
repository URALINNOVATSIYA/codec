package codec

import (
	"reflect"
	"unsafe"
)

type flag uintptr

type flagROTester struct {
	A int
	a int
	int
}

var flagOffset uintptr
var maskFlagRO flag
var hasExpectedReflectStruct bool

func setValue(oldValue reflect.Value, newValue reflect.Value) {

	if hasExpectedReflectStruct {
		pFlag := (*flag)(unsafe.Pointer(uintptr(unsafe.Pointer(&newValue)) + flagOffset))
		*pFlag &= maskFlagRO
		oldValue.Set(newValue)
	} else {
		oldValue.Set(newValue)
	}
}
