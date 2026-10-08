## Serialized Data Format

This document describes the binary format emitted by the `codec` package.
The current implementation is version 1; the first byte of every payload is a
header byte whose low 4 bits contain the format version and whose high bits flag
which metadata sections are present before the node stream.

## Contents

- [Notation](#notation)
- [Top-level layout](#top-level-layout)
- [Header sections](#header-sections)
- [Self-delimiting integers](#self-delimiting-integers)
- [Type identifiers](#type-identifiers)
- [Value encoding](#value-encoding)
  - [Nil and bools](#nil-and-bools)
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

- `[...]` means an optional component.
- `{...}` means a repeated component.
- `uN` means an unsigned integer encoded in `N` bits with a length prefix.
- `iN` means a signed integer encoded by the same compact format after applying
  the `i2u` transform.
- Multi-byte values are stored in big-endian order.

## Top-level layout

A serialized value is laid out as:

```text
header [references_section] [pointers_section] [shared_slices_section] node_stream
```

The first byte is the header. Its low four bits encode the version number; in
this implementation, version 1 is `0x01`.

The high bits are flags:

- `0x10` (`meta_sref`): references section follows
- `0x20` (`meta_sptr`): pointers section follows
- `0x40` (`meta_sslc`): shared-slices section follows

Those sections are written in the order shown above whenever present.

The node stream contains the root value and every reachable value. Each node is
written in traversal order. A node contains a type ID followed by the encoded
payload for that value. The type ID is only written for top-level values and for
interface values whose dynamic type must be decoded.

The optional sections describe aliases, pointer targets, and shared backing
storage. They do not replace the node payloads; they provide additional graph
relationships that are restored after the node stream is decoded.

## Header sections

All counters and IDs in the header sections use `u4` encoding.

### References section

The references section groups aliases by target node ID:

```text
u4(target_count)
{ u4(target_id) u4(alias_count) { u4(alias_id) } }
```

This records places where the same logical value is reused. In the node stream,
repeated values are marked with `meta_ref`; the section says which node ID they
point to and which locations should receive the same final value after decoding.

### Pointers section

The pointers section groups pointer locations by the value they point to:

```text
u4(target_count)
{ u4(target_id) u4(pointer_count) { u4(pointer_id) } }
```

A pointer payload is a non-nil marker in the node stream; the section says which
pointer locations must be assigned to which target value after the main graph is
restored.

### Shared-slices section

The shared-slices section records slice and string expressions that must be
recreated from a parent value:

```text
u4(parent_count)
{ u4(parent_id) u4(expression_count)
  { u4(target_id) u4(start) u4(end) u4(max) } }
```

The destination node is the value that shares storage with the parent. For a
string, only `start` and `end` are meaningful; for a slice, `max` is also used.
The node stream marks the shared value with `meta_ref`, while the section carries
its actual slice bounds and parent relationship.

## Self-delimiting integers

Unsigned integers are stored using the minimum byte count. The first byte stores
how many bytes are used in total, encoded in the leading length bits. The value
bits then follow in the remaining bits of that byte and in subsequent bytes.

Two width families are used:

| Encoding | Length bits | Used for |
| --- | ---: | --- |
| `u3` | 3 | type IDs and function IDs |
| `u4` | 4 | counts, lengths, capacities, indexes, references |

For example, `u4` reserves the four most significant bits of the first byte for
byte-count metadata; the value bits occupy the remaining bits plus the following
bytes.

Signed values are encoded by applying the transform below before writing the
unsigned form:

```text
i2u(n) = n << 1                 when n >= 0
i2u(n) = (^n << 1) | 1          when n < 0
```

This is the same zigzag-like transform used for compact signed representation.

## Type identifiers

Every top-level node and every dynamic interface value begins with a `u3` type
ID. Nested fields and elements inherit the type of their container and do not
carry an additional type ID of their own.

The type ID is resolved through a `TypeRegistry`. The decoder must have an
identical registry or otherwise the same type/function mappings used by the
encoder. Function values are encoded with a second `u3` function ID in the
payload.

## Value encoding

### Nil and bools

| Value | Bytes |
| --- | --- |
| `nil` | `0x10` (`meta_nil`) |
| `false` | `0x01` (`meta_fls`) |
| `true` | `0x03` (`meta_tru`) |

### Integers

`uint8` and `int8` use a single byte without an extra length prefix. The signed
variant is stored using `i2u`.

The remaining integer kinds use the compact integer format:

| Go kind | Encoding |
| --- | --- |
| `uint16`, `int16` | `u2`, `i2` |
| `uint32`, `int32` | `u3`, `i3` |
| `uint64`, `int64` | `u4`, `i4` |
| `uint`, `int` | same as `uint64`, `int64` |
| `uintptr` | same as `uint64` |

`uintptr` and `unsafe.Pointer` are serialized as numeric values, not portable
pointer identities.

### Floating-point and complex values

`float32` is encoded as its IEEE-754 bit pattern, byte-swapped so the codec can
rebuild the value deterministically, and then written as a compact `u3` value.
`float64` is encoded the same way with `u4`.

`complex64` is the concatenation of the encoded real and imaginary parts.
`complex128` uses the same rule with `float64` parts.

### Strings

A string without shared backing storage is encoded as a length followed by its
bytes:

```text
u4(length) string_bytes
```

When a string shares shared storage with another string or slice, it is encoded
with a `meta_ref` marker in the node stream and its parent/bounds are recorded
in the shared-slices section. Repeated strings can also be represented through
the references section.

### Functions and channels

A nil function or channel is encoded as `meta_nil`.

A non-nil function is encoded as:

```text
meta_nonil function_id
```

where `function_id` is a `u3` taken from the registry. The function value itself
is not serialized; the decoder resolves it from the registry.

A non-nil channel is encoded as:

```text
meta_nonil capacity
```

where `capacity` is a `u4`. Channel contents are not serialized. The decoder
recreates a new channel with the same capacity and direction.

### Arrays and slices

An array has no header; its elements are encoded in index order.

A nil slice is encoded as `meta_nil`.

A non-nil slice is encoded as:

```text
meta_nonil length capacity element_values...
```

`length` and `capacity` are `u4` values. Zero-capacity slices still store the
length and capacity, but the element payload is omitted. A slice or string that
shares backing storage is represented by a `meta_ref` and is described in the
shared-slices section.

### Maps

A nil map is encoded as `meta_nil`. A non-nil map is encoded as:

```text
meta_nonil entry_count key value key value ...
```

`entry_count` is a `u4`. Keys and values are written in sequence; map iteration
order is not guaranteed and may vary across Go runs.

### Structs

A struct is written as:

```text
meta_strc field_values...
```

If at least one struct field has a `codec:"id=N"` tag, the serializer switches
into tagged mode:

```text
meta_strc | meta_tags tagged_field_count
field_id field_value ...
```

`tagged_field_count` and each `field_id` are `u4` values. Only tagged fields
that are not marked `deprecated` are written. Untagged fields are omitted in
that mode. Missing fields retain their zero values during decoding.

Without tags, fields are written in declaration order.

### Interfaces

An interface writes the complete dynamic value node:

```text
dynamic_type_id dynamic_value
```

The dynamic type ID controls how the value is decoded; the static interface type
itself is not used for that decision.

### Pointers

A nil pointer is encoded as `meta_nil`.

A non-nil pointer is encoded as:

```text
meta_nonil
```

The target value is stored as another node. The pointer relationship is restored
later using the pointers section.

### Custom serializable values

If a type implements `Serializable`, the normal recursive traversal is replaced by
calling its `Serialize()` method. The resulting byte slice is encoded as:

```text
meta_nonil body_length body_bytes
```

or as `meta_nil` for nil values. The decoder invokes `Unserialize([]byte)` and
assigns the returned value if it is compatible with the expected Go type.

## References, pointers, and shared slices

The byte `meta_ref` (`0x00`) marks a value whose actual payload is represented
elsewhere. It is not followed by an inline ID. Instead, the references section
contains the mapping from target IDs to their alias IDs, and the shared-slices
section contains slice and string expressions.

This means that repeated values, forward references, and slice expressions are
resolved after the node stream has been decoded, not while it is being read.

## Compatibility notes

- The low four bits of the first byte contain the format version.
- Type and function IDs are registry-local and are meaningful only with the
  compatible registry used by the decoder.
- `unsafe.Pointer` and `uintptr` values are serialized as raw numeric addresses,
  not as portable pointer identities.
- A non-nil channel is restored as a new channel with the same capacity and
  direction; buffered values are not serialized.
- Function values are restored only if the function value is registered in the
  decoder registry.
- The format is versioned but not self-contained; a decoder still needs the
  matching Go types and registered functions.

