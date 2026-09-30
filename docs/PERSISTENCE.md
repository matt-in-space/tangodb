# Persistence: working notes

A running list of ideas and open questions for TangoDB's storage engine. Nothing here is decided yet: **Leaning** marks the current suggestion for a simple starting point, not a commitment. It will change as the design firms up.

The goal is to learn how a database actually stores data. So this is deliberately a physical storage engine (pages, B+trees, a buffer pool) rather than just dumping state to a file. The workload is expected to be **read-heavy**.

---

## Layers

Keeping these separate is what protects the lower layers from changes to the data model that are still coming (lists `[]`, `@collection`). The lower layers only ever see opaque bytes, so a new kind of value only changes the record encoder.

```
  query layer        (exists today: parse, validate, filter, project)
       |
  record layer       encode/decode a record <-> bytes   <- lists / @collection only change this layer
       |
  access methods     heap file (records in pages) + B+tree indexes
       |
  buffer pool        fixed number of cached pages, eviction, dirty tracking
       |
  page / file layer  fixed-size pages, page ids, free-space tracking
       |
  disk               (+ write-ahead log for durability)
```

---

## 1. Pages and files

- **Page size:** fixed, typically 4 KB or 8 KB (matching the OS and disk).
  - Make it configurable, so tests can use tiny pages (e.g. 256 bytes) to force splits and overflows early.
- **File layout:** one file per database, or one file per collection plus one per index.
  - **Leaning:** a directory per database, with one file per collection and per index. It's simpler to inspect and debug.
- **Header page:** a magic number, a format version, the page size, and where the catalog is.
  - The version lets old files simply be refused; there are no compatibility concerns.
- **Page ids:** a page's number is its position in the file, so page `n` lives at `n * pageSize`.

## 2. Encoding records (serialization)

- **Self-describing values:** a type tag, then the bytes.
  - int64, float64, text (length + bytes), bool, object (nested record).
  - Lists and references become new tags later.
- **Field names versus positions:**
  - Storing field names in every record wastes space but survives schema changes.
  - Storing by position (the schema's field order) is compact but ties every record to one schema version.
  - **Leaning:** names, for simplicity and easy debugging.
- **Nulls:** "no value" is already an absent field in memory, so nothing is written for it.
- **Checksums** per page (or per record), to catch corruption and torn writes.

## 3. Records in pages (including variable-length updates)

- **Slotted pages** are the standard layout: a slot array grows from the front, record bytes grow from the back, and the free space sits in the middle.
  - A record is addressed by **RID = (page, slot)**.
- **A record that grows when updated** (for example, a longer text value):
  - If it still fits in the page's free space, rewrite it in place, compacting the page if needed.
  - If it doesn't fit, **move** it to a page with room and leave a **forwarding stub** in the old slot pointing to the new RID.
  - Forwarding keeps RIDs stable, so indexes pointing at the old RID don't all need updating.
- **Reserving space up front ("fill factor"):** leave, say, 10–20% of each page free on insert, so growth usually fits in place.
  - It's cheap and effective.
  - **Leaning:** start with forwarding only, and add a fill factor later.
- **Records bigger than a page** need overflow pages chained together.
  - **Leaning:** defer this with a size limit ("a record must fit in a page"), and lift the limit later.

## 4. Deletes, tombstones, and free space

- **Delete** marks the slot empty (a tombstone). Its space is reclaimed when the page is compacted.
- **Finding room for an insert:** a **free-space map** tracks roughly how full each page is.
  - **Leaning:** start by checking the last page, then append a new one.
- **Cleanup:**
  - **Leaning:** compact a page only when an insert or update needs the room, so there's no background work at first.
  - A background "vacuum" can come later.

## 5. Identity and what indexes point at

- There's already a surrogate internal id (`nextID`) and a primary index (primary key to id).
- **What should an index entry store?**
  - **A RID:** fast lookup, and forwarding stubs keep it valid.
  - **The primary key:** another lookup to reach the record, but it survives the record moving.
  - **Leaning:** RIDs, plus forwarding stubs.

## 6. B+tree indexes

- **B+tree rather than a plain B-tree:** all values live in the leaves, and the leaves are linked.
  - That makes range scans cheap, which suits a read-heavy workload and future `order` / `limit` / range filters.
- **One node per page.**
  - Nodes split when they fill.
  - Merging nodes on delete can wait: underfull nodes are fine at first.
- **Order-preserving key encoding:** encode int, float, text, and bool so that plain byte comparison sorts them correctly.
  - A fun sub-problem: negative numbers, floats, the sign bit.
- **Which fields get an index:**
  - The primary key always, for `@id` uniqueness and lookups.
  - Secondary indexes later, maybe through an `@index` annotation, possibly on dotted paths like `address.city`.
- **Maintenance:** every insert, update, and delete updates every index.
  - An update that changes an indexed field removes the old entry and adds the new one.
- **Choosing a plan:** an equality filter on an indexed field uses the index; anything else is a full scan. That's the first, tiny query planner.

## 7. Buffer pool (memory limits and caching)

- **A fixed number of page frames** in memory. This caps how much data is loaded, however big the files get.
- **Page table:** maps a page id to the frame it's in.
- **Pin counts:** a page that's in use can't be evicted.
- **Eviction:** LRU or the clock algorithm. Clock is simpler and cheaper.
- **Dirty pages** are written back when evicted, or at a flush or checkpoint.
- **Reading the same data again:** repeated reads hit memory instead of disk.
  - A cached page is never stale, because every read and write goes through the buffer pool. The pool *is* the current state, and disk catches up.
  - Cache invalidation only gets hard with multiple processes, which is out of scope.

## 8. Durability: when does a write count as done?

- A write shouldn't be reported as successful until it's actually stored.
- **Simplest correct option:** before a statement returns, write its changed pages and **fsync**.
  - Slow, but easy to reason about. **Leaning:** a good first step.
- **Then a write-ahead log (WAL):**
  - Append a log record describing the change, fsync the *log*, and return. Changed pages are flushed later.
  - On startup, **recovery** replays the log.
  - Commits get much cheaper, because appending to one file beats rewriting scattered pages.
- **Checkpoints** flush dirty pages so the log can be truncated.
- **An "async" durability mode** would let the caller choose not to wait for storage.
  - With a WAL it's easy: skip the fsync, and accept losing the last few statements on a crash.
  - Add it after the WAL exists.

## 9. All-or-nothing on disk (needed before transactions)

- The language already promises that **a statement is all-or-nothing**, including batch inserts and bulk merges.
  - A batch insert of 50 records spread over several pages has to survive a crash as all 50 or none. That's really a single-statement transaction.
- A WAL provides this: log the whole statement, then commit it.
- **Known gap until then:** before the WAL exists, a crash partway through a statement can leave partial data. That's acceptable to start with, as long as it's written down.

## 10. Background work and goroutines

- **Leaning:** no background goroutines at first.
  - Everything runs synchronously on the caller's goroutine, one statement at a time.
  - Much easier to debug and test.
- **Later candidates:**
  - a background page flusher and checkpointer (once there's a WAL)
  - vacuum and page compaction (tombstone cleanup)
- **Shutdown** matters once background work exists: flush and close cleanly on `exit` or Ctrl+D, and stop the goroutines.
- **Concurrency:** a single REPL means a single writer. A lock around the database is plenty until there's a server.

## 11. Catalog and metadata

What has to be stored alongside the records, and loaded at startup:

- collection schemas, including nested `Schema`s and annotations
- `@auto` counters
- `nextID` (or its on-disk equivalent)
- each index's root page

**Leaning:** a catalog page, or a small catalog file.

## 12. Startup, shutdown, and naming databases

- **Startup:**
  1. Open or create the database directory.
  2. Check the header and format version.
  3. Run recovery (once there's a WAL).
  4. Load the catalog.
- **Naming and choosing a database:** the REPL needs a way to pick one, e.g. `tangodb mydb` or an `open` command.
  - This ties into possibly changing the REPL prompt from `tango>` once databases have names.

## 13. Testing a storage engine

- Tiny page sizes, to force splits, overflows, and moves.
- **Crash tests:** kill the process mid-write, then reopen and check the data.
- **Encoder round-trip tests:** encoding then decoding a record gives back the same record.
- **Randomized tests** for the B+tree (insert and delete lots of keys; check order and lookups).

## 14. Explicitly later

- transactions and MVCC
- real concurrency control
- compression
- schema changes to existing collections (e.g. adding a field when records already exist)
- overflow pages for large records

---

## Build order

Each step works end to end on its own:

1. **Page file, slotted pages, and record encoding.** Full scans only; synchronous fsync writes; data survives a restart. Replaces today's in-memory maps with pages.
2. **Buffer pool.** A memory cap, eviction, and cached reads.
3. **B+tree primary index.** Replaces the in-memory `primaryIndex`.
4. **Write-ahead log and recovery.** Statements become all-or-nothing on disk, and commits get cheaper.
5. **Secondary indexes**, and using them in filters.
6. **Background flushing and vacuum**, and the async durability option.

A good first question to work through is step 1's record encoding and slotted page layout. It also forces the RID and forwarding decisions (sections 3 and 5).

---

## Related open questions elsewhere

These aren't persistence questions, but they affect what gets stored:

- **Lists (`[]`) of embedded objects:** empty versus absent, filtering on elements, how lists display in tables, merge semantics.
- **`@collection`:** separate storage, identity (`@id` / `@auto` inside the block), parent references, cascading deletes, inserting into both collections at once.

With the layer separation above, both should mostly change the record encoding and the catalog, not pages or indexes.
