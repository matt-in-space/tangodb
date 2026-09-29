## Why

Every declared field is currently optional by accident: `>> user {id: 1}` is accepted even though `name: text` is declared, and once a field has a value there is no way to clear it. The schema says a field exists but not whether a record must have it. Under the language's explicit-over-implicit principle, a field should be required unless the schema says otherwise, and "no value" should be a single, visible concept.

## What Changes

- Fields are **required by default**. An insert that omits a required field fails.
- A new `@optional` field annotation marks a field that may have no value. `@id` together with `@optional` is an error.
- A new `null` literal means "no value". There is exactly one such concept: leaving an optional field out of an insert and writing `null` for it are the same thing.
- `null` is only valid for optional fields, everywhere a value appears:
  - insert: `{nickname: null}` is the same as omitting `nickname`
  - merge: `{nickname: null}` clears the field's value
  - filter: `(nickname: null)` matches records where `nickname` has no value, with plain equality (no three-valued logic)
  - `null` on a required field is an error in inserts, payloads, and filters
- A field with no value displays as `null` in tables instead of a blank cell.
- A collection's schema output shows `@optional`.

## Capabilities

### New Capabilities
- `collection-definition`: rules for declaring a collection's fields and their annotations, starting with `@optional` and its conflict with `@id`.

### Modified Capabilities
- `schema-enforcement`: the exact-type rule gains its one exception (`null` on an optional field), and new requirements cover required fields, clearing via merge, and filtering on `null`.
- `query-syntax`: adds the `null` literal.

## Impact

- `core/parse_define_collection.go`, `core/define_collection.go`, `core/collection.go`: parse, carry, and print `@optional`.
- `core/parse_insert.go`: `null` in `parseValue`.
- `core/validate.go`: `null` handling and the required-field check.
- `core/insert.go`, `core/merge.go`: required fields on insert, clearing on merge.
- `core/read.go`: `null` in table cells.
- Tests and docs whose records leave out declared fields must either supply them or mark them `@optional`.
- Followed by the `auto-any-int-field` change, which adds `@auto @optional` as a second conflict.
