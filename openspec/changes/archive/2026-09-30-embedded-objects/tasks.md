## 1. Schema and definition

- [x] 1.1 Add `TypeObject` and the recursive `Schema` type; `objects` on `Collection` and `Objects` on `DefineCollectionOperation`; `rootSchema()`
- [x] 1.2 Parser: a `{...}` block in type position, parsed recursively; only `@optional` after a block; `@id`/`@auto` inside a block rejected with the full path
- [x] 1.3 `defineCollection`: the same annotation rules for operations built in Go
- [x] 1.4 `Collection.String()`: nested, indented blocks with annotations after `}`
- [x] 1.5 Tests: blocks, nested blocks, optional blocks, `@id`/`@auto` inside rejected, schema output

## 2. Nested values

- [x] 2.1 `parseValue`: `{` parses a nested record literal
- [x] 2.2 Validation recurses with a path prefix; `expected object` / `got object` shape errors; `valueTypeName` names objects
- [x] 2.3 Insert: null-stripping recurses
- [x] 2.4 Tests for every scenario in the `schema-enforcement` delta, plus a nested literal parse test and a batch containing nested records

## 3. Projections

- [x] 3.1 `resolveProjection` replaces `expandProjection` and the duplicated projection checks in read, delete, merge, and insert
- [x] 3.2 `resolvePath` for table cells
- [x] 3.3 Tests for every projection scenario in the `query-syntax` delta, including an insert `=> {*}` with a nested record

## 4. Guard unsupported uses

- [x] 4.1 Filters (read, delete, merge) reject object fields and dotted paths; merge payloads reject object fields
- [x] 4.2 Tests for the filter and merge scenarios, including that nothing is deleted or changed

## 5. Docs and verification

- [x] 5.1 README: an embedded-objects section (declaring, inserting, validation errors, dotted columns, what's not supported yet); Status line; `QUERY_LANGUAGE.md`: note which parts of nesting are implemented, and that `@id`/`@auto` inside blocks may return with `@collection`
- [x] 5.2 Run `go vet ./...` and `go test ./...`
