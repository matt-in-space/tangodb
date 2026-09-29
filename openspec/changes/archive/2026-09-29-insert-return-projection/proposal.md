## Why

An insert prints `{map[id:1 name:Matt]}`, which is Go's default formatting leaking into the REPL, and defining a collection wraps its schema in the same stray struct braces (`{user {...}}`). Insert is also the only operation that returns records without being asked; read, delete, and merge all return a bare count unless a `=>` projection opts into records. Bulk inserts (`&`) are coming, and a count is their natural result, so insert should follow the same rule as everything else.

## What Changes

- **BREAKING**: an insert with no `=>` returns a bare-integer count (`1`), not the record.
- An insert can end in `=> {...}` or `=> {*}` to return the inserted record as a table, limited to the projected fields. The record reflects what was stored, including auto-assigned values, so `=> {id}` is how you get a generated id back. This implements the return clause `QUERY_LANGUAGE.md` already describes.
- **BREAKING**: since `=>` can now follow a record literal, a bare insert like `>> user {name: "Matt"}` is no longer known to be complete. The REPL waits for more input until a `;` or `=>` clause ends it, the same as delete and merge today.
- Defining a collection prints its schema without the surrounding struct braces.

## Capabilities

### New Capabilities
- `insert`: the insert statement's result shape (count by default, opt-in records via `=>`) and when it's complete.

### Modified Capabilities
- `collection-definition`: defining a collection returns its schema in the schema syntax, with nothing wrapped around it.

## Impact

- `core/parse_insert.go`: optional `=>` projection after the record literal; EOF after the literal means incomplete.
- `core/insert.go`: `InsertOperation` gains `Projection`; `InsertResult` becomes `{Count, Projection, Records}`, rendered with the shared `renderCountOrTable`.
- `core/define_collection.go`: `DefineCollectionResult.String()`.
- Tests: bare inserts in statements gain `;`, and tests reading `InsertResult.Record` switch to `Records` via a projection.
- Docs: every README insert example, plus `QUERY_LANGUAGE.md`'s insert section and the README Status line.
