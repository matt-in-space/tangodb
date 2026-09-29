## Context

`core/lexer.go` emits a `tokenComma` for every `,`. Three parser loops then require one between entries — `parseRecordLiteral` (`core/parse_insert.go`), `parseFilter` and `parseProjection` (`core/parse_read.go`) — each with the same shape: parse an entry, then `if next != closer { expect(tokenComma) }`. The schema parser (`parseDefineCollection`) has no comma handling at all; its entries are separated by whitespace only. See `proposal.md` for why these should agree.

## Goals / Non-Goals

**Goals:**
- One separator rule for all four list constructs: commas are optional, whitespace alone is enough.
- The smallest change that gets there.

**Non-Goals:**
- Nested schema blocks and nested record literals — the next step, built on top of this one.
- Any other lexical change (e.g. boolean or negative-number literals).

## Decisions

1. **Commas become whitespace in the lexer, not an optional token in the parser.** The `,` case in `lex()` advances past the rune without emitting anything, and `tokenComma` is deleted. The three `expect(tokenComma)` blocks are removed outright.

   Alternative considered: keep `tokenComma` and make each parser loop accept it optionally (`if peek == tokenComma { next() }`). Rejected — it's the same behavior implemented four times instead of once, and the schema parser would need the same addition to gain commas. Doing it in the lexer makes all four constructs, including any future ones, consistent automatically. Clojure uses exactly this approach.

2. **No new ambiguity.** Every current list entry is self-delimiting: record/filter entries are `ident : value` where a value is a single string or number token, projection entries are single identifiers, schema entries are `ident : type @annotation*`. The next entry always starts with an identifier, so nothing needs a separator to be parsed unambiguously.

3. **Existing edge cases keep working without special handling:**
   - Trailing commas (`{id: 1,}`) — already allowed; still allowed, since the comma just disappears.
   - Wildcard mixing (`{*, id}` / `{id, *}`) — still rejected: `{* id}` fails the "expect `}` after `*`" check, and `{id *}` fails `expectIdent` on `*`.
   - Commas inside strings — `lexString` consumes them as string content before the `,` case is ever reached.

4. **Stray commas are accepted.** `{,id,,name,}` parses as `{id name}`. That falls out of treating commas as whitespace, and isn't worth extra code to forbid.

## Risks / Trade-offs

- [Commas can't carry meaning later — if values ever grow into expressions, `{x: 1 -2}` has no separator to disambiguate it] → Accepted: values are literals today, and this is easy to revisit if that changes.
- [Error messages lose "expected ','" as a hint] → Minor: a missing *value* or *colon* still produces a specific error, since those tokens are still required.
