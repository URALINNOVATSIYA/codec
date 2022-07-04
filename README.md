# Overview

Package codec contains a serializer with that you can serialize (and unserialize)
go object of any type (including channels and functions) in runtime.

## Install

```go
go get -u github.com/URALINNOVATSIYA/codec
```

# Usage

To serialize any value in binary data use function Serialize:

```go
value := ... // a value to serialize

data := Serialize(value)
```

or use type Serializer:

```go
value := ... // a value to serialize

serializer := NewSerializer()
data := serializer.Encode(value)
```

To unserialize binary data do as follows:

```go
var data []byte = ... // serialized value

value, err := Unserialize(data)
```

or 

```go
var data []byte = ... // serialized value

unserializer := NewUnserializer()
value, err := unserializer.Decode(data)
```

# Type registration

To correct recover a serialized value we need to create its type dynamically 
in runtime. For this, we must register type of value before its serialization 
and unserialization.

By default, serializer registers each encountered type automatically.
It's enough for some cases, but somewhere it leads to violation of type
registration order. To ensure fixed order of type registration we should
register each type in out program before any serialization or unserialization.

We can do it as follows:

```go
// register zero value of a type
RegisterTypeOf(map[string][]bool{})
RegisterTypeOf(struct{b bool; i int}{})

// or register type directly
v := [][]any{} 
RegisterType(reflect.TypeOf(v))
```

**Note:** ```RegisterType``` works for functions but cannot guarantee uniqueness
of function value because function type does not have ant information about function
name. To get full information you must register functions through ```RegisterTypeOf```  

You can also turn off automatic registration of types calling ```TurnOffTypeAutoRegistration```

# Custom serialization

To implement your own custom serialization a type must implements ```Serializable``` interface:

```go
type Flag bool

func (f Flag) Serialize() []byte {
	if f {
		return []byte{1}
	}
	return []byte{0}
}

func (f Flag) Unserialize(data []byte) (any, error) {
	return data[0] == 1, nil
}
```

**Note:** methods of ```Serializable``` must relate to a type value, not to pointer to the value.  