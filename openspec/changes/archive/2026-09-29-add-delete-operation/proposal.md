## Why

tangodb's query language defines four core statements (read, insert, merge, delete), but only read and insert are implemented. Delete is the next operation needed to round out basic CRUD through the REPL, and it's also the first real exercise of a storage decision made earlier specifically to support it: collection records are stored in a `map[uint64]Entity` keyed by a surrogate row id (rather than an append-only slice) precisely so a row can be removed in O(1) without shifting or invalidating any other record's identity.

## What Changes

- Add a `!>` lexer token and a `parseDelete()` grammar: `!> collection(filter) (=> projection)? ;?`.
- Unlike read, the filter parens are **mandatory**, even when empty (`!> user();`). A bare `!> user` (no parens at all) is a hard parse error, not a shorthand for "delete everything" — delete doesn't get the low-friction bare-collection form read's `;` shortcut has, since an omitted filter is the classic accidental-mass-deletion failure mode (forgetting a `WHERE` clause), independent of which operator symbol was used.
- Add `db.delete()`: matches records using the same equality-filter logic `db.read()` already uses, then removes each matched row from `collection.records` and (if a primary key exists) `collection.primaryIndex`.
- Matching zero records is a no-op — returns a count of 0, not an error.
- Auto-increment counters (`nextID`, `nextAutoValue`) are never rolled back or reused after a delete; they stay monotonic.
- `DeleteResult` defaults to just a count of deleted records. `=>` is an opt-in, Postgres-`RETURNING`-style projection that adds the actual deleted records (optionally narrowed by field) to the result — the records are already loaded in-memory mid-delete regardless, so this costs nothing extra to support.
- Wire `DeleteOperation` into `Database.run()`'s dispatch; no REPL changes needed beyond that, since the REPL already renders whatever `OperationResult` comes back.
- Use the confirmed `!>` symbol (not the word `del`) — a word-form keyword would collide with the "bare identifier could still become a collection name" ambiguity the same way `exit` already had to be special-cased for; `!>` avoids that entirely and is also physically distant from the other three operators (`<<`, `>>`, `~>`) on a standard keyboard.

## Capabilities

### New Capabilities
- `delete`: the `!>` delete statement — grammar (mandatory filter parens, optional `=>` projection), row removal from both `records` and `primaryIndex`, no-op-on-zero-matches semantics, and the count-vs-RETURNING result shape.

### Modified Capabilities
(none — this is the first spec-level capability written for this project; `read`, `insert`, and collection definition have no specs yet either, and are out of scope for this change)

## Impact

- New files: `delete.go` and `parse_delete.go`, mirroring the existing `insert.go`/`parse_insert.go` and `read.go`/`parse_read.go` per-operation file convention, plus their test files.
- Modified files: `lexer.go` (new `!>` token), `parser.go` (dispatch case for the new token), `database.go` (`run()` dispatch case).
- Docs: `README.md` gains a "Deleting records" section; `QUERY_LANGUAGE.md`'s Philosophy line (which still says delete is spelled out as `del`) and its Open Questions bullet about delete ceremony both get updated to reflect the `!>` decision and the mandatory-parens resolution.
