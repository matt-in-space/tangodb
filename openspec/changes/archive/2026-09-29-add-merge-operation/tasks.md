## 1. Lexer

- [x] 1.1 Add a `tokenMergeOp` (`~>`) token to `lexer.go`, mirroring the existing `<`/`>`/`!` two-char lookahead handling (including returning `ErrIncompleteInput` when `~` is the last rune) and verify `go vet ./...` passes with the new case wired in

## 2. Parser

- [x] 2.1 Add `parseMerge()` on the shared `*parser` in a new `parse_merge.go`, requiring `(` immediately after the collection name (mandatory, like delete's filter — not optional like read's), then requiring `{` for the payload (reusing `parseRecordLiteral()` from `parse_insert.go`), then optionally `=> projection` and an optional trailing `;` via `expectEndOfStatement()`. `EOF` right after the collection name (before `(`) is `ErrIncompleteInput`; anything else in place of `(` is a hard error, same pattern as delete
- [x] 2.2 Add `ParseMerge(input string) (MergeOperation, error)` as the typed convenience wrapper, matching `ParseDelete`/`ParseInsert`'s pattern
- [x] 2.3 Add a `tokenMergeOp` dispatch case to `Parse()` in `parser.go` and verify with a test asserting `Parse("~> user(id: 1) {name: \"Matt\"};")` returns a `MergeOperation`
- [x] 2.4 Add parser-level tests covering: `~> user(id: 1) {name: "Matt"};`, `~> user() {name: "Matt"};` (explicit empty filter), `~> user(id: 1) {name: "Matt"} => {id, name};`, `~> user(id: 1)` rejected as incomplete (no payload yet), `~> user {name: "Matt"};` (no parens) rejected as a hard error, `~> user(id: 1) => {id}` rejected (payload skipped straight to `=>`)

## 3. Database operation

- [x] 3.1 Define `MergeOperation{Collection string, Filter map[string]any, Payload Entity, Projection []string}` and `MergeResult{Count int, Projection []string, Records []Entity}` in a new `merge.go`
- [x] 3.2 Implement `db.merge(collectionName string, filter map[string]any, payload Entity, projection []string) (OperationResult, error)`: validate the collection exists and that every filter/projection field is declared in its schema (same pattern as `db.delete`); if the collection has a primary key, reject with an error (no mutation) if `payload` contains that key — checked once, up front, before touching any record
- [x] 3.3 Find matching row ids via `recordMatchesFilter` (reuse from `read.go`), and for each match, merge the payload's fields into the existing record in place (only the fields named in `payload` change; anything else on the record is left as-is) and increment the count
- [x] 3.4 Populate `MergeResult.Records`/`Projection` only when a projection was given (non-nil `projection` from the parser, same nil-vs-explicit-empty-`{}` handling `parse_delete.go` already has to get right); otherwise leave them nil so only `Count` is meaningful. When populated, `Records` holds each matched record's state *after* the merge, not before
- [x] 3.5 Do not touch `nextID`, `nextAutoValue`, or `primaryIndex` anywhere in the merge path — merge never changes a record's primary key value (enforced by 3.2) or creates/removes rows, so none of those need updating
- [x] 3.6 Add a `String()` method on `MergeResult`, mirroring `DeleteResult.String()`: prints the count when `Projection` is nil, `"no records found"` when a projection was given but nothing matched, and `renderTable(r.Projection, r.Records)` otherwise
- [x] 3.7 Wire `case MergeOperation: return db.merge(op.Collection, op.Filter, op.Payload, op.Projection)` into `Database.run()` in `database.go`

## 4. Tests

- [x] 4.1 Add `merge_test.go` covering: merging into a filtered subset leaves non-matching records' fields (and other records entirely) intact; merging with an empty filter updates every record in the collection; merging with fields not present in the payload leaves those fields' existing values unchanged (partial update, not full replace); merging with zero matches returns `Count: 0` with no error and no record created; merging without a projection returns only a count (`Records` nil); merging with a projection returns the post-merge state of updated records, limited to the projected fields; merging with a payload that includes the primary key field is rejected with no mutation to any record; merging into an unknown collection errors; merging with an unknown filter or projection field errors
- [x] 4.2 Add a REPL-level test in `repl_test.go` exercising define -> insert -> merge -> read in one session, confirming the merged field actually changed and untouched fields survived
- [x] 4.3 Run `go vet ./...` and `go test ./...` and confirm all tests pass, including every pre-existing test (no regressions)

## 5. Docs

- [x] 5.1 Update `QUERY_LANGUAGE.md`: correct the Statements table and Philosophy references away from "merge/upsert (match-or-create)" to describe bulk-update-only semantics (no create path), update the Merge section's example to the new `~> collection(filter) {payload}` grammar (no `=>` before the payload), and remove the now-resolved Open Questions bullet about merge's `=>` overload
- [x] 5.2 Add a "Merging records" section to `README.md`, verified against real `go run .` output the same way every other section was (manually exercise each example before writing it down), covering: a basic filtered merge, an empty-parens merge-everything, the rejected bare `~> user` form, a partial-update example showing an untouched field, the primary-key-in-payload rejection, zero-match no-op, and the count-vs-`=>` result difference
- [x] 5.3 Update `README.md`'s Status section to mention merge alongside delete
