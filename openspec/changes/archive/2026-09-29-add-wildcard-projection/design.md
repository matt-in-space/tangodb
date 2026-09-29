## Context

`core/parse_read.go` holds `parseProjection()`, shared by `parseRead`/`parseDelete`/`parseMerge`. `core/read.go` holds `db.read`, plus `renderTable`/`formatCell`, already shared by all three `Result` types' `String()` methods. Today: `db.read` treats `projection == nil` as "expand to every schema field" (unique to read); `db.delete`/`db.merge` treat `projection == nil` as "count only, don't populate `Records`" via `wantRecords := projection != nil`, and format their count-only result as `"%d deleted"`/`"%d updated"`.

See `proposal.md` for the full motivation, including why the wildcard work and the count-only unification are being done together (they'd otherwise edit the same requirement blocks in two separate changes).

## Goals / Non-Goals

**Goals:**
- `{*}` works identically wherever `parseProjection()` is used.
- `nil` projection means the same thing everywhere: count-only, no records populated, one shared bare-integer rendering.
- `db.read`, `db.delete`, `db.merge` share as much of their "was a projection given, and if so is it a wildcard" logic as possible.

**Non-Goals:**
- Nested/embedded wildcards, insert's not-yet-built return-projection, a richer `Projection` type — same non-goals as the original wildcard-only design.
- Changing delete/merge's *matching* or *mutation* semantics — only their result *formatting* and read's *default* change.

## Decisions

1. **New `*` lexer token**, single character, no lookahead needed.

2. **`parseProjection()` special-cases a leading `*`** before its normal field-list loop: if the first token after `{` is `*`, consume it, require `}` immediately next, and return the reserved sentinel `[]string{"*"}` instead of running the named-field loop. Mixing (`{id, *}` or `{*, id}`) is rejected by the existing "expected identifier" / "unexpected token" error paths, not a new bespoke check.

3. **`ReadResult` gains `Count int`**, matching `DeleteResult`/`MergeResult`'s field shape and order exactly (`Count`, `Projection`, `Records`).

4. **`db.read` adopts the `wantRecords := projection != nil` pattern.** When `projection == nil`, matching still happens (to produce an accurate count) but `Records`/`Projection` stay unset — mirroring delete/merge exactly. When non-nil (an explicit field list or the wildcard sentinel), behave as today: validate fields, expand if wildcard, populate `Records`/`Projection`.

5. **`expandProjection` only handles the wildcard case now.** Since `nil` uniformly means count-only (never "expand") across all three operations, the function no longer has a nil-branch — it fires only when `isWildcardProjection(projection)` is true. This is strictly simpler than the original wildcard-only design, which needed `expandProjection` to treat `nil` and the wildcard as equivalent specifically because read's `nil` used to mean something different from delete/merge's.

6. **One shared rendering helper replaces three near-duplicate `String()` bodies.** Concretely, in `core/read.go`:
   ```go
   func renderCountOrTable(count int, projection []string, records []Entity) string {
       if projection == nil {
           return strconv.Itoa(count)
       }
       if len(records) == 0 {
           return "no records found"
       }
       return renderTable(projection, records)
   }
   ```
   `ReadResult.String()`, `DeleteResult.String()`, and `MergeResult.String()` each become a one-line call: `return renderCountOrTable(r.Count, r.Projection, r.Records)`.

7. **Column order for `{*}`** — unchanged, alphabetical, matching the sort already used elsewhere (`Collection.String()`, the pre-existing nil-expansion logic being removed here).

## Risks / Trade-offs

- [Breaking change to already-shipped delete/merge output (`"N deleted"` → `"N"`)] → Mitigation: personal/learning project, no external consumers depending on exact output text yet; the tradeoff is made consciously as part of this change, with existing tests/docs updated in the same pass rather than left stale.
- [Read gains a new count-only default that changes existing REPL examples] → Mitigation: same — `README.md`'s existing `;`-omission example is being rewritten as part of this change, not left inconsistent.
- [A single shared rendering helper across three operations risks becoming a dumping ground if a fourth operation's result shape ever diverges] → Mitigation: the three shapes (`Count`, `Projection`, `Records`) are already structurally identical; if a future operation's result genuinely needs a different shape, it simply doesn't use this helper — nothing forces it to.

## Migration Plan

N/A — no data migration. Behavior migration: any existing usage relying on delete/merge's `"N deleted"`/`"N updated"` wording, or read's implicit all-fields default, needs to adjust — covered by this same change's doc/test updates, nothing left dangling.
