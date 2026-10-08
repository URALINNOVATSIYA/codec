## Serialized Data Format

This document describes version 1 of the binary format produced by the
`codec` package.

## Contents

- [Notation](#notation)
- [Top-level layout](#top-level-layout)
- [Header sections](#header-sections)
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
- [References, pointers, and shared slices](#references-pointers-and-shared-slices)
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
header [references_section] [pointers_section] [slices_section] node_stream
```

The one-byte header contains the version in its low four bits and flags for
optional sections in its high bits. Version 1 is `0x01`; the section flags are
`0x10` for references, `0x20` for pointers, and `0x40` for shared slices. If
more than one section is present, they appear in the order shown above.

The node stream is the concatenation of the root and any other top-level
values, with no separate length or terminator. Each top-level node starts with
a type identifier followed by its value. Nested fields and elements are
normally encoded using their declared type and do not repeat a type
identifier; dynamic interface values do.

Node IDs are implicit in the traversal order and are used by the optional
sections. The decoder reads those sections first, then decodes nodes through
the end of the input. This allows links to refer to nodes decoded later.

## Header sections

Each section is present only when its corresponding header flag is set. All
counts and IDs in these sections use `u4`.

The references section groups alias locations by the ID of their target:

```text
u4(target_count)
{ u4(target_id) u4(alias_count) { u4(alias_id) } }
```

The pointers section groups pointer locations by the ID of the value they
point to:

```text
u4(target_count)
{ u4(target_id) u4(pointer_count) { u4(pointer_id) } }
```

The shared-slices section records expressions that recreate slices or strings
over a decoded parent value:

```text
u4(parent_count)
{ u4(parent_id) u4(expression_count)
  { u4(target_id) u4(start) u4(end) u4(max) } }
```

`max` is the three-index slice bound; for strings the decoder uses only
`start` and `end`. A target node containing a shared value has a `meta_ref`
marker in the node stream. The tables provide the target and parent IDs; no
IDs or indexes follow that marker inline.

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

Every top-level node and dynamic interface value starts with a `u3` type ID.
Nested fields and elements use the type declared by their container and do not
carry a type ID of their own. IDs are assigned by `TypeRegistry` and must
resolve to the same Go types during decoding. Function values use a second
`u3` function ID in their payload.

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

A string without shared storage is encoded as its `u4` byte length followed by
its bytes:

```text
u4(byte_length) string_bytes
```

A string that shares storage with another string or slice is represented by a
`meta_ref` marker in the node stream and an entry in the shared-slices header
section. Repeated strings can also use the references section. Their links are
restored after the nodes have been decoded.

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

A nil slice is encoded as `meta_nil`. A non-nil slice is encoded as:

```text
meta_nonil length capacity element_values...
```

`length` and `capacity` are `u4` values. A non-nil zero-capacity slice still
stores its length and capacity, but has no element payload. A slice or string
that shares a backing value is represented by `meta_ref`; its parent ID and
bounds are stored in the shared-slices section.

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
the `codec:"id=N"` struct tags. Fields without a tag and fields marked
`deprecated` are omitted. When decoding, fields absent from the stream retain
their zero values; an unknown field ID is an error.

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

## References, pointers, and shared slices

The byte `meta_ref` (`0x00`) marks a value whose contents or storage are
represented elsewhere. It is followed by no inline ID. For ordinary aliases,
the references section maps the target node ID to one or more alias location
IDs. Pointer relationships are recorded separately in the pointers section.
Slice and string expressions are recorded in the shared-slices section.

The decoder first reads the flagged sections, then reads the node stream and
restores slice expressions, pointers, ordinary references, and deferred map
entries. Since IDs are implicit positions in the traversed graph, a section
can safely describe a link to a node that appears later in the node stream.

## Compatibility notes

- The low four bits of the first byte contain the format version; the high bits
  indicate which optional header sections follow.
- Type IDs are registry-local; serialized data is not self-contained without a
	compatible type registry.
- Function IDs identify registered function values and cannot be reconstructed
	from a function type alone.
- `unsafe.Pointer` stores the numeric pointer value, but that address is not
	valid as a portable reconstructed pointer.
- `uintptr` has the same address-portability limitation.
- Channel state and buffered elements are not part of the format; only nilness,
  direction, and capacity are represented.
