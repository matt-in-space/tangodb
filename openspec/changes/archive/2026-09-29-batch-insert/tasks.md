## 1. Syntax

- [x] 1.1 Lexer: add `tokenAmp` for `&`
- [x] 1.2 `parseInsert`: read `&`-separated record literals (EOF after `&` → `ErrIncompleteInput`), then the existing `=>`/terminator tail; `InsertOperation.Record` becomes `Records []Entity`
- [x] 1.3 Parser tests: two and three records, trailing `&` is incomplete, `& ;` is a parse error, `=>` after the last record, `=>` between records is an error
- [x] 1.4 Update tests that use `InsertOperation{Record: ...}` or `o.Record`

## 2. Collecting problems

- [x] 2.1 Add a per-record helper in `core/validate.go` returning the record's problems as a `[]string`, in today's check order, first check per field wins
- [x] 2.2 Unit tests: several problems in one record, one problem per field, no problems

## 3. Batch insert

- [x] 3.1 `db.insert`: validate every record (plus in-batch duplicate keys and the projection), build the error per the design (`record N:` prefix only for batches; header only for several problems), and write nothing unless there are no problems
- [x] 3.2 Then, in input order: strip nulls, assign auto counters, store, index; `Count` is the number of records, `Records` in input order when projected
- [x] 3.3 Tests for every scenario in the `insert` and `schema-enforcement` delta specs
- [x] 3.4 Confirm every existing single-insert error test still passes unchanged

## 4. Docs and verification

- [x] 4.1 README: a batch insert section (syntax, all-or-nothing, the problem list); remove batches from the Status line's unsupported list; update `QUERY_LANGUAGE.md`'s batch example
- [x] 4.2 Run `go vet ./...` and `go test ./...`
