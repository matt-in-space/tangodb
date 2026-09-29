## 1. Lexer

- [x] 1.1 Add a `tokenStar` (`*`) token to `core/lexer.go` — a single-character token, no lookahead needed — and verify `go vet ./...` passes with the new case wired in

## 2. Parser

- [x] 2.1 In `core/parse_read.go`'s `parseProjection()`, special-case a leading `*`: if the first token after `{` is `tokenStar`, consume it, require `}` immediately next, and return the sentinel `[]string{"*"}` instead of running the existing named-field loop
- [x] 2.2 Add parser-level tests covering: `<< user => {*};`, `!> user(id: 1) => {*};`, and `~> user(id: 1) {name: "Matt"} => {*};` all parse with `Projection == []string{"*"}`; `{*, id}` and `{id, *}` are rejected as parse errors via the existing "expected identifier"/"unexpected token" error paths (not a new bespoke error)

## 3. Result shape and shared rendering

- [x] 3.1 Add `Count int` to `ReadResult` in `core/read.go`, matching `DeleteResult`/`MergeResult`'s field order (`Count`, `Projection`, `Records`)
- [x] 3.2 Add `renderCountOrTable(count int, projection []string, records []Entity) string` to `core/read.go`: returns the bare integer (`strconv.Itoa(count)`) when `projection == nil`, `"no records found"` when `projection` is set but `records` is empty, and `renderTable(projection, records)` otherwise
- [x] 3.3 Replace `ReadResult.String()`, `DeleteResult.String()`, and `MergeResult.String()` bodies with a single call each to `renderCountOrTable(r.Count, r.Projection, r.Records)`, removing their previous verb-specific formatting (`"%d deleted"`, `"%d updated"`) and read's previous `len(r.Records) == 0` check

## 4. Execution logic

- [x] 4.1 Add `isWildcardProjection(projection []string) bool` and `expandProjection(collection *Collection, projection []string) []string` to `core/read.go`: `expandProjection` returns `projection` unchanged unless it's the wildcard sentinel, in which case it enumerates `collection.data`'s keys, sorts them, and returns that
- [x] 4.2 Restructure `db.read` in `core/read.go` to the `wantRecords := projection != nil` pattern: always compute the match count; only validate projected fields, call `expandProjection`, and populate `Records`/`Projection` on the result when `wantRecords` is true
- [x] 4.3 In `core/delete.go`'s `db.delete`, after `wantRecords := projection != nil`, when `wantRecords` is true call `projection = expandProjection(collection, projection)` before the existing per-projected-field schema-validation loop
- [x] 4.4 In `core/merge.go`'s `db.merge`, the same change as 4.3

## 5. Tests

- [x] 5.1 Update `core/read_test.go`: replace the existing "nil projection expands to all fields" test with one confirming nil projection now returns `Count` only (no `Records`/`Projection`); add a wildcard-projection test confirming `{*}` still expands to every current schema field, alphabetically ordered
- [x] 5.2 Update `core/delete_test.go`: confirm the no-projection case's `DeleteResult.String()` is a bare integer (e.g. `"1"`, not `"1 deleted"`); add a wildcard-projection test
- [x] 5.3 Update `core/merge_test.go`: same two additions as 5.2, for merge
- [x] 5.4 Update `repl/repl_test.go`: the existing bare-read-with-semicolon test (`TestRunREPL_BareReadWithSemicolonReturnsEverything` or equivalent) now expects a bare count instead of a full table — rewrite it accordingly, and add a new REPL-level test showing `<< collection => {*};` returning the full table explicitly
- [x] 5.5 Run `go vet ./...` and `go test ./...` and confirm all tests pass, including every pre-existing test not touched by this change (no unrelated regressions)

## 6. Docs

- [x] 6.1 Update `QUERY_LANGUAGE.md`: mention `{*}` in the Read/Delete/Merge sections' projection examples; update the Read section to describe the new count-only default when `=>` is omitted (no more "defaults to every field"); update Delete/Merge's return-value examples to show bare integers instead of `"N deleted"`/`"N updated"`
- [x] 6.2 Update `README.md`'s "Querying records," "Deleting records," and "Merging records" sections: rewrite the `;`-omission example for read (now shows a bare count, with `=> {*};` shown as the explicit "give me everything" form), and update every delete/merge count example from `"N deleted"`/`"N updated"` to bare `"N"`. Verify every example against real `go run .` output before writing it down, the same way every prior doc section was
