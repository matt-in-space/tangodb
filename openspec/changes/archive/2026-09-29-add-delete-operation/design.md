## Context

tangodb implements each query-language statement as a consistent triad: an `Operation` struct, a `Result` struct, and a `(db *Database) <op>()` method, dispatched via `Database.run()`'s type switch; parsing mirrors this with a `parse<Op>()` method on the shared `*parser`, dispatched via `Parse()`'s token-based switch (see `read.go`/`parse_read.go` and `insert.go`/`parse_insert.go`).

`Collection` stores records in `records map[uint64]Entity`, keyed by an internal surrogate row id, with a separate `primaryIndex map[any]uint64` mapping the declared primary key's value to that row id (populated only when a primary key exists). This shape — over an append-only slice — was chosen specifically so a row could later be removed in O(1) without shifting or invalidating any other record's identity. This change is the first thing that actually exercises that.

See `proposal.md` for the full set of behavior decisions settled during discussion (mandatory filter parens, no-op on zero matches, count-vs-RETURNING result shape, monotonic auto-increment) — this document focuses on how they get implemented.

## Goals / Non-Goals

**Goals:**
- Implement the simple delete form: `!> collection(filter) (=> projection)? ;?`.
- Reuse the existing filter/projection/value parsing and matching machinery (`parseFilter`, `parseProjection`, `parseValue`, `recordMatchesFilter`) rather than duplicating it for delete.
- `DeleteResult` defaults to a plain count; `=>` opts into also returning the deleted records, optionally projected.

**Non-Goals:**
- The full pipeline form (`<< user(filter) => u | delete(u)`) — needs bound variables and a pipe operator, neither of which exist yet.
- Merge/upsert (`~>`) — a separate operation, not covered here.
- Dotted-path filtering on nested fields — read's filter doesn't support this yet either; delete inherits the same flat-field-only limitation.
- Physical compaction or space reclamation — not meaningful yet with purely in-memory storage; becomes relevant once persistence (a separate, already-discussed effort) is designed.

## Decisions

1. **`!>` operator token, not the word `del`.** A word-form keyword would collide with the "a bare identifier might still become a collection name" ambiguity the same way `exit` already had to be special-cased at the REPL level before parsing. A symbol has no such collision. `!` is also physically distant from `<`, `>`, `~`, `=` on a standard keyboard, which mitigates the "one keystroke away from something else" risk the original Philosophy note was concerned about.

2. **Filter parens are mandatory, even when empty — there's no bare-collection shorthand for delete.** Read's optional-parens-plus-`;` shorthand was considered and rejected for delete: the risk isn't confusing which operator was typed (decision 1 covers that), it's the classic "forgot the filter" failure mode, which persists regardless of how distinct `!>` is. Requiring `()` — even empty, `!> user();` — forces the filter position to be visibly acknowledged rather than silently skippable, at the cost of exactly one extra token, and reuses grammar that already exists (empty parens already mean "match everything" for read).
   - Parser behavior: after the collection name, `(` starts filter parsing as normal; `EOF` is `ErrIncompleteInput` (might still become `(` on the next line); anything else (`;`, `=>`, etc.) is an immediate hard error — the required token was skipped, not merely delayed.

3. **`db.delete()` reuses `recordMatchesFilter` from `read.go`** for matching, and the same schema-field-exists validation read already applies to both filter and projection fields.

4. **Row removal touches two structures per matched id:** `collection.records` and, if a primary key is declared, `collection.primaryIndex`. The record is looked up once (needed regardless, to compute the primary-key value for the index and to potentially return the row), then removed from both maps with a plain `delete()` — no shifting, no tombstoning.

5. **`DeleteResult` is its own type, not a reuse of `ReadResult`**, despite superficial shape overlap. Its primary payload is a count (`Count int`); `Projection`/`Records` are populated only when `=>` was used. This is a different contract than `ReadResult`, where records are always the primary payload — worth a distinct type even if the optional table-rendering path ends up sharing a small helper with `ReadResult.String()`.

6. **Zero matches is a no-op**, returning `DeleteResult{Count: 0}` (empty `Records` too, if `=>` was used) rather than an error — matches SQL `DELETE` semantics and mirrors read's existing non-error "no records found" behavior.

7. **Auto-increment counters (`nextID`, `nextAutoValue`) are never decremented or reused after a delete.** They stay strictly monotonic per collection, so a deleted row's id is never silently reassigned to a later, unrelated row.

## Risks / Trade-offs

- [Delete's mandatory-parens grammar is asymmetric with read's optional-parens grammar] → Mitigation: deliberate and documented (Decision 2), not an oversight — both `README.md` and `QUERY_LANGUAGE.md` call out the asymmetry explicitly so it reads as an intentional safety choice rather than an inconsistency.
- [`DeleteResult` duplicates some of `ReadResult`'s shape] → Mitigation: acceptable given the different primary contracts (count-first vs. records-first); revisit if a third operation needs the same "count + optional records" shape and a shared type becomes worth it.
- [No compaction/reclaim story for deleted rows] → Mitigation: not a real risk today (purely in-memory, a Go `delete()` on a map is enough); flagged for the separate persistence design, where tombstones vs. real removal on-disk is already an open thread.

## Migration Plan

N/A — purely additive; no existing operation's behavior changes, no data migration involved.
