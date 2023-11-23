package codec

import "reflect"

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
