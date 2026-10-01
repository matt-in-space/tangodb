## Context

`core.Collection` mixes schema (fields, types, optional fields, nested blocks, the primary key field) with storage state:
- `records map[uint64]Entity`
- `nextID uint64`
- `primaryIndex map[any]uint64`
- `autoCounters map[string]int64`, which also serves as the list of which fields are `@auto`

Read, delete, and merge all collect and sort every id, then walk them. Delete removes records while walking.

`docs/PERSISTENCE.md` places a **collection store** between the query layer and the access methods. It uses the record layer to turn entities into bytes, and keeps a collection's heap and indexes in step. `storage/record` already provides `Encode` / `Decode`. `core.Entity` is an alias of `record.Entity`.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- A store interface that the memory implementation now, and a page-backed implementation later, both satisfy, without `core` noticing the difference.
- The record layer on every write and read.
- The same all-or-nothing and size-limit behavior the page-backed store will have.

**Non-Goals:**
- Changing `core` (that's the follow-up, `wire-collection-store`).
- Pages, the heap, the B+tree, the buffer pool, or files.
- Result ordering.
- An order-preserving key encoding (the memory index doesn't need one; it arrives with the B+tree).

## Decisions

1. **Package `storage`**, depending only on `storage/record`, so `core` can import it with no cycle.

2. **The store gets configuration, not the schema:**
   ```go
   type KeyType int
   const (
   	NoKey KeyType = iota
   	IntKey
   	TextKey
   )

   type Config struct {
   	Collection string   // for messages
   	PrimaryKey string   // "" for no primary key
   	KeyType    KeyType  // IntKey or TextKey when PrimaryKey is set
   	AutoFields []string // @auto fields, each starting at 1
   }
   ```
   `NewMemoryStore(cfg Config) (*MemoryStore, error)` rejects a primary key without a key type (or with `NoKey`), and a key type without a primary key. The schema stays on `core.Collection`.

3. **The interface:**
   ```go
   type Store interface {
   	Insert(records []record.Entity) ([]RecordRef, error)
   	Lookup(key any) (RecordRef, record.Entity, bool, error)
   	Scan() (Cursor, error)
   	Update(ref RecordRef, rec record.Entity) error
   	Delete(ref RecordRef) error
   	NextAuto(field string) (int64, error)
   }

   type Cursor interface {
   	Next() bool             // advance; false at the end or on an error
   	Ref() RecordRef
   	Record() record.Entity  // a decoded, independent copy
   	Err() error
   	Close() error
   }
   ```
   Go's `sql.Rows` uses the same cursor shape.

4. **`RecordRef` is opaque:** `type RecordRef struct{ v uint64 }`. The memory store puts its internal id in `v`. The page-backed store will pack a RID into it: a 32-bit page and a 16-bit slot fit in 48 bits. It's comparable, so callers can use it as a map key, but they can't build or inspect one.

5. **The memory store keeps bytes.**
   ```go
   type MemoryStore struct {
   	cfg     Config
   	records map[uint64][]byte   // internal id -> encoded record
   	nextID  uint64
   	index   map[any]uint64      // primary key value -> internal id
   	auto    map[string]int64    // @auto field -> next value
   }
   ```
   Every write calls `record.Encode`, and every read calls `record.Decode`. Because reads decode fresh bytes, every record returned is an independent copy.

6. **Insert is all-or-nothing, in two phases:**
   1. **Prepare:** for each record, check that the primary key is present and of the configured type, check it against the index and against the earlier records in the batch, encode it, and check its size. Any failure returns an error and stores nothing.
   2. **Apply:** store every record, index it, and advance each `@auto` counter to one past the highest value inserted for that field.

   Refs are returned in input order.

7. **`@auto`:**
   - `NextAuto(field)` returns the counter's current value, the next one to use. It starts at 1, and an unknown field is an error.
   - `core` assigns values (that's a language rule) and passes them in the records.
   - Counters only move when an insert succeeds, so a failed insert never consumes a value.

8. **Size limit:**
   ```go
   const (
   	PageSize       = 4096
   	PageHeaderSize = 16
   	SlotSize       = 4
   	MaxRecordSize  = PageSize - PageHeaderSize - SlotSize // 4076
   )
   ```
   The limit is derived from the page format, so the memory store and the future page-backed store agree by construction.

9. **`Update`:**
   - **Errors:** an unknown ref; a record that's missing the primary key, has a different key, or holds a key of the wrong type (the index would go stale, and merge never changes keys); and any encode or size problem.
   - **On success:** it replaces the stored bytes.
   - **If any check fails:** the stored record is unchanged.

10. **`Delete`:** removes the record and its index entry. An unknown ref is an error.

11. **`Lookup`:**
    - With no primary key configured, it's an error.
    - A key of the wrong Go type (not `int64` for `IntKey`, not `string` for `TextKey`) is an error.
    - A missing key returns `found = false` with no error.

12. **Scans:**
    - **The cursor works from a snapshot of the refs taken when the scan starts.** Each record that existed then is visited once, unless it's deleted before the cursor reaches it, in which case it's skipped. Records inserted during the scan aren't visited.
    - **No order is promised.** The memory store happens to iterate by internal id, and callers mustn't rely on that.
    - Callers (delete, merge) should still collect refs first, then change records, since the page-backed store may not snapshot.

13. **Errors:**
    - Storage errors are prefixed `storage: `, e.g. `storage: record 3 is 5120 bytes; the maximum is 4076`, or `storage: duplicate primary key 42 in collection "user"`.
    - In a batch of more than one record, problems name the record's position, following `core`'s convention; a single record gets no position:
      - most problems are prefixed `record N: ` (`storage: record 2: duplicate primary key 1 in collection "user"`)
      - the size problem reads as a sentence: `storage: record 3 is 5120 bytes; the maximum is 4076` (or `storage: record is 4077 bytes; ...` on its own)
    - An encode error is wrapped: `storage: cannot encode record: <the record layer's error>`.

## Risks / Trade-offs

- [Encoding and decoding on every operation is slower than keeping `Entity` values in maps] → Fine for an in-memory step, and it's what makes the record layer real now, rather than first exercised when files arrive.
- [The snapshot scan is more forgiving than a page-backed cursor may be] → The contract tells callers to collect refs first, and the follow-up change makes delete and merge do so.
- [The store re-checks duplicate keys that `core` already checks] → It's a deliberate backstop that keeps the index consistent no matter who calls it. `core`'s own check runs first, so users still see language-level messages.
