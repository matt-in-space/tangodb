## Why

tangodb's query language defines four core statements (read, insert, merge, delete); read, insert, and delete are implemented. Merge — a bulk conditional update — is the remaining core operation, and it turns out to reuse almost all of delete's already-settled design (filter matching, mandatory parens, count-vs-RETURNING result shape) once "match-or-create" was narrowed down to "match, then bulk-update"; the two operations are structurally identical apart from what happens to a matched row.

## What Changes

- Add a `~>` lexer token and a `parseMerge()` grammar: `~> collection(filter) {payload} (=> projection)? ;?`. The payload comes directly after the filter, with no `=>` in front of it — matching insert's already-resolved grammar — so `=>` keeps meaning exactly one thing everywhere in the language: "the shape of what comes back."
- Filter parens are **mandatory, even empty** — the same rule, and the same reasoning, as delete: merge is a bulk mutating operation, and an omitted filter shouldn't be able to silently apply to an entire collection.
- **Narrows the design draft** (nothing implemented yet, so not a breaking change to any shipped behavior): `QUERY_LANGUAGE.md` currently describes `~>` as "merge / upsert (match-or-create)." This change removes the "create if not found" half entirely — there is no create path. Matching zero records is a true no-op (no update, no creation, no error). `QUERY_LANGUAGE.md`'s Statements table, Philosophy references, and Merge section wording all get corrected to match, and the existing Open Questions bullet about merge's `=>` overload is resolved and removed.
- Add `db.merge()`: for every record matching the filter, merge the payload's fields into it — a **partial** update, so fields not named in the payload are left untouched on the existing record. Reuses `recordMatchesFilter` and the same schema-field-exists validation read/delete already apply to filter and projection fields.
- The payload **SHALL NOT** include the collection's declared primary key field — rejected as an error unconditionally, regardless of whether that field is `@auto` or manually assigned. (This only restricts the payload; filtering *on* the primary key is unaffected and works exactly like any other field.) This exists because a bulk update could otherwise assign the same primary key value to multiple matched rows at once, corrupting `primaryIndex` (each write would silently clobber the previous mapping, leaving some updated rows unreachable by key even though they still physically exist).
- `MergeResult` mirrors `DeleteResult`'s shape: defaults to a count of records updated; `=>` is an opt-in, Postgres-`RETURNING`-style projection that also returns the updated records, limited to the projected fields.
- Wire `MergeOperation` into `Database.run()`'s dispatch; no REPL changes needed beyond that.

## Capabilities

### New Capabilities
- `merge`: the `~>` bulk-conditional-update statement — grammar (mandatory filter parens, mandatory payload, optional `=> projection`), partial-field update semantics, primary-key-in-payload rejection, no-op-on-zero-matches, and the count-vs-RETURNING result shape.

### Modified Capabilities
(none — `delete`'s existing spec and behavior are unchanged; this change is purely additive)

## Impact

- New files: `merge.go` and `parse_merge.go`, mirroring the existing `delete.go`/`parse_delete.go` per-operation file convention, plus their test files.
- Modified files: `lexer.go` (new `~>` token), `parser.go` (dispatch case for the new token), `database.go` (`run()` dispatch case).
- Docs: `README.md` gains a "Merging records" section; `QUERY_LANGUAGE.md`'s Statements table, Philosophy line, and Merge section get corrected to describe bulk-update-only semantics (no create), and its Open Questions bullet about merge's `=>` overload is resolved and removed.
