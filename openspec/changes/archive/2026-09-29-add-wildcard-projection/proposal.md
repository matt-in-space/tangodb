## Why

Delete and merge's `=> {...}` RETURNING clause can only name specific fields, with no way to ask for everything without hand-typing the schema. Read has the opposite asymmetry: omitting `=>` silently defaults to every field, while delete/merge's omission means something else entirely (count only) — inconsistent, and there's no way to explicitly request "everything" for any of the three. Working through this surfaced a second inconsistency worth fixing at the same time: delete/merge's existing count-only result formats itself with a verb (`"5 deleted"`, `"5 updated"`), and read has no count-only mode at all yet. Fixing the wildcard gap and the count-only asymmetry together avoids two separate changes competing to edit the same requirement blocks in `delete`/`merge`'s specs.

## What Changes

- Add a `*` lexer token and teach `parseProjection()` (shared by read/delete/merge) to accept `{*}` meaning "every field currently in the schema." `*` must be the sole content of the braces; mixing with named fields is a parse error (falls out of existing error paths, not a new bespoke check).
- **BREAKING** (already-shipped behavior): read's "no projection given" meaning changes from "default to every field" to **count-only**, matching delete/merge's existing behavior. `<< user(id: 1);` now returns a bare count, not the full record; use `<< user(id: 1) => {*};` to get everything back explicitly.
- **BREAKING** (already-shipped output): delete's and merge's count-only result format changes from `"N deleted"` / `"N updated"` to a bare integer `"N"`, matching read's new count-only format. One consistent rendering for "no projection was given," used identically by all three operations.
- `ReadResult` gains a `Count int` field, mirroring `DeleteResult`/`MergeResult`'s existing shape exactly (`Count`, `Projection`, `Records`).
- `db.read` adopts the same `wantRecords := projection != nil` gating pattern delete/merge already use, rather than its current "nil means expand to all fields" logic.
- The wildcard sentinel (`{*}` → `["*"]`) becomes the *only* thing `expandProjection` needs to resolve — `nil` no longer means "expand" for any operation, only "count-only," uniformly across read/delete/merge.
- `ReadResult`, `DeleteResult`, `MergeResult`'s `String()` methods collapse into calls to one shared rendering helper (bare count / "no records found" / table), replacing three near-duplicate implementations.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `delete`:
  - "Delete result can opt into returning deleted records" gains `{*}` support and a mixing-rejected scenario.
  - "Delete result defaults to a count" changes its output format from `"N deleted"` to a bare integer.
- `merge`:
  - "Merge result can opt into returning updated records" gains `{*}` support and a mixing-rejected scenario.
  - "Merge result defaults to a count" changes its output format from `"N updated"` to a bare integer.

Read still has no capability spec of its own (per prior precedent — read/insert/define-collection were never captured as OpenSpec capabilities). Its new count-only default and `{*}` support are implemented and tested at the code level but not recorded as a spec delta here, for the same reason.

## Impact

- New files: none.
- Modified files: `core/lexer.go` (new `*` token), `core/parse_read.go` (shared `parseProjection()` wildcard handling), `core/read.go` (`ReadResult` gains `Count`, `db.read` restructured to the `wantRecords` pattern, new shared count/table rendering helper), `core/delete.go`, `core/merge.go` (both switch to the shared rendering helper and drop their verb-specific formatting).
- Docs: `README.md`'s querying/deleting/merging sections all need their example output updated — delete/merge's count examples currently show `"1 deleted"`/`"1 updated"`, now `"1"`; read's `;`-omission example currently shows the full table, now shows a bare count, with `{*}` shown as the explicit way to get the table back. `QUERY_LANGUAGE.md`'s Read/Delete/Merge sections get equivalent updates.
