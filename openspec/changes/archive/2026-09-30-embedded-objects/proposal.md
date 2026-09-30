## Why

Every field today is a scalar (`int`, `float`, `text`, `bool`). Data with natural structure, like an address or a size, has to be flattened into loose top-level fields by hand. `QUERY_LANGUAGE.md` already designs embedded blocks: a nested field shape with no identity of its own, stored as part of its parent. The validator was built to recurse into them. This change adds them for collection definition and inserts; filtering and merging on them come later.

## What Changes

- A field's type can be an embedded block: `address: { street: text city: text }`. Blocks can nest inside blocks.
- An embedded block can be `@optional` as a whole: the object may be absent or `null`. Fields inside it are required or `@optional` by the usual rule, and required nested fields are only checked when the object is present.
- `@id` and `@auto` are not allowed inside an embedded block, since the block has no identity of its own. (They will likely return with `@collection`, which gives a nested block an identity.)
- Record literals can contain nested record literals: `>> user {id: 1 address: {street: "1 Main" city: "MSP"}};`.
- Nested values are validated recursively, with the same rules as top-level fields. Every problem names the full dotted path: `field "address.city": expected text, got int`. A value of the wrong shape (`address: 5`, or an object on a scalar field) is an error.
- `null`s inside nested objects are stored as absent, the same as at the top level.
- Projections flatten objects into dotted columns: `{*}` lists every leaf path (`address.city`, `address.street`, ...), naming an object field (`=> {address}`) expands to its leaves, and a dotted path (`=> {address.city}`) is its own column. Leaves of an absent object show `null`.
- Schema output prints embedded blocks nested and indented, in the declaration syntax.
- **Not yet supported, and explicit errors rather than wrong results:** filtering on an object field or a dotted path (in read, delete, and merge), and merge payloads that set an object field.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `collection-definition`: embedded block fields, optional blocks, no `@id`/`@auto` inside, nested schema output.
- `schema-enforcement`: recursive validation of nested values with full-path problems; wrong-shape values.
- `query-syntax`: nested record literals as values; projections flatten objects into dotted columns; filters on object fields or dotted paths are rejected for now.
- `merge`: payloads can't set an object field yet.

## Impact

- `core/collection.go`: a nested schema type; `TypeObject`; nested `String()` output.
- `core/parse_define_collection.go`: parse a `{...}` block in type position, recursively, with the annotation rules for blocks.
- `core/parse_insert.go`: `parseValue` accepts a nested record literal.
- `core/validate.go`: validation recurses with a path prefix.
- `core/insert.go`: null-stripping recurses.
- `core/read.go` (shared projection helpers): expanding and validating dotted projections; `resolvePath` for table cells. Read, delete, merge, and insert projections all go through it.
- Filter validation (read, delete, merge) and merge payload validation: explicit "not supported yet" errors for object fields.
- Docs: README gains an embedded-objects section; `QUERY_LANGUAGE.md` notes what's implemented.
