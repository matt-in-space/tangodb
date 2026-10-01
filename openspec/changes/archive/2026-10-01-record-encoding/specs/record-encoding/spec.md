## Purpose

Defines the byte format TangoDB uses to store a record, and the guarantees for encoding a record into bytes and decoding bytes back into a record.

## ADDED Requirements

### Requirement: Records encode to a self-describing byte format
A record SHALL encode as:
- a format version byte (`0x01`)
- a kind byte (`0x00` for a record)
- an object: a varint field count, then each field sorted by name, each as a 1-byte name length, the UTF-8 name, a 1-byte type tag, and the value's payload

The type tags SHALL be:
- `0x01` int, as a zigzag varint
- `0x02` float, as 8 bytes of IEEE 754 bits, little-endian
- `0x03` text, as a varint byte length then UTF-8 bytes
- `0x04` false and `0x05` true, with no payload
- `0x06` object, as a varint byte length then a nested object

There SHALL be no tag for null: a field with no value is not written.

#### Scenario: A flat record
- **WHEN** the record `{id: 42 name: "Matt"}` is encoded
- **THEN** the bytes are `01 00 02 02 69 64 01 54 04 6E 61 6D 65 03 04 4D 61 74 74` (19 bytes)

#### Scenario: A nested record
- **WHEN** the record `{id: 1 address: {street: "1 Main" city: "MSP"}}` is encoded
- **THEN** the bytes are `01 00 02 07 61 64 64 72 65 73 73 06 1A 02 04 63 69 74 79 03 03 4D 53 50 06 73 74 72 65 65 74 03 06 31 20 4D 61 69 6E 02 69 64 01 02`, with the nested fields stored under their own names inside the length-prefixed `address` object

#### Scenario: An empty nested object
- **WHEN** the record `{address: {}}` is encoded
- **THEN** the bytes are `01 00 01 07 61 64 64 72 65 73 73 06 01 00`, distinguishable from a record with no `address`

#### Scenario: Booleans, negative ints, and floats
- **WHEN** the records `{ok: true}`, `{n: -1}`, and `{p: 1.5}` are encoded
- **THEN** the bytes are `01 00 01 02 6F 6B 05`, `01 00 01 01 6E 01 01`, and `01 00 01 01 70 02 00 00 00 00 00 00 F8 3F`

### Requirement: Encoding is canonical and round-trips
Encoding a record SHALL always produce the same bytes, regardless of the order its fields were given in. Decoding a record's encoding SHALL give back an equal record, and encoding the result of any successful decode SHALL give back the original bytes.

#### Scenario: Field order doesn't matter
- **WHEN** two records with the same fields and values, built in different orders, are encoded
- **THEN** both produce identical bytes

#### Scenario: Round trip
- **WHEN** a valid record containing every value type, including nested objects, is encoded and then decoded
- **THEN** the decoded record equals the original

### Requirement: Decoding rejects malformed input without panicking
Decoding SHALL return an error naming the byte offset of the problem, and SHALL NOT panic, for any input that isn't a valid encoding. That includes:
- input that ends early
- an unknown version, kind, or tag
- a length larger than the bytes remaining, or a zero-length name
- names or text that aren't valid UTF-8
- a NaN or infinite float
- a non-minimal varint
- field names out of order, or repeated
- an object whose stated length doesn't match its contents
- nesting deeper than 32 levels
- bytes left over after the record

#### Scenario: Truncated input
- **WHEN** the first 10 bytes of `{id: 42 name: "Matt"}`'s encoding are decoded
- **THEN** decoding returns an error naming the offset where it ran out of bytes

#### Scenario: Unknown tag
- **WHEN** the bytes `01 00 01 01 78 09` are decoded
- **THEN** decoding returns `record: unknown value tag 0x09 at byte 5`

#### Scenario: Non-minimal varint
- **WHEN** the bytes `01 00 01 02 69 64 01 D4 00` are decoded (42 written in two bytes)
- **THEN** decoding returns an error for the non-minimal varint

#### Scenario: Arbitrary bytes
- **WHEN** random byte strings are decoded
- **THEN** decoding either succeeds or returns an error, and never panics
