## 1. Lexer

- [x] 1.1 Add a `tokenDeleteOp` (`!>`) token to `lexer.go`, mirroring the existing `<`/`>` two-char lookahead handling (including returning `ErrIncompleteInput` when `!` is the last rune) and verify `go vet ./...` passes with the new case wired in

## 2. Parser

- [x] 2.1 Add `parseDelete()` on the shared `*parser` in a new `parse_delete.go`, requiring `(` immediately after the collection name (not optional like read's) — `EOF` there is `ErrIncompleteInput`, anything else is a hard error — reusing `parseFilter()` for the (possibly empty) filter and `parseProjection()`/`expectEndOfStatement()` for the optional `=> projection` and optional trailing `;`, and verify with a parser-level test covering: `!> user(id: 1);`, `!> user();`, `!> user(id: 1) => {id, name};`, `!> user` rejected as incomplete, `!> user;` (no parens) rejected as a hard error
- [x] 2.2 Add `ParseDelete(input string) (DeleteOperation, error)` as the typed convenience wrapper, matching `ParseInsert`/`ParseRead`'s pattern
- [x] 2.3 Add a `tokenDeleteOp` dispatch case to `Parse()` in `parser.go` and verify with a test asserting `Parse("!> user(id: 1);")` returns a `DeleteOperation`

## 3. Database operation

- [x] 3.1 Define `DeleteOperation{Collection string, Filter map[string]any, Projection []string}` and `DeleteResult{Count int, Projection []string, Records []Entity}` in a new `delete.go`
- [x] 3.2 Implement `db.delete(collectionName string, filter map[string]any, projection []string) (OperationResult, error)`: validate the collection exists and that every filter/projection field is declared in its schema (same pattern as `db.read`), find matching row ids via `recordMatchesFilter`, and for each matched id look up the record, delete it from `collection.records`, and delete its primary-key value from `collection.primaryIndex` when the collection has a primary key
- [x] 3.3 Populate `DeleteResult.Records`/`Projection` only when a projection was given (non-nil `projection` from the parser); otherwise leave them nil so only `Count` is meaningful
- [x] 3.4 Do not touch `nextID` or `nextAutoValue` anywhere in the delete path, and verify with a test that inserting after a delete produces a strictly greater id than any previously deleted one
- [x] 3.5 Add a `String()` method on `DeleteResult` that prints the count, and — when `Records` is non-nil — also renders them as a table (reuse or mirror `ReadResult.String()`'s `tabwriter` approach)
- [x] 3.6 Wire `case DeleteOperation: return db.delete(op.Collection, op.Filter, op.Projection)` into `Database.run()` in `database.go`

## 4. Tests

- [x] 4.1 Add `delete_test.go` covering: deleting a filtered subset leaves non-matching records intact; deleting with an empty filter removes everything in the collection; deleting zero matches returns `Count: 0` with no error; deleting without a projection returns only a count (`Records` nil); deleting with a projection returns the deleted records limited to the projected fields; deleting from an unknown collection errors; deleting with an unknown filter or projection field errors
- [x] 4.2 Add a REPL-level test in `repl_test.go` exercising define -> insert -> delete -> read in one session, confirming the deleted record no longer appears in a subsequent read
- [x] 4.3 Run `go vet ./...` and `go test ./...` and confirm all tests pass, including every pre-existing test (no regressions)

## 5. Docs

- [x] 5.1 Update `QUERY_LANGUAGE.md`: change the Philosophy line's `del` reference to `!>`, update the Delete section's simple-form example to show the mandatory-empty-parens requirement, and remove or resolve the "ceremony around unbounded bulk deletes" Open Questions bullet now that it's decided
- [x] 5.2 Add a "Deleting records" section to `README.md`, verified against real `go run .` output the same way every other section was (manually exercise each example before writing it down), covering: basic filtered delete, empty-parens delete-everything, the rejected bare `!> user` form, zero-match no-op, and the count-vs-`=>` result difference
- [x] 5.3 Update `README.md`'s Status section to move delete from "not yet supported" to supported, alongside its mandatory-filter-parens caveat
