## Why

`@auto` is currently tied to `@id`: it can only generate the primary key. But "this is the record's identity" and "the database assigns this value" are separate ideas. A collection keyed by a text code can't also have a generated sequence number. Making `@auto` work on any `int` field keeps each annotation to one job.

## What Changes

- `@auto` can be used on any `int` field, with or without `@id`. The "`@auto` requires `@id`" error is removed. The `int`-only rule stays.
- A collection can have more than one `@auto` field. Each has its own counter, starting at `1`, and values are never reused.
- Every `@auto` field is owned by the database:
  - insert: supplying any `@auto` field, including as `null`, is an error
  - merge: a payload setting any `@auto` field is an error (new; today only the primary key is protected)
- Because nothing else can write them, `@auto` values are unique by construction, without an index.
- `@auto @optional` is an error, since the field always has a value.
- A failed insert never consumes a counter value. Values are assigned only after every check passes.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `collection-definition` (introduced by `optional-fields`, which must be archived first): `@auto` on any `int` field, independent counters, and the `@auto @optional` conflict.
- `merge`: payloads must not set an `@auto` field.

## Impact

- `core/parse_define_collection.go`, `core/define_collection.go`: drop the `@id` requirement; `AutoIncrement bool` becomes a set of auto field names.
- `core/collection.go`: one counter per auto field instead of a single `nextAutoValue`; `String()` prints `@auto` on any field.
- `core/insert.go`, `core/merge.go`: supplied/payload checks cover every auto field; counters are assigned last.
- Tests that set `AutoIncrement: true` or expect the "`@auto` requires `@id`" error.
- Docs: `README.md` and `QUERY_LANGUAGE.md` describe `@auto` as independent of `@id`.
