## Context

`Parse` lexes the input, returns `ErrIncompleteInput` while brackets are unclosed, then dispatches to a per-statement parser. Each parser decides on its own whether reaching the end of input means "incomplete" or "done":
- define collection: done at its `}`
- insert: incomplete after the record literal or a trailing `&`
- read: incomplete after the collection name or filter
- delete and merge: incomplete after the filter or payload

`expectEndOfStatement` accepts an optional `;`, then requires end of input. The REPL waits on `ErrIncompleteInput` and otherwise runs the result or prints the error. `exit` is checked in the REPL before parsing.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- One completeness rule for every statement: it ends with `;`.
- Nothing is parsed until the statement is complete.

**Non-Goals:**
- Several statements per input. Anything after the `;` is an error for now.
- Hints when the `;` is forgotten.

## Decisions

1. **Completeness is decided once, in `Parse`, before any statement parser runs.** After lexing, `Parse` scans the tokens for the first `;` at bracket depth zero:
   - none → `ErrIncompleteInput` (this also covers unclosed brackets, since a `;` inside brackets doesn't count)
   - one, followed by more tokens → `unexpected input after statement: "<next token>"`
   - one, at the end → parse normally

   An unmatched closing bracket still parses immediately so its error isn't delayed, as today. Because nothing is parsed before the `;`, a mistake mid-statement waits until the statement is finished and is reported once.

2. **`expectEndOfStatement` requires the `;`.** At end of input it returns `ErrIncompleteInput`, so direct callers of `ParseInsert` and the like get the same rule. Any other token is `unexpected input after statement`. After the `;`, it requires end of input.

3. **The per-parser "more might follow" cases are removed.** With the `;` guaranteed present, reaching end of input inside a statement can't happen via `Parse`. The EOF checks in insert (after the record, after `&`), read (after the name or filter), delete, and merge that existed only to say "`=>` might follow" are removed. `ErrIncompleteInput` stays where it means genuinely truncated input: lexer cases like a lone `>` or an unterminated string, and the incomplete check in `expectEndOfStatement`.

4. **`exit` is unchanged.** The REPL checks for it before calling `Parse`, when the buffer is empty. `exit;` is not special; it's parsed as a statement, and fails as one.

5. **A `;` inside a string doesn't count**, because it's part of a `tokenString`. The same goes for brackets.

## Risks / Trade-offs

- [Every existing statement in tests and docs needs a `;`] → Mechanical churn, same as when insert started needing one.
- [A forgotten `;` leaves the REPL waiting] → Deliberate. The continuation prompt shows it's waiting, and the rule is stated up front in the README.
- [A future API or server client must send the `;`] → Accepted: one language with one rule. A client can append it.
