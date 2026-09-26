## Serialized Data Format

This document describes version 1 of the binary format produced by the
`codec` package.

## Contents

- [Notation](#notation)
- [Top-level layout](#top-level-layout)
- [Self-delimiting integers](#self-delimiting-integers)
- [Type identifiers](#type-identifiers)
- [Value encoding](#value-encoding)
	- [Nil and booleans](#nil-and-booleans)
	- [Integers](#integers)
	- [Floating-point and complex values](#floating-point-and-complex-values)
	- [Strings](#strings)
	- [Functions and channels](#functions-and-channels)
	- [Arrays and slices](#arrays-and-slices)
	- [Maps](#maps)
	- [Structs](#structs)
	- [Interfaces](#interfaces)
	- [Pointers](#pointers)
	- [Custom serializable values](#custom-serializable-values)
- [References and pointer restoration](#references-and-pointer-restoration)
- [Compatibility notes](#compatibility-notes)

## Notation

The following notation is used throughout this document:

- `[...]` means an optional component, present zero or one time.
- `{...}` means a repeated component, present zero or more times.
- `uN` means an unsigned integer encoded with `N` length bits.
- `iN` means a signed integer encoded using the unsigned `i2u` transform and
	`N` length bits.
- All multi-byte payloads are stored in big-endian order.

## Top-level layout

A serialized value has this layout:

```text
version encoded_values [pointer_links]
```

`version` is one byte. Version 1 is encoded as `0x01`.

The root value and every value reachable from it are encoded as nodes. A node
has a type identifier followed by the representation of that value:

```text
node = type_id value
```

The value graph is written in traversal order. References can point to an
earlier node, which allows shared values and cycles to be restored.

## Self-delimiting integers

The format stores unsigned integers in the smallest possible number of bytes.
The first byte carries the total encoded length in its leading length bits; the
remaining bits and following bytes carry the value. The length includes the
first byte.

Two variants are used:

| Encoding | Length bits | Used for |
| --- | ---: | --- |
| `u3` | 3 | type IDs and function IDs |
| `u4` | 4 | lengths, capacities, indexes, counts, and references |

For example, `u4` reserves the four most significant bits of the first byte
for the encoded byte count. The value occupies the remaining bits of that
byte and all subsequent bytes. Values are unsigned unless explicitly stated
otherwise.

Signed integers use zigzag-like encoding:

```text
i2u(n) = n << 1                 when n >= 0
i2u(n) = (^n << 1) | 1          when n < 0
```

The resulting unsigned value is then encoded with the appropriate integer
width.

## Type identifiers

Every node starts with a `u3` type ID. IDs are assigned by `TypeRegistry` and
must resolve to the same Go types during decoding. Function values use a
second `u3` function ID in their payload.

Type registration is therefore part of the format contract: a decoder must
have the types and function values referenced by the stream in its registry.

## Value encoding

### Nil and booleans

| Value | Bytes |
| --- | --- |
| `nil` | `0x10` (`meta_nil`) |
| `false` | `0x01` (`meta_fls`) |
| `true` | `0x03` (`meta_tru`) |

### Integers

`uint8` and `int8` occupy one byte without a length prefix. Signed `int8`
uses the `i2u` transform.

The other integer types use a self-delimiting integer:

| Go kind | Encoding |
| --- | --- |
| `uint16`, `int16` | `u2`, `i2` |
| `uint32`, `int32` | `u3`, `i3` |
| `uint64`, `int64` | `u4`, `i4` |
| `uint`, `int` | same as `uint64`, `int64` |
| `uintptr` | same as `uint64` |

`uintptr` is encoded as its numeric address-sized value, but this value is
not a portable or safely restorable pointer identity.

### Floating-point and complex values

`float32` is encoded as its IEEE-754 bit pattern, byte-reversed by the codec,
then written as `u3`. `float64` is encoded the same way with `u4`.

`complex64` is the concatenation of an encoded `float32` real part and an
encoded `float32` imaginary part. `complex128` contains two encoded `float64`
values in the same order.

### Strings

A string is encoded as its byte length followed by its UTF-8/string bytes:

```text
u4 byte_length string_bytes
```

Repeated strings may instead be represented by a reference node.

### Functions and channels

A nil function or channel is encoded as `meta_nil`.

A non-nil function is encoded as:

```text
meta_nonil function_id
```

where `function_id` is a `u3`. The function value itself is not serialized;
the decoder obtains the registered function from its `TypeRegistry`.

A non-nil channel is encoded as:

```text
meta_nonil capacity
```

where `capacity` is a `u4`. Channel contents are not serialized. The decoder
creates a new channel with the recorded capacity and direction.

### Arrays and slices

An array has no header. Its elements are encoded in index order as values of
the array element type.

A slice starts with `meta_slice`:

```text
meta_slice length capacity element_values...
```

`length` and `capacity` are `u4` values. A nil slice is marked by setting the
`meta_nil` bit in the slice header:

```text
meta_slice | meta_nil
```

A slice created from another slice with a slicing expression uses
`meta_slice | meta_sexp` and stores:

```text
meta_slice | meta_sexp parent_id start end max
```

`parent_id`, `start`, `end`, and `max` are `u4` values, except that the parent
ID is decoded using the signed integer path because it is stored with the
same compact integer helper. The expression is applied after all nodes have
been read.

### Maps

A nil map is encoded as `meta_nil`. A non-nil map is encoded as:

```text
meta_nonil entry_count key value key value ...
```

`entry_count` is a `u4`. Keys and values are encoded as nodes or values of
their declared map types. Entries involving pointers or interfaces are
completed during the map-restoration phase.

### Structs

A struct starts with `meta_strc` and then contains its fields in declaration
order:

```text
meta_strc field_values...
```

If at least one field has a `codec` tag, the header also contains
`meta_tags`:

```text
meta_strc | meta_tags tagged_field_count
field_id field_value ...
```

`tagged_field_count` and every `field_id` are `u4` values. Field IDs come from
the `codec:"id=N"` struct tags. A field tagged `deprecated` is retained in the
registry metadata but may be omitted or ignored by application compatibility
logic.

### Interfaces

An interface contains the complete node for its dynamic value:

```text
dynamic_type_id dynamic_value
```

The dynamic type ID, rather than the static interface type, determines how
the value is decoded.

### Pointers

A nil pointer is encoded as `meta_nil`. A non-nil pointer is encoded as
`meta_nonil`; its pointed-to value is stored as another node and pointer
relationships are restored after the main value stream has been decoded.

### Custom serializable values

A type implementing `Serializable` uses the result of its `Serialize` method:

```text
meta_nil
```

for a nil value, or:

```text
meta_nonil body_length body_bytes
```

for a non-nil value. `body_length` is a `u4`. The decoder passes `body_bytes`
to the type's `Unserialize` method.

## References and pointer restoration

The byte `meta_ref` (`0x00`) marks a reference. It is followed by a `u4` node
ID:

```text
meta_ref referenced_node_id
```

This representation is used for repeated strings, maps, slices, channels,
functions, custom serializable reference-like values, and other shared
objects.

After the encoded values, the stream may contain pointer-link records:

```text
meta_ref target_id pointer_id {pointer_id}
```

Each ID is a `u4`. The decoder assigns pointers to the target value during its
pointer-restoration phase. The stream ends after the last pointer-link record.

## Compatibility notes

- The first byte is the format version and must be checked before decoding.
- Type IDs are registry-local; serialized data is not self-contained without a
	compatible type registry.
- Function IDs identify registered function values and cannot be reconstructed
	from a function type alone.
- `unsafe.Pointer` stores the numeric pointer value, but that address is not
	valid as a portable reconstructed pointer.
- `uintptr` has the same address-portability limitation.
- Channel state and buffered elements are not part of the format; only nilness,
	direction, and capacity are represented.
