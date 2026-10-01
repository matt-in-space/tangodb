## Context

Records in memory are `core.Entity` (`map[string]any`). Their values are only ever `int64`, `float64`, `string`, `bool`, or a nested `Entity`, and a stored record never holds `nil`, because "no value" is an absent key at every depth (insert strips nulls, and merge deletes keys for nulls). The validator guarantees types match the schema before anything is stored. `docs/PERSISTENCE.md` puts the record layer between the query layer and the pages, so pages only ever see opaque bytes.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- A byte format we design and control, readable in a hex dump.
- Encode and decode with exact round trips, and one canonical encoding per record.
- A decoder that never panics on bad input.

**Non-Goals:**
- Pages, files, the catalog, or wiring into `Database`.
- Decoding a single field without the rest (the format allows it later; see decision 9).
- Lists and references (their tags are reserved).
- Field ids or positional encoding (the version byte leaves room to switch later).

## Decisions

1. **Package and types.** A new package `storage/record`, with no dependency on `core`:
   ```go
   type Entity map[string]any

   func Encode(e Entity) ([]byte, error)
   func Decode(b []byte) (Entity, error)
   ```
   `core` changes `type Entity map[string]any` to `type Entity = record.Entity`, a type alias. The alias matters: a nested object inside a record is stored as an `Entity` behind `any`, and a type switch in another package only matches the exact named type. With an alias, `core.Entity` and `record.Entity` are the same type, so `case Entity:` matches in both packages. No other `core` code changes.

2. **Record layout.**
   ```
     version   1 byte   0x01
     kind      1 byte   0x00 record   (0x01 forwarding stub, 0x02 overflow: reserved, not yet produced)
     body      an object
   ```
   `Decode` rejects any version other than `0x01`, and any kind other than `0x00` for now.

3. **Object layout:** a varint field count, then each field, sorted by name (byte order):
   ```
     name length   1 byte (1-255)
     name          UTF-8 bytes
     value         tag + payload
   ```

4. **Value tags:**
   ```
     0x01 int      zigzag varint
     0x02 float    8 bytes: IEEE 754 bits (math.Float64bits), little-endian
     0x03 text     uvarint byte length + UTF-8 bytes
     0x04 false    no payload
     0x05 true     no payload
     0x06 object   uvarint byte length of the object that follows, then the object
     0x07 list     reserved
     0x08 ref      reserved
   ```
   - There is no null tag. `Encode` rejects a `nil` value (it can't come from validated data) instead of silently dropping it.
   - The object byte length covers the object's field count and fields, so a reader can skip it in one step.

5. **Varints are written by hand.**
   - An unsigned varint (`uvarint`) stores 7 bits per byte, least significant group first, with the top bit set on every byte except the last: 42 is `2A`, and 300 is `AC 02`.
   - Zigzag maps a signed number to unsigned, `(n << 1) ^ (n >> 63)`, so small negatives stay small: 0→0, -1→1, 1→2, -2→3.
   - Ints are zigzag uvarints; counts and lengths are plain uvarints.
   - Tests check them against `encoding/binary`'s `PutVarint` / `PutUvarint` to prove they're right, but the package doesn't use them.

6. **Canonical encoding.** Sorted names plus minimal varints mean every record has exactly one encoding, so `Encode(Decode(b)) == b` for any `b` that decodes. `Decode` rejects a non-minimal varint (for example, 42 written as `AA 00`), out-of-order field names, and duplicate names.

7. **Defensive decoding.** `Decode` returns a `*DecodeError{Offset int; Reason string}`, formatted as `record: <reason> at byte <offset>`, for:
   - running out of bytes anywhere (every read is bounds-checked first)
   - an unknown version, kind, or tag
   - a length larger than the bytes remaining (so a corrupt length can't make it allocate gigabytes)
   - a name length of 0
   - names or text that aren't valid UTF-8
   - NaN or infinite floats (the language can't produce them, so they mean corruption; `-0.0` is fine)
   - a varint that's non-minimal or longer than 10 bytes
   - out-of-order or duplicate field names
   - an object whose byte length doesn't match what its fields consumed
   - nesting deeper than 32 levels
   - trailing bytes after the record

8. **Limits that keep every valid record encodable**, enforced when a collection is defined:
   - field names are at most 255 bytes (one length byte): `field name "..." is longer than 255 bytes`
   - blocks nest at most 32 levels: `field "<path>" is nested more than 32 levels deep`

   `Encode` checks the same limits as a backstop, for records built directly in Go.

9. **Partial decoding is possible later.** Sorted names and length-prefixed objects let a future `DecodeField(b, "address.city")` skip everything else. That's not built here.

## Risks / Trade-offs

- [Field names repeat in every record] → Accepted for debuggability. The version byte lets a field-id format replace it later.
- [The `Entity` alias moves the type's home into `storage/record`] → It's a one-line change with no effect on `core`'s code. The record package stays independent, with no import cycle.
- [Fixed tag numbers] → `0x07` and `0x08` are reserved for lists and references, so they slot in without renumbering.
