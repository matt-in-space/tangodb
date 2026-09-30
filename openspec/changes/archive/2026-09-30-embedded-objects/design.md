## Context

`Collection` holds its fields as `data map[string]DataType` plus `optional map[string]bool`, and `DefineCollectionOperation` mirrors them (`Data`, `Optional`). Records are `Entity` (`map[string]any`) with scalar values. `recordProblems` / `fieldProblem` validate a record against `collection.data`, reporting problems as `field "<name>": ...`. Projections are expanded by `expandProjection` (`*` → sorted schema fields) and checked against `collection.data` in a loop duplicated in read, delete, merge, and insert. `renderTable` reads `record[field]` per column. `recordMatchesFilter` compares with `!=`, which would panic on two maps.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Embedded blocks in definitions and inserts, validated recursively.
- Objects visible through flattened, dotted projections.
- Object fields in filters and merge payloads fail clearly instead of misbehaving.

**Non-Goals:**
- Filtering on nested values (dotted paths or subset object filters), deep merge, or replacing objects via merge.
- `@collection` and one-to-many (`[]`).
- `@id` / `@auto` inside blocks.

## Decisions

1. **A recursive `Schema` type for nested blocks.** `type Schema struct { Data map[string]DataType; Optional map[string]bool; Objects map[string]*Schema }`, with a new `TypeObject` in `Data` for block fields and its shape in `Objects`. The top level stays as it is on `Collection` (`data`, `optional`) and gains `objects map[string]*Schema`. `DefineCollectionOperation` gains `Objects map[string]*Schema`. This keeps the many existing test literals unchanged. A small `collection.rootSchema()` returns a `*Schema` view of the top level for code that walks recursively.

2. **Parsing blocks.** In `parseDefineCollection`, a `{` in type position parses a nested field block with the same loop, recursively. After a block's closing `}`, only `@optional` is accepted. Inside a block, `@id` and `@auto` fail with `@id is not allowed inside an embedded block (field "address.id")` (likewise for `@auto`), naming the full path. `defineCollection` re-checks the same rules for operations built in Go, as it does for the other annotations.

3a. **Dotted names lex as one identifier.** (Added during implementation, since dotted projections and filter keys can't otherwise be written.) In the lexer, a `.` followed by a name character continues an identifier, so `address.city` is one `tokenIdent`. Numbers are unaffected (`9.99` starts with a digit). Where a name is *declared or written*, meaning collection names, schema field names, and record-literal field names, a dotted name is rejected: `field name "address.city" cannot contain "."`.

3. **Nested record literals.** `parseValue` treats `{` as a nested `parseRecordLiteral` and returns the `Entity`. Commas, `null`, booleans, and bracket balancing all apply unchanged inside.

4. **Validation recurses with a path prefix.** `fieldProblem` and the per-record checks take a `*Schema` and a path prefix (`""` at the top level, `"address."` inside). For a `TypeObject` field:
   - a `null` or absent value follows the optional/required rule
   - a non-object value fails with `field "address": expected object, got int`
   - an object recurses: undeclared nested fields, nested types, and missing nested required fields are all reported with full paths, e.g. `field "address.zip" is required for collection "user"`

   An object value on a scalar field fails with `expected text, got object`. `valueTypeName` names an `Entity` `object`. Top-level-only checks (primary key, `@auto`) stay at the top level. The "one problem per field" rule applies per full path.

5. **Null-stripping recurses**, so nested `null`s are stored as absent keys and "no value" has one representation at every depth.

6. **One projection helper for all operations.** A new `resolveProjection(collection, projection) ([]string, error)` replaces `expandProjection` and the four duplicated validation loops:
   - `*` → every leaf path in the schema, sorted as a whole
   - an object field name → its leaf paths
   - a dotted path to a leaf → itself
   - anything else → `field "x" not found in schema for collection "user"`

   Duplicate columns are removed while keeping the order they were given in. The helper is error-returning, so insert can add its error to its problem list.

7. **`resolvePath(record, path)`** walks dotted segments through nested `Entity` values and returns `nil` when any step is missing, so a leaf of an absent optional object renders as `null`. `renderTable` uses it for every cell.

8. **Filters and merge payloads reject object fields for now.** Filter validation (read, delete, merge) rejects a dotted key or a key naming an object field: `filtering on embedded object field "address" is not supported yet`. Merge payload validation rejects an object field: `merge payload cannot set embedded object field "address" yet`. Both run before any comparison, so the map-comparison panic can't happen.

9. **Schema output.** `Collection.String()` renders a block field as `address: {` followed by its fields indented one more level, then `}` and its annotations (`} @optional`), in the declaration syntax.

## Risks / Trade-offs

- [A wide object makes a wide `{*}` table] → Accepted. Dotted projections let you pick columns.
- [Top-level and nested fields use slightly different representations (`Collection` fields vs `*Schema`)] → Kept so existing tests don't churn. `rootSchema()` bridges them. Worth folding together later, perhaps when `@collection` lands.
- [Objects can be inserted but not filtered on yet] → Deliberate staging. The filter restriction is explicit, and that's the next change.
