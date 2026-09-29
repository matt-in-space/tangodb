## 1. Null literal

- [x] 1.1 Add `null` → `nil` to `parseValue`'s `tokenIdent` case in `core/parse_insert.go`
- [x] 1.2 Parser tests: `null` in a record literal, a filter, and a merge payload; `"null"` stays a string

## 2. @optional annotation

- [x] 2.1 Parse `@optional` in `core/parse_define_collection.go` (reject duplicates, reject with `@id`) and carry it as `Optional map[string]bool` on `DefineCollectionOperation`
- [x] 2.2 Store it as `optional map[string]bool` on `Collection`; `defineCollection` re-checks the `@id` conflict
- [x] 2.3 `Collection.String()` prints `@optional`
- [x] 2.4 Tests: declaring optional fields, duplicate `@optional`, `@id @optional`, `@id @auto @optional`, schema output

## 3. Validation

- [x] 3.1 `validateFields`: `nil` is valid for optional fields; for required fields it fails with `field %q is required and cannot be null`
- [x] 3.2 Add `validateRequired(collection, record)` for insert: missing required fields fail in sorted order, skipping the `@id @auto` field
- [x] 3.3 Unit tests in `core/validate_test.go`

## 4. Wire into operations

- [x] 4.1 `db.insert`: auto-supplied check before `validateFields`, then `validateFields`, the existing primary-key check, `validateRequired`; strip `nil` values before storing
- [x] 4.2 `db.merge`: primary-key payload check before payload validation; a `nil` payload value deletes the key from each matched record
- [x] 4.3 Confirm `(field: null)` filters match absent keys in read, delete, and merge
- [x] 4.4 `formatCell(nil)` returns `"null"`
- [x] 4.5 Update existing tests that leave out declared fields: supply the field or mark it `@optional`, whichever the test means
- [x] 4.6 Operation tests for every scenario in the `schema-enforcement` delta spec

## 5. Docs and verification

- [x] 5.1 Update `README.md` and `QUERY_LANGUAGE.md`: required by default, `@optional`, the `null` literal, clearing via merge, `null` filters, `null` in tables; fix examples that leave out declared fields
- [x] 5.2 Run `go vet ./...` and `go test ./...`
