package codec

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/URALINNOVATSIYA/reflex"
)

type TypeRegistry struct {
	typeAutoReg bool
	types       map[int]reflect.Type  // registered types
	funcs       map[int]reflect.Value // registered functions
	ids         map[string]int        // type or func full names and their ids
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
	r.RegisterType(reflect.TypeFor[any]()) // interface {}
}

func (r *TypeRegistry) RegisterTypeOf(v any) {
	t := reflect.TypeOf(v)
	r.RegisterTypeRec(t)
	if t != nil && t.Kind() == reflect.Func {
		r.RegisterFunc(v)
	}
}

func (r *TypeRegistry) RegisterTypeRec(t reflect.Type) {
	r.RegisterType(t)
	switch t.Kind() {
	case reflect.Map:
		r.RegisterTypeRec(t.Key())
		fallthrough
	case reflect.Array, reflect.Slice, reflect.Chan, reflect.Pointer:
		r.RegisterTypeRec(t.Elem())
	case reflect.Struct:
		for field := range t.Fields() {
			r.RegisterTypeRec(field.Type)
		}
	case reflect.Func:
		for in := range t.Ins() {
			r.RegisterTypeRec(in)
		}
		for out := range t.Outs() {
			r.RegisterTypeRec(out)
		}
	}
}

func (r *TypeRegistry) RegisterType(t reflect.Type) {
	name := reflex.NameOf(t)
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
	name := reflex.FuncNameOf(v)
	if _, exists := r.typeIdByName(name); exists {
		return
	}
	r.bindFuncWithName(v, name)
}

func (r *TypeRegistry) typeById(id int) reflect.Type {
	r.mx.RLock()
	t, exists := r.types[id]
	r.mx.RUnlock()
	if !exists {
		panic(fmt.Errorf("unrecognized type [id: %d]", id))
	}
	return t
}

func (r *TypeRegistry) funcById(id int) reflect.Value {
	r.mx.RLock()
	f, exists := r.funcs[id]
	r.mx.RUnlock()
	if !exists {
		panic(fmt.Errorf("unrecognized func [id: %d]", id))
	}
	return f
}

func (r *TypeRegistry) typeIdByValue(v reflect.Value) int {
	var t reflect.Type
	if v.IsValid() {
		t = v.Type()
	}
	name := reflex.NameOf(t)
	if id, exists := r.typeIdByName(name); exists {
		return id
	}
	if !r.typeAutoReg {
		panic(fmt.Errorf("unregistered type: %s", name))
	}
	return r.bindTypeWithName(t, name)
}

func (r *TypeRegistry) funcIdByValue(v reflect.Value) int {
	if v.Kind() != reflect.Func {
		panic("argument must be a function")
	}
	name := reflex.FuncNameOf(v)
	if id, exists := r.typeIdByName(name); exists {
		return id
	}
	if !r.typeAutoReg {
		panic(fmt.Errorf("unregistered func: %s", name))
	}
	r.RegisterType(v.Type())
	return r.bindFuncWithName(v, name)
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
