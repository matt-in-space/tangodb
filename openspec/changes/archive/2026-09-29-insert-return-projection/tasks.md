## 1. Define-collection output

- [x] 1.1 Add `DefineCollectionResult.String()` returning `r.Collection.String()`
- [x] 1.2 Test the exact define output, and update REPL tests that expect the `{...}` wrapper

## 2. Parse the return projection

- [x] 2.1 `parseInsert`: after the record literal, handle `=>` (normalizing nil to `[]string{}`), EOF → `ErrIncompleteInput`, otherwise `expectEndOfStatement`; add `Projection` to `InsertOperation`
- [x] 2.2 Parser tests: no projection with `;`, named projection, `{*}`, `=> {}` kept non-nil, EOF after the literal is incomplete, `{*, id}` is rejected
- [x] 2.3 Add `;` to bare inserts in existing parser tests

## 3. Insert result

- [x] 3.1 `InsertResult` becomes `{Count, Projection, Records}` with `String()` via `renderCountOrTable`
- [x] 3.2 `db.insert`: expand and validate the projection before any write; return the stored record (after null-stripping and auto assignment) when a projection was given
- [x] 3.3 Update tests: `;` on bare inserts in statement strings; tests reading `InsertResult.Record` use a projection and `Records[0]`
- [x] 3.4 Tests for each scenario in the `insert` delta spec, including the REPL waiting on an unterminated insert

## 4. Docs and verification

- [x] 4.1 Update `README.md`: every insert example (`;`, `1` or a table), the `@auto` section showing `=> {id}`, the Status line; update `QUERY_LANGUAGE.md`'s insert section
- [x] 4.2 Run `go vet ./...` and `go test ./...`
