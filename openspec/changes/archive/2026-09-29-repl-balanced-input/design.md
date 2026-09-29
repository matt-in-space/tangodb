## Context

`repl.RunREPL` appends each line to a buffer and calls `core.Parse`. On success it runs the statement; on `ErrIncompleteInput` it prints the continuation prompt and keeps reading; on any other error it prints the error and **resets the buffer**. The parser fails at the first bad token, so for `>> user {` / `id: 1` / `name: Matt` / `age: 3` / `};` the error fires on line 3, and lines 4 and 5 are each parsed as separate statements (`unrecognized statement` twice).

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- One statement in, one result or one error out, however many lines it spans.

**Non-Goals:**
- Reporting errors before the statement is finished. Waiting until brackets close is the chosen trade-off.
- Recovering from a statement whose brackets never balance (e.g. a stray `{`), other than the existing Ctrl+D exit.

## Decisions

1. **The check lives in `core.Parse`, not the REPL.** After lexing, `Parse` counts bracket depth over the tokens: `{`/`(` open, `}`/`)` close. If depth ends above zero, `Parse` returns `ErrIncompleteInput` without parsing. The REPL already waits on that error, so it needs no new logic. Any other caller of `Parse` gets the same rule for free.

2. **Counted on tokens, not characters.** Brackets inside string literals are part of a `tokenString` and never counted. An unterminated string is already `ErrIncompleteInput` from the lexer.

3. **Extra closing brackets are not waited on.** If depth ever drops below zero, `Parse` parses normally, so the resulting error is reported immediately rather than waiting for input that can't fix it.

4. **`{` and `(` are counted as one depth.** Mismatched pairs like `{ )` aren't a balancing question; they're a parse error, which the parser reports once depth returns to zero.

5. **Errors not caused by missing input still reset the buffer**, as today. Once brackets balance, the statement is complete, so an error means the whole statement is wrong.

## Risks / Trade-offs

- [A mistake on line 2 of a 20-line statement isn't reported until line 20] → Accepted: that's the trade-off chosen, and it's how psql behaves.
- [A stray `{` leaves the REPL waiting indefinitely] → The user can close the bracket (any statement then errors once) or press Ctrl+D. Worth revisiting if it's annoying in practice.
