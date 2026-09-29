## Why

The four places the language lists things disagree about separators: schema blocks use whitespace only (`user { id: int @id name: text }`), while record literals, filters, and projections require commas (`{id: 1, name: "Matt"}`, `(id: 1, name: "Matt")`, `{id, name}`). That's two dialects for the same idea, and it's about to get worse — nested schema blocks are next, and the spec's own nested example already uses commas inside a schema block. Commas help a one-liner read well; whitespace reads better across multiple lines. Treating commas as optional everywhere gives both, with one rule.

## What Changes

- The lexer treats `,` as whitespace — it's skipped the same way spaces and newlines are, and never produces a token (following Clojure's convention).
- Record literals, filters, and projections no longer require commas between entries: `{id: 1 name: "Matt"}`, `(id: 1 name: "Matt")`, and `{id name}` all parse, and so do their comma-separated forms.
- Schema blocks gain optional commas for free: `user { id: int @id, name: text }` now parses too, alongside the existing whitespace-only form.
- **BREAKING**: input that used to be a parse error — entries without a comma between them — now succeeds. Nothing depends on the old error.
- Commas inside quoted strings are unaffected; they're part of the string value, not separators.

## Capabilities

### New Capabilities
- `query-syntax`: lexical rules shared by every statement — starting with how entries in schema blocks, record literals, filters, and projections are separated.

### Modified Capabilities
(none — `delete` and `merge`'s requirements describe filter semantics, not separator syntax, so none of their requirements change)

## Impact

- `core/lexer.go`: the `,` case skips instead of emitting a token; `tokenComma` goes away.
- `core/parse_insert.go` (`parseRecordLiteral`), `core/parse_read.go` (`parseFilter`, `parseProjection`): the "expect a comma between entries" step is removed.
- Tests: `TestParseInsert_RejectsMissingComma` asserts the old behavior and flips; new tests cover comma-free and comma-in-schema forms.
- Docs: `README.md` and `QUERY_LANGUAGE.md` describe commas as optional.
