# Serialization and Deserialization Algorithm

This document describes the algorithm implemented by `codec`: how Go values
are traversed, converted into a graph of nodes and a binary stream, and then
restored, including cycles, shared references, and slices that share a backing
array. It documents the current version 1 implementation; it is not a promise
that arbitrary program versions are wire-compatible. Byte-level layout details
are also documented in [FORMAT.md](./FORMAT.md).

## Contents

- [1. Core idea](#1-core-idea)
- [2. Participants and state](#2-participants-and-state)
- [3. Building the graph for serialization](#3-building-the-graph-for-serialization)
  - [3.1. Traversing values](#31-traversing-values)
  - [3.2. Tracking references and cycles](#32-tracking-references-and-cycles)
  - [3.3. Merging slices](#33-merging-slices)
  - [3.4. Renumbering nodes](#34-renumbering-nodes)
- [4. Writing the binary stream](#4-writing-the-binary-stream)
  - [4.1. Nodes and type identifiers](#41-nodes-and-type-identifiers)
  - [4.2. Primitive and special values](#42-primitive-and-special-values)
  - [4.3. Composite values](#43-composite-values)
  - [4.4. Struct tags](#44-struct-tags)
  - [4.5. User-serializable types](#45-user-serializable-types)
  - [4.6. Pointer links](#46-pointer-links)
- [5. Deserialization](#5-deserialization)
  - [5.1. Reading the main stream](#51-reading-the-main-stream)
  - [5.2. Deferred map entries](#52-deferred-map-entries)
  - [5.3. Restoration phases](#53-restoration-phases)
- [6. Type and function registry](#6-type-and-function-registry)
- [7. Reusing codec instances](#7-reusing-codec-instances)
- [8. Limitations and important properties](#8-limitations-and-important-properties)

## 1. Core idea

The serializer does not write an object directly to the stream while recursively
walking the input. It first builds an internal representation of the reachable
graph:

```text
Go value -> nodes and links -> renumbered graph -> binary stream
```

This preliminary pass is needed to discover repeated addresses, cycles,
pointers to previously encountered values, and slices that overlap the same
backing array. After traversal, node values are written in graph order. Simple
references, forward references, and pointer links may refer to a node whose
value is restored later.

Deserialization reverses this process in multiple steps:

```text
binary stream -> values and temporary containers -> restored links
```

Values are created first and unresolved relationships are recorded. Separate
phases then restore slice expressions, pointers, references, and map entries.

## 2. Participants and state

`Serializer` stores graph values in `values` and maintains several indexes:

- `childs` records child nodes and drives traversal and encoding;
- `parents` associates a node with its parent container;
- `addresses` detects repeated addressable values;
- `ptrValues` and `ptrs` associate target values with pointers to them;
- `refs` stores detected repeated references;
- `slices` (`reflex.SliceMap`) groups compatible strings, arrays, and slices
  that share memory;
- `imap` maps internal graph IDs to the IDs used in the stream.

Graph IDs assigned during traversal must not be confused with stream IDs.
Before writing, a traversal of `childs` fills `imap`; serialized references and
pointer records use this mapping.

`Unserializer` stores the input stream and read position, a `values` map of
restored values indexed by stream ID, and tables of pending work:

- `slices` contains expressions to apply to parent values;
- `refs` contains locations waiting for the value at a node ID;
- `ptrs` contains pointer locations grouped by their target node ID;
- `maps` contains key/value pairs to insert after references have been fixed.

## 3. Building the graph for serialization

### 3.1. Traversing values

`Encode` starts by traversing the root value through `visit`. A node is created
for each value. Further traversal depends on `reflect.Kind`:

- structs are traversed by field;
- arrays and slices are traversed by element;
- maps are traversed by key and value using `MapRange`;
- interfaces are unwrapped to their dynamic values;
- non-nil pointers are associated with the node for their targets;
- strings are registered in `SliceMap` to detect shared storage;
- primitives remain leaf nodes.

For an ordinary struct, child fields are added in declaration order. If the
registry has `codec` tags for the type, traversal adds only tagged fields that
are not marked `deprecated`. A field is represented by a container node and a
separate child value node; this makes field addressability explicit and
supports pointers to fields.

An interface preserves its dynamic type: its child value is written as a full
node with its own type ID. The interface's static declaration does not determine
the type of the contained value.

### 3.2. Tracking references and cycles

For reference-like values, the serializer uses addresses together with types
so that different values are not conflated merely because they have the same
numeric address. When a known value is encountered again, its contents need
not be written a second time; the graph records a link to its existing node.

Pointers are handled separately from ordinary references. For a non-nil
pointer, the serializer registers the address and type of the pointed-to value.
If the target has already been seen, the pointer is associated with that
target; otherwise, the target is added as a node reachable from the graph's
top level. This breaks recursion for cycles such as `node.Next = node` without
discarding the relationship.

Reference markers in node payloads are `meta_ref` bytes with no inline ID.
Before the node stream, the references section maps each target ID to the IDs
of locations that alias it. The decoder records these mappings before reading
nodes, so targets may occur later in the stream.

Map iteration order is controlled by Go and is not guaranteed to be stable
between serializations. This does not change the semantics of the restored map,
but its byte representation may differ.

### 3.3. Merging slices

`SliceMap` describes memory ranges for registered strings, arrays, and slices
with the same element type. For ranges whose element size is non-zero, it
determines whether they are identical, parent/child, or partially overlapping.
If ranges overlap but neither contains the other, a virtual common parent is
created to cover the required length and capacity.

After the main traversal, `visitSlices`:

1. visits virtual parents so their data is also represented in the graph;
2. moves child element values into the shared parent's nodes;
3. replaces separate nested-slice representations with references/expressions
   to the parent node where that matches the shared storage.

Slice expressions are not written inline. The header's shared-slices section
contains the parent ID and, for each expression, the destination ID and
`start/end/max` bounds. The decoder applies them after reading nodes to restore
both the data and the shared-backing-array relationship.

For zero-sized elements, `SliceMap` does not infer shared backing storage from
addresses: Go may use one address for many such elements. Slice length and
capacity are still preserved. Slices/strings with different element types,
created for example through `unsafe.Pointer`, are not merged into one common
parent.

### 3.4. Renumbering nodes

After slice processing, `renumberGraph` traverses child nodes from the virtual
root and assigns sequential IDs. The IDs are implicit positions in the
traversed graph and are used by the reference, pointer, and shared-slices
header sections. Nodes required by pointers or virtual parents are written as
top-level graph nodes.

## 4. Writing the binary stream

The stream begins with a one-byte header, followed by optional metadata
sections and then the node stream:

```text
header [references_section] [pointers_section] [slices_section] encoded_nodes
```

The low four bits of the header contain the format version (`1`). Bits `0x10`,
`0x20`, and `0x40` indicate the presence of references, pointers, and
shared-slices sections, respectively; when present, sections appear in that
order. Each section uses `u4` counts and IDs. Each top-level node starts with a
`u3` type ID, followed by its value payload. `uN` denotes an integer with an
`N`-bit length prefix; integers use the minimum number of bytes. Signed values
are transformed with `i2u` and then encoded as unsigned. Multi-byte payloads
use big-endian order.

### 4.1. Nodes and type identifiers

An ordinary node is:

```text
u3(type_id) value_payload
```

The type ID is resolved through `TypeRegistry`; the stream does not contain a
Go type description. The order of `childs` determines the order of nested
payloads. For values that are direct struct fields, array/slice elements, or
map entries, the type comes from the declared container. Top-level nodes and
dynamic interface values carry a type ID.

### 4.2. Primitive and special values

- `nil`, `false`, and `true` have dedicated markers.
- `uint8`/`int8` are written as one byte; other integers use compact `u2`/`u3`/
  `u4` encodings according to the type width.
- `int`, `uint`, and `uintptr` use the 64-bit-width encoding. `uintptr`
  preserves a numeric value but does not make an address portable.
- `float32` and `float64` encode their IEEE-754 representations; `complex`
  stores the real and imaginary parts consecutively.
- A string without shared backing storage is written as a `u4` byte length and
  its bytes. A string represented over shared storage has a `meta_ref` marker;
  its parent and bounds are in the shared-slices section.
- Nil and non-nil empty values are distinct. Slice length and capacity are
  stored separately.
- For channels, nilness, type direction, and capacity are represented; channel
  contents are not serialized.
- A non-nil function is represented by a registered function ID, not by machine
  code or a closure.
- `unsafe.Pointer` is written as a numeric address and is not restored as a
  safe, portable pointer.

Exact markers and payload layouts are documented in [FORMAT.md](./FORMAT.md).

### 4.3. Composite values

- **Array:** elements are written by index; there is no separate array header.
- **Slice:** marker, length, capacity, and element values. A zero-capacity slice
  is encoded without traversing backing storage.
- **Shared slice expression:** a `meta_ref` marker in the value stream; the
  parent ID, destination ID, and `start/end/max` indexes are in the header's
  shared-slices section.
- **Map:** nil marker or non-nil marker, entry count, and key/value sequence.
  Pair order is not guaranteed.
- **Struct:** struct marker and fields in declaration order when untagged.
- **Interface:** complete dynamic node, including dynamic type ID.
- **Pointer:** nil/non-nil marker in the node stream; links to target values
  are recorded in the header's pointers section.

A repeated object may be encoded as `meta_ref` instead of repeating its
payload. Its target and alias IDs are recorded in the header's references
section, not after the marker. Therefore, the presence of a nested value does
not necessarily imply that its contents are recursively written again.

### 4.4. Struct tags

If a struct type has registered `codec` tags, encoding switches to tagged mode.
Each field ID must be unique and non-negative; duplicate IDs cause tag parsing
to fail when the type is registered. The stream contains the number of
serializable tagged fields, followed by `field_id, field_value` pairs.

In tagged mode, the serializer includes only tagged fields not marked
`deprecated`. Untagged fields are omitted. The deserializer looks up each field
by ID in registry metadata; fields missing from the stream retain their zero
values. A tagged deprecated field remains in the metadata and can be read from
an older stream, but is not written by the current serializer. An unknown field
ID causes a decoding error.

Without active tags, structs are encoded positionally. Therefore, changes to
field order or composition are incompatible with previously written positional
data.

### 4.5. User-serializable types

When a type implements `Serializable`, normal traversal of its internals is
replaced by a call to `Serialize()`. The returned byte slice is written with a
non-nil marker and a length. The decoder passes the payload to
`Unserialize([]byte)`, checks for an error or missing result, and assigns the
returned value if its type is convertible to the expected type.

This payload is opaque to the graph algorithm: references inside the custom
byte slice are not automatically discovered or restored. The custom format
must include all required data and validate its input.

### 4.6. Pointer links

Before the node stream, the serializer writes a pointers section when at least
one pointer link exists. It contains records of the form:

```text
target_id pointer_count pointer_id ...
```

The section begins with the record count; each record's first ID identifies
the target value, followed by the count and IDs of pointer values that refer to
it. All IDs use `u4`. Deferred restoration supports cycles and pointers to
fields/elements even when their nodes were discovered elsewhere in the graph.

## 5. Deserialization

### 5.1. Reading the main stream

`Decode` first checks that the input is non-empty, clears state from a previous
call, and verifies the version in the header's low four bits. It reads the
optional references, pointers, and shared-slices sections in flag order, then
decodes nodes until the end of the input. There is no pointer-link tail or
`meta_ref` delimiter.

For each node, the decoder:

1. reads its type ID and resolves it through `TypeRegistry`;
2. creates a zero value of that type;
3. appends value/container slots to `values`, aligned with stream identifiers;
4. reads the payload according to the type;
5. stores forward references and slice expressions in pending-work tables.

Slices are allocated with `reflect.MakeSlice` using the recorded length and
capacity; structs, arrays, and interfaces are filled recursively. An interface
decodes its dynamic node and stores the resulting value in the interface.

Reading uses reflection and compact-integer helpers. Truncated or malformed
data may panic inside decoding code; `Decode` recovers the panic and returns it
as an `error`. This does not constitute complete safety validation for
arbitrary untrusted streams: resource and size limits must be considered
separately.

### 5.2. Deferred map entries

When decoding a map, the decoder first creates it with the expected entry
capacity. If neither the key nor the value type is a pointer or interface, the
pair is inserted immediately with `SetMapIndex`. If either side is a pointer or
interface, the key and value IDs are saved in `maps`, and insertion is deferred
until links have been restored. This allows maps to contain values that refer
to other nodes or participate in cycles.

### 5.3. Restoration phases

After the node stream, restoration happens in this order:

1. **Slice expressions (`restoreSlices`).** Parent values are sliced with
   `Slice`/`Slice3`; strings and `[]byte` can share byte storage through an
   unsafe representation where supported. The result is assigned to the saved
   destination slot.
2. **Pointers (`restorePointers`).** Pointer locations from the header section
   are assigned to their target values.
3. **Ordinary references (`restoreReferences`).** Every location waiting for a
   node ID receives the final value from `values`.
4. **Maps (`restoreMaps`).** Deferred key/value pairs are inserted into their
   already-created maps.

The order matters: by the time map insertion occurs, references and pointers
must refer to restored values, and slice expressions must already have been
applied to their parents.

## 6. Type and function registry

Type IDs and function IDs are meaningful only within a particular
`TypeRegistry`. The serializer obtains IDs from type/function names; the
decoder uses those IDs to find the corresponding type or registered function
value. Data exchanged between processes therefore requires compatible
registries and consistent ID assignments. Automatic registration is
convenient, but the order in which types are first encountered can differ
between execution paths and runs.

For a stable protocol, register all used types/functions in advance and pass
the compatible registry to both the serializer and deserializer. Recursive
type registration also registers field and element types, but does not by
itself register arbitrary function values; a function value must be registered
explicitly.

## 7. Reusing codec instances

`Serializer.Encode` clears its indexes after writing. The `values` buffer is
zeroed and retains its capacity, allowing subsequent calls to reuse its backing
storage without keeping references to previously serialized objects. Other
internal indexes are also cleared for the next graph. `Serializer` and
`Unserializer` instances contain mutable state; concurrent calls on the same
instance should not be considered thread-safe.

`Unserializer.Decode` clears temporary link tables and sets the current input
stream again before reading the next value.

## 8. Limitations and important properties

- The format is versioned but not self-contained: decoding requires a
  compatible registry for type/function IDs.
- A non-nil channel is restored as a new, empty channel with the same capacity
  and direction; values sent through it are not preserved.
- A function is restored only if its value is registered; closure state is not
  serialized by this mechanism.
- Numeric `uintptr` and `unsafe.Pointer` values must not be treated as portable
  or safely restored addresses.
- Shared storage between strings/slices with different element types, such as
  `[]uint32` and `[]byte` created through `unsafe.Pointer`, is not represented
  as one shared backing array.
- Opaque `Serializable` payloads do not participate in detecting links in the
  surrounding graph.
- Positional struct encoding depends on field order. Tagged mode reduces this
  dependency for fields with IDs, but requires unique IDs and compatible tags
  in both registries.
- Map traversal order is not fixed, so identical logical maps are not
  guaranteed to produce identical bytes across calls.
- Deserialization uses reflection and can allocate memory according to lengths
  and capacities in the stream; untrusted data requires external size and
  provenance controls.
