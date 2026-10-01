## 1. Package and varints

- [x] 1.1 Create `storage/record` with `type Entity map[string]any`; change `core` to `type Entity = record.Entity`
- [x] 1.2 Hand-written uvarint and zigzag varint encode/decode, including rejecting non-minimal and over-long varints
- [x] 1.3 Varint tests, checked against `encoding/binary`'s `PutUvarint` / `PutVarint` for many values (0, 1, 127, 128, 300, max, negatives)

## 2. Encoding

- [x] 2.1 `Encode`: version and kind bytes, sorted fields, all value tags, length-prefixed nested objects; reject `nil` values, names that are empty or over 255 bytes, and nesting over 32 levels
- [x] 2.2 Golden tests for every encoding scenario in the `record-encoding` delta (exact bytes), and that field order doesn't change the output

## 3. Decoding

- [x] 3.1 `Decode` with `DecodeError{Offset, Reason}`, bounds-checking every read and rejecting every malformed case in the design
- [x] 3.2 Round-trip tests (`Decode(Encode(e))` and `Encode(Decode(b))`), one test per malformed case, and the delta's decoding scenarios
- [x] 3.3 A fuzz test for `Decode` (`go test -fuzz`), seeded with the golden encodings

## 4. Definition limits

- [x] 4.1 Reject field names over 255 bytes and nesting over 32 levels when parsing a definition and in `defineCollection`
- [x] 4.2 Tests for both scenarios in the `collection-definition` delta

## 5. Docs and verification

- [x] 5.1 `docs/PERSISTENCE.md` section 2: mark the format decided and describe it (layout, tags, varints, canonical form, defensive decoding, limits)
- [x] 5.2 Run `go vet ./...` and `go test ./...`
