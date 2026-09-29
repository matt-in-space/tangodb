## Context

tangodb implements each query-language statement as a consistent triad — an `Operation` struct, a `Result` struct, and a `(db *Database) <op>()` method, dispatched via `Database.run()`'s type switch — with parsing mirroring this as a `parse<Op>()` method on the shared `*parser`, dispatched via `Parse()`'s token-based switch. `delete.go`/`parse_delete.go` are the most directly relevant precedent: delete already establishes mandatory (even empty) filter parens, reuses `recordMatchesFilter` for matching, and returns a count-by-default / `=>`-opt-in-RETURNING result shape via `DeleteResult`.

See `proposal.md` for the full set of behavior decisions settled during discussion (bulk update on match, no create path, mandatory parens, primary-key-in-payload rejection, count-vs-RETURNING result) — this document focuses on how they get implemented.

## Goals / Non-Goals

**Goals:**
- Implement `~> collection(filter) {payload} (=> projection)? ;?`.
- Reuse delete's already-proven pieces: `parseFilter()`, `parseProjection()`, `expectEndOfStatement()`, `recordMatchesFilter`, and the mandatory-non-empty-parens parsing pattern.
- `MergeResult` mirrors `DeleteResult`'s shape and rendering (`renderTable` helper) exactly, differing only in what happened to the matched rows.

**Non-Goals:**
- Any create-on-zero-match ("upsert") path — explicitly removed from scope during discussion, not deferred; if a real upsert primitive is wanted later, it needs its own design (likely tied to a single unique match, unlike this operation's bulk semantics).
- Restricting matches to "at most one" — merge intentionally reuses the filter's ordinary multi-match semantics (same as read/delete); precise single-record targeting is left to future pipeline/bound-variable functionality, not built here.
- Batch payloads, nested/embedded field updates, dotted-path filtering — none of these exist for insert/read/delete yet either; merge inherits the same flat-field-only scope.

## Decisions

1. **Grammar places the payload directly after the filter, with no `=>` in front of it** — `~> user(id: 1) {name: "Matt"}`, not `~> user(id: 1) => {name: "Matt"}`. This is the exact fix already applied to insert's grammar, extended to merge as flagged in `QUERY_LANGUAGE.md`'s own Open Questions. `=>` is reserved solely for the optional return-projection that follows the payload.

2. **Filter parens are mandatory, even when empty — no bare `~> user;` shorthand.** Same rule as delete, same reasoning: merge is a bulk mutating operation (it can update every record in a collection), so an omitted filter shouldn't be able to silently apply to the whole thing. Parser behavior mirrors delete's exactly: after the collection name, `(` starts filter parsing; `EOF` there is `ErrIncompleteInput`; anything else is an immediate hard error.

3. **Bulk update, not "at most one."** Every record matching the filter gets the payload merged into it — no attempt to detect or reject multiple matches. This was a deliberate choice (not an oversight): the filter has identical semantics everywhere else in the language (read, delete), and merge doesn't get a special, more restrictive reading of the same syntax. Precise "touch exactly one" targeting is left to future pipeline stages (e.g. bound variables + `limit(1)`), not built into merge's base grammar.

4. **The update itself is a partial merge, not a full replace.** Only the fields named in the payload change on a matched record; every other existing field is left as-is. This is the plain meaning of "merge" and matches how SQL's `UPDATE ... SET col = val` only ever touches the named columns.

5. **The payload must not include the collection's primary key field**, whether or not that field is `@auto`. Rejected as a validation error before any mutation happens. Alternative considered: allow it when only one row matches (since a single-row key change is semantically fine on its own). Rejected for simplicity and consistency — the rule doesn't need to reason about how many rows *will* match before deciding validity, and "the primary key is something you filter on, never something a merge payload sets" is a simple, uniform rule to state and to implement.

6. **Zero matches is a true no-op**: no update, no creation, no error. `MergeResult.Count` is 0. Matches delete's existing zero-match precedent.

7. **`MergeResult` is its own type, structurally identical to `DeleteResult`** (`Count int`, `Projection []string`, `Records []Entity`, same nil-projection-means-count-only convention, same `renderTable` reuse for the `=>` case). Considered making merge return a `DeleteResult` directly, or extracting a shared `Count`/`Projection`/`Records` type both use — rejected for now, consistent with the same reasoning applied when `DeleteResult` was kept separate from `ReadResult`: revisit if a third operation needs this exact shape and sharing becomes clearly worth it.

## Risks / Trade-offs

- [Bulk-match semantics mean a loose filter can update far more rows than intended] → Mitigation: mandatory filter parens (Decision 2) at least prevents the *fully* omitted-filter case; a loose-but-present filter matching more than intended is an inherent property of the filter grammar itself (same risk already exists for delete) rather than something new to merge.
- [`MergeResult` duplicates `DeleteResult`'s shape almost exactly] → Mitigation: acceptable, same reasoning as delete-vs-read; revisit if a third "count + optional records" consumer appears.
- [Rejecting primary-key-in-payload forecloses a legitimate single-row key rename via merge] → Mitigation: acceptable for now — a key rename can still be done as an explicit delete-then-insert; revisiting this would need its own single-match-only carve-out, which conflicts with Decision 3's bulk semantics.

## Migration Plan

N/A — purely additive; no existing operation's behavior changes, no data migration involved. `QUERY_LANGUAGE.md` wording corrections are documentation-only.
