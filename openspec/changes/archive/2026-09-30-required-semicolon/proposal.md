## Why

Whether a statement runs when you press Enter depends on a rule with two exceptions: `;` is required, *unless* the statement is a collection definition or ends in a `=>` projection. Both exceptions depend on the parser guessing whether more input could follow. So an insert of a nested record doesn't run at its last `}` (a `&` could follow), while a definition does. That's surprising in practice, and every feature that allows something optional at the end of a statement adds another case, as `&` just did and pipelines (`| order(...)`) would. A statement should run because you said it's finished, not because the parser guessed.

## What Changes

- **BREAKING**: every statement ends with `;`, including definitions and statements with a projection. It's a rule of the language, not only of the REPL: `Parse` treats input without a `;` as incomplete.
- A statement is complete at its first `;` outside any brackets. Until then nothing is parsed, so mistakes are reported once, when the statement is finished. That's consistent with how unclosed brackets already work.
- One statement at a time: anything after the `;` is an error, and nothing runs.
- `exit` stays a REPL command and needs no `;`.
- Forgetting the `;` just means the REPL keeps reading until one arrives.
- The per-statement "could more follow?" special cases go away, including insert's own terminator rule and the bare `<< user` ambiguity.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `query-syntax`: adds "Every statement ends with a semicolon"; scenario statements gain `;`.
- `insert`: removes "A bare insert needs a terminator to be complete" (replaced by the general rule); scenario statements gain `;`.
- `repl`: adds "Exit is a REPL command, not a statement".
- `collection-definition`, `merge`, `schema-enforcement`: scenario statements gain `;`. The wording changes; the behavior they describe doesn't.

## Impact

- `core/parser.go`: `Parse` finds the first top-level `;`; with none it returns `ErrIncompleteInput`; with tokens after it, an error; `expectEndOfStatement` requires the `;`.
- `core/parse_*.go`: the scattered "EOF here means `=>`/`&`/filter might still follow" cases become unnecessary and are removed.
- Tests: every statement gains `;`; tests of "incomplete because more might follow" become tests of "incomplete because there's no `;`".
- Docs: README states the rule up front, loses the "Selecting everything" ambiguity section, and every example ends with `;`. `QUERY_LANGUAGE.md` states the rule and updates its examples.
