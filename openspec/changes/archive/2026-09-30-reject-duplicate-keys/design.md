## Context

`parseRecordLiteral`, `parseFilter`, and `parseFieldBlock` each build a map with `m[key] = value`, so a repeated key silently overwrites the earlier one. Record literals nest (the value of an embedded block field is a nested `parseRecordLiteral`), and schema blocks nest (`parseFieldBlock` recurses with a dotted path prefix). Filter keys may be dotted paths (`address.city`).

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Any repeated key in a record literal, filter, or schema block is a parse error, naming the full path.

**Non-Goals:**
- Conditions that overlap without being the same key, like `(address.city: "A" address: {city: "B"})`. Those only become writable with nested filters, so `nested-filters` handles them.
- Projections, where naming a column twice already just shows it once.

## Decisions

1. **Detect repeats while parsing.** Each of the three loops checks whether the key is already in the map it's building before storing it. A repeated key is a syntax-level mistake, so it's reported as a parse error, before any validation or execution. That keeps the statement all-or-nothing.

2. **Messages name the full path.** `parseRecordLiteral` gains a `path` parameter: `""` for a top-level record, `"address."` for the literal inside `address`, and so on. It passes `path + field + "."` when it recurses through `parseValue`, which gets the same parameter so the path flows down. `parseFieldBlock` already has its path. Filter keys are used as written, since a dotted key is already its full path.

3. **Wording.** Records and filters use `field %q is given more than once`, since those keys are given values. Schema blocks use `field %q is declared more than once`.

## Risks / Trade-offs

- [Threading a path through `parseValue`] → A small signature change on a few internal functions. Filters pass the key's path, so nested filter objects are ready for `nested-filters`.
