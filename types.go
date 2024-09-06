package codec

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

type TypeRegistry struct {
	typeAutoReg bool
	types       map[int]reflect.Type  // registered types
	funcs       map[int]reflect.Value // registered functions
	ids         map[string]int        // type full names and their ids
	mx          sync.RWMutex
}

func NewTypeRegistry(typeAutoReg bool) *TypeRegistry {
	return &TypeRegistry{
		typeAutoReg: typeAutoReg,
		types:       make(map[int]reflect.Type),
		funcs:       make(map[int]reflect.Value),
		ids:         make(map[string]int),
	}
}

func (r *TypeRegistry) TurnOnTypeAutoRegistration() {
	r.typeAutoReg = true
}

func (r *TypeRegistry) TurnOffTypeAutoRegistration() {
	r.typeAutoReg = false
}

func (r *TypeRegistry) RegisteredTypeNames() []string {
	names := make([]string, len(r.ids))
	for name, i := range r.ids {
		names[i] = name
	}
	return names
}

func (r *TypeRegistry) RegisterBaseTypes() {
	r.RegisterTypeOf(nil)
	r.RegisterTypeOf(false)
	r.RegisterTypeOf("")
	r.RegisterTypeOf(int8(0))
	r.RegisterTypeOf(uint8(0))
	r.RegisterTypeOf(int16(0))
	r.RegisterTypeOf(uint16(0))
	r.RegisterTypeOf(int32(0))
	r.RegisterTypeOf(uint32(0))
	r.RegisterTypeOf(int64(0))
	r.RegisterTypeOf(uint64(0))
	r.RegisterTypeOf(int(0))
	r.RegisterTypeOf(uint(0))
	r.RegisterTypeOf(float32(0))
	r.RegisterTypeOf(float64(0))
	r.RegisterTypeOf(complex64(0))
	r.RegisterTypeOf(complex128(0))
	r.RegisterTypeOf(uintptr(0))
	r.RegisterTypeOf(unsafe.Pointer(nil))
	r.RegisterType(reflect.TypeOf((*any)(nil)).Elem()) // interface {}
}

func (r *TypeRegistry) RegisterTypeOf(v any) {
	t := reflect.TypeOf(v)
	if t != nil && t.Kind() == reflect.Func {
		r.RegisterFunc(v)
	} else {
		r.RegisterType(t)
	}
}

func (r *TypeRegistry) RegisterType(t reflect.Type) {
	name := r.typeName(t)
	if _, exists := r.typeIdByName(name); exists {
		return
	}
	r.bindTypeWithName(t, name)
}

func (r *TypeRegistry) RegisterFunc(f any) {
	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Func {
		panic(fmt.Errorf("argument of RegisterFunc is not function"))
	}
	name := r.funcName(v)
	if _, exists := r.typeIdByName(name); exists {
		return
	}
	r.bindFuncWithName(v, name)
}

func (r *TypeRegistry) typeIdByValue(v reflect.Value) int {
	var name string
	var t reflect.Type
	if v.Kind() == reflect.Func {
		name = r.funcName(v)
	} else {
		if v.IsValid() {
			t = v.Type()
		}
		name = r.typeName(t)
	}
	if id, exists := r.typeIdByName(name); exists {
		return id
	}
	if !r.typeAutoReg {
		panic(fmt.Errorf("unregistered type: %s", name))
	}
	var id int
	if v.Kind() == reflect.Func {
		id = r.bindFuncWithName(v, name)
	} else {
		id = r.bindTypeWithName(t, name)
	}
	return id
}

func (r *TypeRegistry) typeIdByName(name string) (id int, exists bool) {
	r.mx.RLock()
	id, exists = r.ids[name]
	r.mx.RUnlock()
	return
}

func (r *TypeRegistry) bindTypeWithName(t reflect.Type, name string) int {
	r.mx.Lock()
	id := r.assignTypeId(name)
	r.types[id] = t
	r.mx.Unlock()
	return id
}

func (r *TypeRegistry) bindFuncWithName(v reflect.Value, name string) int {
	r.mx.Lock()
	id := r.assignTypeId(name)
	r.types[id] = v.Type()
	r.funcs[id] = v
	r.mx.Unlock()
	return id
}

func (r *TypeRegistry) assignTypeId(name string) int {
	id, exists := r.ids[name]
	if !exists {
		id = len(r.ids) + 1
		r.ids[name] = id
	}
	return id
}

func (r *TypeRegistry) funcName(v reflect.Value) string {
	name := funcName(v)
	if name == "" {
		name = r.typeName(v.Type())
	}
	return name
}

func (r *TypeRegistry) typeName(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	if t.Name() != "" {
		name := t.Name()
		if t.PkgPath() != "" {
			name = t.PkgPath() + "." + name
		}
		return name
	}
	switch t.Kind() {
	case reflect.Invalid:
		return "<nil>"
	case reflect.Pointer:
		return "*" + r.typeName(t.Elem())
	case reflect.Slice:
		return "[]" + r.typeName(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), r.typeName(t.Elem()))
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", r.typeName(t.Key()), r.typeName(t.Elem()))
	case reflect.Struct:
		f := ""
		for i, numFields := 0, t.NumField(); i < numFields; i++ {
			field := t.Field(i)
			if i > 0 {
				f += "; "
			}
			f += fmt.Sprintf("%s %s", field.Name, r.typeName(t.Field(i).Type))
		}
		return fmt.Sprintf("struct { %s }", f)
	case reflect.Chan:
		switch t.ChanDir() {
		case reflect.BothDir:
			return "chan " + r.typeName(t.Elem())
		case reflect.RecvDir:
			return "<-chan" + r.typeName(t.Elem())
		default:
			return "chan<-" + r.typeName(t.Elem())
		}
	}
	return t.String()
}

func GetDefaultTypeRegistry() *TypeRegistry {
	return defaultTypeReg
}

func GetRegisteredTypeNames() []string {
	return GetDefaultTypeRegistry().RegisteredTypeNames()
}

func RegisterTypeOf(value any) {
	GetDefaultTypeRegistry().RegisterTypeOf(value)
}

func RegisterType(t reflect.Type) {
	GetDefaultTypeRegistry().RegisterType(t)
}

func RegisterFunc(v any) {
	GetDefaultTypeRegistry().RegisterFunc(v)
}

func TurnOffTypeAutoRegistration() {
	GetDefaultTypeRegistry().TurnOffTypeAutoRegistration()
}

func TurnOnTypeAutoRegistration() {
	GetDefaultTypeRegistry().TurnOnTypeAutoRegistration()
}
