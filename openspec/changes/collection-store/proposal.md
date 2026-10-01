## Why

The query layer keeps records in maps on `core.Collection` (`records`, `primaryIndex`, `nextID`, `autoCounters`). To persist anything, `core` needs to hand storage off to something with a clear contract, so the heap file, B+tree, and files can be built underneath it later without the query layer changing. Working top-down, this change defines that contract, the **collection store**, and a memory-backed implementation of it. It also runs the record layer (`Encode` / `Decode`) on every write and read.

## What Changes

- A new `storage` package with a **`Store` interface**, one store per collection:
  - `Insert`
  - `Lookup` by primary key
  - `Scan`, returning a cursor
  - `Update`, and `Delete` by an opaque `RecordRef`
  - reading `@auto` counters
- A **memory-backed implementation**, `MemoryStore`, that keeps **encoded bytes** rather than `Entity` values: every write goes through `record.Encode`, and every read through `record.Decode`.
- **All-or-nothing writes:** a store encodes and checks every record in an operation before storing any of them.
- **A record size limit, enforced now:** a record must fit in one data page. The limit is computed from the page format: 4096 − 16 (page header) − 4 (slot entry) = **4076 bytes**. Records spanning several pages are a later enhancement.
- **Primary keys are `int` or `text`.** The store maintains its primary index and rejects duplicate or changed keys as a backstop. (`core` already checks duplicates first, with its own messages.)
- **`@auto` counters live in the store.** They advance only when an insert succeeds, so a failed insert never consumes a value.
- **Scans promise no order.**
- **Records returned are independent copies,** so changing one never changes what's stored.
- **Storage errors are prefixed `storage: `,** to tell them apart from language errors.
- **Not in this change:** wiring `core` to the store (the follow-up change, which also restricts `@id` to `int` and `text` as a language rule), pages, the heap, the B+tree, and files.

## Capabilities

### New Capabilities
- `collection-store`: the contract between the query layer and storage for one collection's records, primary index, and `@auto` counters.

### Modified Capabilities
None.

## Impact

- New package `storage`: `Store`, `Cursor`, `RecordRef`, `Config`, `MemoryStore`, the size limit constants, and error helpers.
- No changes to `core` or the REPL; the store is exercised by its own tests until the follow-up change wires it in.
- `docs/PERSISTENCE.md`: the collection store section notes the interface and the memory implementation.
