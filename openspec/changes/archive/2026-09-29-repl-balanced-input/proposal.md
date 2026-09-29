## Why

When a multi-line statement has a mistake partway through, the parser fails on the first bad token while the user is still typing. The REPL treats that failure like a finished statement and resets, so every remaining line (`age: 3`, `};`) is parsed as a new, meaningless statement. That produces a pile of confusing errors, and a leftover line that happens to be valid on its own could even run. Batch inserts are about to make long multi-line statements common, so this needs fixing first.

## What Changes

- A statement is not parsed while any `{` or `(` is still open. The REPL keeps reading lines until every opened bracket is closed, then parses the whole statement once.
- A mistake anywhere in a multi-line statement produces exactly one error, after the brackets balance. No leftover lines are re-parsed.
- Brackets inside quoted strings don't count.
- An unmatched closing bracket is parsed and reported right away, rather than waited on.

## Capabilities

### New Capabilities
- `repl`: how the interactive prompt reads input, decides a statement is complete, and reports errors.

### Modified Capabilities
None.

## Impact

- `core/parser.go`: `Parse` checks bracket balance on the lexed tokens before dispatching.
- `repl/repl.go`: no logic change expected, since it already waits on `ErrIncompleteInput`.
- Tests: new REPL and parser tests; existing `Parse` tests that expect a specific error from unbalanced input may now see `ErrIncompleteInput` instead.
