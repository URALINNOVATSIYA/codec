package codec

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/URALINNOVATSIYA/reflex"
)

type check func(any) bool

type testItem struct {
	value any
	data  []byte
	check check
}

func runTests(items []testItem, typeRegistry *TypeRegistry, t *testing.T) {
	serializer := NewSerializer().WithTypeRegistry(typeRegistry)
	unserializer := NewUnserializer().WithTypeRegistry(typeRegistry)
	for i, item := range items {
		expected := item.value
		data := serializer.Encode(expected)
		if !bytes.Equal(data, item.data) {
			t.Errorf("Test #%d: Encode(%T) must return %v, but actual value is %v", i+1, expected, item.data, data)
			continue
		}
		actual, err := unserializer.Decode(data)
		if err != nil {
			t.Errorf("Test #%d: Decode(%T) raises error: %q", i+1, expected, err)
		} else  {
			var equals bool
			if expected != nil {
				switch reflect.TypeOf(expected).Kind() {
				case reflect.Chan:
					equals = channelEqual(expected, actual)
				case reflect.Func:
					equals = funcEqual(expected, actual)
				default:
					equals = reflect.DeepEqual(expected, actual)
				}
			} else {
				equals = reflect.DeepEqual(expected, actual)
			}
			if !equals {
				t.Errorf("Test #%d: Decode(%T) returns wrong value %T", i+1, expected, actual)
			} else if item.check != nil && !item.check(actual) {
				t.Errorf("Test #%d: Decode(%T) returns value of wrong structure", i+1, expected)
			}
		}
	}
}

func registry() (*TypeRegistry, func(v any) byte) {
	reg := NewTypeRegistry(true)
	return reg, func(v any) byte {
		id := reg.typeIdByValue(reflect.ValueOf(v))
		return u2bs(uint64(id), 3)[0]
	}
}

func interfaceId(reg *TypeRegistry) byte {
	if id, exists := reg.typeIdByName("interface {}"); exists {
		return u2bs(uint64(id), 3)[0]
	}
	reg.RegisterType(reflect.TypeOf((*any)(nil)).Elem())
	return interfaceId(reg)
}

func channelEqual(expected any, actual any) bool {
	if reflex.NameOf(reflect.TypeOf(expected)) != reflex.NameOf(reflect.TypeOf(actual)) {
		return false
	}
	return reflect.ValueOf(expected).Cap() == reflect.ValueOf(actual).Cap()
}

func funcEqual(expected any, actual any) bool {
	return reflex.FuncNameOf(reflect.ValueOf(expected)) == reflex.FuncNameOf(reflect.ValueOf(actual))
}

func Test_Nil(t *testing.T) {
	reg, typeId := registry()
	items := []testItem{
		{
			nil,
			[]byte{version, id(0), typeId(nil), meta_nil},
			nil,
		},
	}
	runTests(items, reg, t)
}