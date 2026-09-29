## 1. Boolean literals

- [x] 1.1 Add a `tokenIdent` case to `parseValue` in `core/parse_insert.go`: `true` → `true`, `false` → `false`, any other identifier → the existing "expected a value" error
- [x] 1.2 Parser tests: `true`/`false` in a record literal, a filter, and a merge payload; `"true"` stays a string; a bare `yes` is a parse error

## 2. Validator

- [x] 2.1 Create `core/validate.go` with `validateValue(DataType, any) error` (exact type match, no conversions) and `validateFields(*Collection, map[string]any) error` (undeclared-field check plus type check, in sorted field order)
- [x] 2.2 Unit tests in `core/validate_test.go`: each type accepts its own Go type and rejects the others (including `int64` into `float`), undeclared field, deterministic error when several fields are bad

## 3. Wire into operations

- [x] 3.1 `db.insert`: validate the record before any primary-key or auto-increment logic
- [x] 3.2 `db.merge`: validate the filter and the payload before the primary-key check and before touching any record
- [x] 3.3 `db.read` and `db.delete`: replace the existence-only filter loop with `validateFields`
- [x] 3.4 Update existing tests that build values with untyped Go `int` constants to use `int64`
- [x] 3.5 Operation tests: wrong type on insert/merge/filter, undeclared field on insert and merge, invalid statements change nothing (insert, bulk merge, delete), bool round-trip

## 4. Docs and verification

- [x] 4.1 Update `README.md` and `QUERY_LANGUAGE.md`: boolean literals, strict typing (`10.0` for floats), undeclared fields rejected, all-or-nothing validation
- [x] 4.2 Run `go vet ./...` and `go test ./...`
