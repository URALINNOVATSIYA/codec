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

func initReflectFlags() {
	if field, ok := reflect.TypeOf(reflect.Value{}).FieldByName("flag"); ok {
		flagOffset = field.Offset
	} else {

		hasExpectedReflectStruct = false
		return
	}

	rv := reflect.ValueOf(flagROTester{})
	getFlag := func(v reflect.Value, name string) flag {
		return flag(reflect.ValueOf(v.FieldByName(name)).FieldByName("flag").Uint())
	}
	flagRO := (getFlag(rv, "a") | getFlag(rv, "int")) ^ getFlag(rv, "A")
	maskFlagRO = ^flagRO

	if flagRO == 0 {
		hasExpectedReflectStruct = false
		return
	}

	hasExpectedReflectStruct = true
}

func setValue(oldValue reflect.Value, newValue reflect.Value) {

	if hasExpectedReflectStruct {
		pFlag := (*flag)(unsafe.Pointer(uintptr(unsafe.Pointer(&newValue)) + flagOffset))
		*pFlag &= maskFlagRO
		oldValue.Set(newValue)
	} else {
		oldValue.Set(newValue)
	}
}
