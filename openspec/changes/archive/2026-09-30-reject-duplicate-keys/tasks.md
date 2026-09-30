## 1. Parser

- [x] 1.1 `parseRecordLiteral` and `parseValue` take a path prefix; `parseRecordLiteral` rejects a repeated key with the full path
- [x] 1.2 `parseFilter` rejects a repeated key
- [x] 1.3 `parseFieldBlock` rejects a repeated name with `is declared more than once`
- [x] 1.4 Tests for every scenario in both delta specs, including that a projection naming a column twice still works

## 2. Docs and verification

- [x] 2.1 README: note that a field can be given or declared only once
- [x] 2.2 Run `go vet ./...` and `go test ./...`
