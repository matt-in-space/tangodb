## 1. Completeness in the parser

- [x] 1.1 `Parse`: find the first top-level `;`; none → `ErrIncompleteInput`; tokens after it → `unexpected input after statement`; otherwise parse
- [x] 1.2 `expectEndOfStatement`: require `;` (end of input → `ErrIncompleteInput`), then end of input
- [x] 1.3 Remove the per-parser "`=>`/`&`/filter might still follow" EOF cases in insert, read, delete, and merge
- [x] 1.4 Parser tests: definition, insert, read with projection, delete, and merge each incomplete without `;`; `;` inside a string or brackets doesn't count; text after `;` is an error; an unmatched closing bracket still errors immediately

## 2. REPL

- [x] 2.1 Confirm the REPL needs no change beyond tests; `exit` still works without `;`
- [x] 2.2 REPL tests: a definition waits for `;`; a `;` on a later line runs the statement once; a mid-statement mistake is reported once, after `;`; `exit` without `;`

## 3. Existing tests

- [x] 3.1 Add `;` to every statement in core and REPL tests; rework tests that expected "incomplete because more might follow" into "incomplete because no `;`"
- [x] 3.2 Tests for every scenario in the `query-syntax` and `repl` deltas

## 4. Docs and verification

- [x] 4.1 README: state the `;` rule up front (near "Running the REPL"), remove the "Selecting everything" ambiguity section and the per-statement `;` explanations, end every example with `;`; `QUERY_LANGUAGE.md`: state the rule, update examples
- [x] 4.2 Run `go vet ./...` and `go test ./...`
