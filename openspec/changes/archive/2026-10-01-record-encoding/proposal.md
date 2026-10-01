## Why

The first step toward persistence is being able to turn a record into bytes and back again, in a format we design and control byte by byte, rather than through Go's built-in marshalling. It's the one storage piece with no dependencies (no pages, files, or indexes), so it can be built and tested on its own. Everything above it keeps working with `Entity`, and everything below it only ever sees opaque bytes.

## What Changes

- A new `storage/record` package that **encodes** an `Entity` into bytes and **decodes** bytes back into an `Entity`.
- The format is self-describing: every record stores its field names and a type tag for every value, so it decodes without the schema and is readable in a hex dump.
  - a format version byte and a record kind byte at the front of every record
  - fields sorted by name, so a record always encodes to the same bytes
  - nested objects stored nested (each field under its own name, inside its parent object, which is length-prefixed so a reader can skip it)
  - hand-written zigzag varints for integers and lengths, so small numbers take one byte
  - IEEE 754 floats, length-prefixed UTF-8 text, and booleans as two payload-free tags
  - no null tag, since "no value" is an absent field
- Decoding is defensive. Truncated or corrupt input, unknown versions, kinds, or tags, invalid UTF-8, NaN or infinite floats, non-minimal varints, out-of-order or duplicate field names, trailing bytes, and nesting deeper than the limit all return an error naming the byte position. None of them panics.
- Every valid record round-trips: `Decode(Encode(e))` equals `e`, and `Encode(Decode(b))` equals `b`.
- Collection definitions gain two limits, so any valid record can be encoded: field names are at most 255 bytes, and blocks nest at most 32 levels deep.
- `core.Entity` becomes a type alias of `record.Entity`, so nested values are the same type in both packages.
- **Not in this change:** pages, files, the catalog, or wiring storage into `Database`. Records still live in memory.

## Capabilities

### New Capabilities
- `record-encoding`: the on-disk byte format for a record, and the guarantees for encoding and decoding it.

### Modified Capabilities
- `collection-definition`: limits on field name length and nesting depth.

## Impact

- New package `storage/record`: `Entity`, `Encode`, `Decode`, the varint helpers, and a decode error type carrying a byte offset.
- `core/collection.go`: `type Entity = record.Entity`.
- `core/parse_define_collection.go` / `core/define_collection.go`: the two definition limits.
- Tests: golden byte tests, round trips, one test per kind of corruption, and a fuzz test for `Decode`.
- `docs/PERSISTENCE.md`: section 2 updated with the decided format.
