## 1. Lexer

- [x] 1.1 In `core/lexer.go`, change the `,` case to skip the rune without emitting a token, and delete the `tokenComma` constant; verify `go vet ./...` flags the three now-dangling `tokenComma` references in the parser (confirming they're the complete set)

## 2. Parser

- [x] 2.1 Remove the "expect a comma between entries" block from `parseRecordLiteral` in `core/parse_insert.go`
- [x] 2.2 Remove the same block from `parseFilter` and `parseProjection` in `core/parse_read.go`, and confirm `go vet ./...` passes

## 3. Tests

- [x] 3.1 Replace `TestParseInsert_RejectsMissingComma` in `core/parse_insert_test.go` with a test that `>> user {id: 1 name: "Matt"}` parses to the same record as the comma-separated form
- [x] 3.2 Add parser tests for: a filter without commas (`<< user(id: 1 name: "Matt") => {id}`), a projection without commas (`<< user() => {id name}`), a schema block with commas (`user { id: int @id, name: text }`), mixed and trailing commas in a record literal, and a string value containing a comma (`{name: "Smith, Matt"}`) keeping its comma
- [x] 3.3 Run `go vet ./...` and `go test ./...` and confirm every test passes, including the existing wildcard-mixing rejection tests (`{*, id}` / `{id, *}`) and the trailing-comma test, which should keep passing unchanged

## 4. Docs

- [x] 4.1 Update `README.md` where it says fields are separated by commas (the "Inserting a record" section) to say commas are optional, with a comma-free example verified against real `go run .` output
- [x] 4.2 Update `QUERY_LANGUAGE.md` to state the optional-comma rule once, near the Philosophy section's "one shared grammar" line
