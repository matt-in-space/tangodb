## 1. Bracket balance

- [ ] 1.1 In `core.Parse`, after lexing, count bracket depth over the tokens (`{`/`(` +1, `}`/`)` -1); if it ends above zero without ever going negative, return `ErrIncompleteInput`
- [ ] 1.2 Parser tests: open brace → incomplete, open paren → incomplete, brace inside a string doesn't count, extra closing bracket errors immediately, balanced input parses as before
- [ ] 1.3 Fix any existing `Parse` tests whose unbalanced input now returns `ErrIncompleteInput`

## 2. REPL

- [ ] 2.1 REPL tests: the multi-line-mistake scenario prints exactly one error; the next statement runs normally; `};` alone errors immediately
- [ ] 2.2 Confirm `repl/repl.go` needs no change (it already waits on `ErrIncompleteInput`); if it does, keep it minimal

## 3. Docs and verification

- [ ] 3.1 README "Running the REPL": explain that the REPL waits until brackets close and reports one error per statement
- [ ] 3.2 Run `go vet ./...` and `go test ./...`
