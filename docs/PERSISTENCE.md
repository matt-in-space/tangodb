# Persistence: working notes

A running list of ideas and open questions for TangoDB's storage engine. **Decided** marks a choice that's been agreed on. **Leaning** marks the current suggestion for a simple starting point, not a commitment. Everything else is still open, and all of it may change as the design firms up.

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

- **Decided: pages with in-place updates**, rather than an append-only log.
  - An append-only log (Bitcask, and LSM trees in LevelDB and RocksDB) makes writes cheap but needs an index and compaction to read well. It suits write-heavy workloads.
  - In-place pages (the B-tree family: SQLite, Postgres, InnoDB) give data a home location that indexes can point at. That suits a read-heavy workload.
- **Why fixed-size pages:**
  - **Addressing is arithmetic:** page `n` lives at byte `n * pageSize`, so any pointer (a B+tree child, a record's RID) is just a page number.
  - **Space is interchangeable:** any free page can become any kind of page, so there's no file-level fragmentation to manage.
  - **Caching is simple:** the buffer pool's frames are all the same size, so any page fits any frame.
  - **B+tree nodes are pages:** page size sets how many keys fit in a node, and so how shallow the tree is.
- **Why match the OS and disk:**
  - Disks, SSDs, and the OS page cache all move data in whole blocks (512 B to 16 KB), so reading 1 byte costs a whole block anyway.
  - Aligned pages avoid wasted reads and read-modify-write of partial sectors.
  - A write larger than a sector isn't guaranteed to be atomic, so a crash can tear a page. That's why pages carry a checksum.
- **Decided: 4 KB pages, configurable,** so tests can use tiny pages (e.g. 256 bytes) to force splits and overflows early.
  - Bigger pages mean shallower trees and room for bigger records, but more wasted reading and rewriting for single-record work. SQLite uses 4 KB, Postgres 8 KB, InnoDB 16 KB.
- **Decided: a directory per database, with one file per collection and one per index.**
  - For example: `mydb/catalog`, `mydb/user.data`, `mydb/user.id.index`.
  - Simpler to inspect and debug than one big file.
- **Page ids:** a page's number is its position in the file.
- **Leaning: read and write pages with `ReadAt` / `WriteAt`** at `pageID * pageSize`, rather than memory-mapping the file (`mmap`).
  - `mmap` hands paging to the OS, which makes controlling when data reaches disk (crash safety) much harder. See the paper "Are You Sure You Want to Use MMAP in Your Database Management System?".
  - Doing it by hand is also the point: it's where the buffer pool gets built.
- **Only pages move between disk and memory, never whole files.** Opening a large collection reads its header page, then only the pages a query touches.
- **One statement usually writes several pages.** An insert touches a data page and an index leaf; it may also touch a parent index page (if the leaf splits) and the collection's header page (counters). That's why crash safety needs the write-ahead log (section 9).

### Page header (leaning, illustrative)

Every page starts with the same small header, so any page can be identified and checked before anything else is trusted:

```
  offset  size  field
  0       1     page type     file header, catalog, index internal, index leaf, data, ...
  1       1     level         index pages: 0 = leaf, 1+ = levels above it
  2       2     count         entries (index) or slots (data)
  4       4     checksum      CRC32 of the rest of the page
  8       4     link          internal: leftmost child; leaf: next leaf; data: unused
  12      2     free end      where free space ends (data pages)
  14      2     reserved
```

- Every page type shares this one layout, so some fields only matter on some pages: free end is only used by data pages (index pages know where their entries stop from the count).
- **Reserved bytes are deliberate spare room** for fields added later. A likely one is the page's position in the write-ahead log ("page LSN"), which lets recovery tell whether a page already includes a logged change.
- Page headers are little-endian (Go's `binary.LittleEndian`).
- Index *keys* use an order-preserving encoding instead (see section 6).
- A file's own header page adds a magic number (e.g. `TNGO`), the format version, and the page size. The version lets old files simply be refused, since there are no compatibility concerns.

### Byte order

- **Little-endian** (least significant byte first) for page headers and stored values. It's what x86 and ARM use, so values read straight into Go numbers. 3994 (`0x0F9A`) is stored as `9A 0F`.
- **Big-endian** (most significant byte first) for B+tree keys, because byte-by-byte comparison of big-endian numbers matches numeric order:

  ```
    value   little-endian    big-endian
    1       01 00            00 01
    256     00 01            01 00
    byte-compare little-endian says 1 > 256 (wrong); big-endian says 1 < 256 (right)
  ```

- **Signed keys also flip the top bit** (`0x80`), so negatives sort before positives: -1 becomes `7F FF ... FF`, 0 becomes `80 00 ... 00`, 42 becomes `80 00 ... 2A`. Then `bytes.Compare` gives numeric order.
- **The same number, two encodings:** as an index key, 42 is `80 00 00 00 00 00 00 2A`; as a stored value, it's `2A 00 00 00 00 00 00 00`. The index needs sortable bytes, and the record just needs decodable ones.
- **The rule that matters: each field has exactly one byte order, and every reader and writer uses it.** Reading a little-endian `9A 0F` as big-endian gives 39439 instead of 3994, with no error, and a checksum won't catch it. Encode/decode round-trip tests will.
- Go: `binary.LittleEndian` / `binary.BigEndian` (`PutUint16`, `Uint16`, and so on).

### Pages in memory (leaning)

- **Keep the raw 4096 bytes as the in-memory page, and read them through typed views,** one per page kind: `DataPage`, `LeafPage`, `InternalPage`, and so on, each a `[]byte` with methods that read and write fields at their offsets.
  - There's no decode/encode step: writing a page back is writing the same bytes.
  - A page can never outgrow its size; an insert that doesn't fit simply fails, which signals a split or move.
  - This is how SQLite, Postgres, and InnoDB work.
- The alternative, decoding each page into a Go struct and re-encoding it on write, is easier to read. But it costs a copy each way, and a struct can grow past a page without noticing.
- **Records** are what get "hydrated": decoded from a data page's bytes into an `Entity` when a query reads them, and encoded back when written. Everything above the record layer keeps working with `Entity`.

## 2. Encoding records (serialization)

**Decided, and implemented in `storage/record`** (`Encode` / `Decode`). The code doesn't use Go's built-in marshalling; every byte is laid out by hand.

- **How fields are named, three options considered:**
  - **A. names in every record:** self-describing, decodes without the schema, readable in a hex dump
  - **B. field ids** (like Protocol Buffers): compact, renames are free, but decoding needs the catalog
  - **C. position in the schema** (like Postgres): smallest, but every record is tied to a schema version
  - **Decided: A**, for simplicity and debuggability. The version byte at the front of every record leaves room to switch to B later.
- **Layout:**

  ```
    record   version (1 byte, 0x01) | kind (1 byte) | object
    object   field count (uvarint) | fields, sorted by name
    field    name length (1 byte, 1-255) | name (UTF-8) | tag (1 byte) | payload
  ```

  - **kind:** `0x00` record. `0x01` (forwarding stub, for a record that moved) and `0x02` (overflow) are reserved.
- **Value tags:**

  | Tag | Type | Payload |
  |---|---|---|
  | `0x01` | int | zigzag varint |
  | `0x02` | float | 8 bytes, IEEE 754 bits, little-endian |
  | `0x03` | text | uvarint byte length, then UTF-8 bytes |
  | `0x04` / `0x05` | false / true | none |
  | `0x06` | object | uvarint byte length, then a nested object |
  | `0x07` / `0x08` | list / reference | reserved |

- **No null tag:** a field with no value isn't written, and decoding never produces `nil`.
- **Nested objects are stored nested.** An inner field stores only its own name (`street`), inside its parent object's bytes; the path `address.street` is implied by structure. Dotted paths stay a query-language idea. Storing them flattened would repeat prefixes, lose object boundaries, make an empty object (`{address: {}}`) look the same as no object, and couldn't represent lists.
- **Objects are length-prefixed,** so a reader can skip one it doesn't need. That makes a future partial decode (`DecodeField(b, "address.city")`) possible without building the whole record.
- **Varints, written by hand:** 7 bits per byte, least significant group first, with the top bit meaning "more bytes follow". 42 is `2A`, and 300 is `AC 02`. Ints use zigzag first (`(n << 1) ^ (n >> 63)`: 0->0, -1->1, 1->2), so small negatives stay small. Tested against `encoding/binary`.
- **Canonical encoding:** fields are sorted by name, and varints must be minimal, so a record has exactly one encoding. `Decode(Encode(e)) == e`, and `Encode(Decode(b)) == b`.
- **Example:** `{id: 42 name: "Matt"}` is 19 bytes:

  ```
    01 00 02  02 69 64 01 54  04 6E 61 6D 65 03 04 4D 61 74 74
    version, kind, 2 fields | "id" int 42 | "name" text "Matt"
  ```

- **Defensive decoding:** every read is bounds-checked, and any malformed input returns a `*DecodeError` naming the byte offset (`record: unknown value tag 0x09 at byte 5`), never a panic. That covers:
  - truncation; unknown versions, kinds, or tags
  - lengths past the end, and empty names
  - invalid UTF-8, and NaN or infinite floats (which the language can't produce)
  - non-minimal varints, and varints that overflow 64 bits
  - out-of-order or duplicate fields
  - object lengths that don't match their contents
  - nesting past the limit, and trailing bytes
- **Limits:**
  - field names at most 255 bytes (one length byte)
  - nesting at most 32 levels (protects the decoder from unbounded recursion)

  Both are also enforced when a collection is defined, so every valid record can be stored.
- **Testing:**
  - golden tests pin exact bytes
  - round trips
  - one test per malformed case
  - every truncation of every golden record must fail cleanly
  - a fuzz test (`go test -fuzz=FuzzDecode ./storage/record`): any bytes that decode must re-encode identically. An initial 45-second run tried about 17 million inputs with no failures.
- **`core.Entity` is a type alias of `record.Entity`,** so a nested object is the same type in both packages. A type switch in another package only matches the exact named type, so without the alias, nested objects would fail to encode.
- **Checksums** belong to pages (section 1), not records: a slot's length already bounds the record.

## 3. Records in pages (including variable-length updates)

- **Slotted pages** are the standard layout: a slot array grows from the front, record bytes grow from the back, and the free space sits in the middle.
  - A record is addressed by **RID = (page, slot)**.

### How a slotted page works

```
  +--------+----------+----------+----------+- - free - -+------+------+--------+
  | header | slot 0   | slot 1   | slot 2   |            | rec2 | rec1 | rec0   |
  |        | off, len | off, len | off, len |            |      |      |        |
  +--------+----------+----------+----------+- - - - - - +------+------+--------+
  byte 0   16         20         24         28           ^ freeEnd               4095
```

- **The slot array is a fixed-size directory to variable-size records.** Every slot entry is 4 bytes (a 2-byte offset and a 2-byte length), so slot `n` is always at byte `16 + n*4`, while the record it points to can be any length, anywhere in the page.
  - It's the same idea as a book's table of contents: every line has the same shape, even though the chapters vary in length.
- **The two regions grow toward each other.** An insert adds a 4-byte slot entry on the left and the record's bytes on the right, just below the previous record. You never decide up front how much space goes to slots versus records: many small records give a long slot array, a few big ones give a short one.
  - free space = `freeEnd - (16 + slotCount*4)`
  - an insert needs `len(record) + 4` bytes
- **Looking up a record:** RID `(17, 3)` means read page 17, then slot 3's entry at byte `16 + 3*4 = 28`, which gives the offset and length; then read those bytes.
- **Why the extra hop through the slot?** The slot number stays fixed while the bytes move:
  - compacting the page slides records around, and only their slot offsets change
  - a record that grows but still fits is rewritten elsewhere in the page, and only its slot entry changes
  - a record that grows and doesn't fit moves to another page, leaving a forwarding stub in its slot

  In every case, index entries pointing at `(page, slot)` stay valid. If indexes stored byte offsets, every move would mean finding and updating them.
- **Deletes keep the numbering stable:** the slot is marked empty (a tombstone, e.g. offset 0 and length 0) rather than shifting later slots down, which would change their RIDs. An empty slot can be reused once nothing points to it; a delete removes its index entries in the same operation.
- **Holes:** deletes, and records that moved, leave gaps in the record area that aren't part of the free space in the middle. Compaction slides the live records together to reclaim them, updating slot offsets but never slot numbers.
- **A record that grows when updated** (for example, a longer text value):
  - If it still fits in the page's free space, rewrite it in place, compacting the page if needed.
  - If it doesn't fit, **move** it to a page with room and leave a **forwarding stub** in the old slot pointing to the new RID.
  - Forwarding keeps RIDs stable, so indexes pointing at the old RID don't all need updating.
- **Reserving space up front ("fill factor"):** leave, say, 10–20% of each page free on insert, so growth usually fits in place.
  - It's cheap and effective.
  - **Leaning:** start with forwarding only, and add a fill factor later.
- **Records bigger than a page:**
  1. **A size limit:** a record must fit in a page (minus header and slot overhead), and a bigger one fails with a clear error, e.g. `record is 5120 bytes; the maximum is 4040`. Many databases have a maximum row size.
  2. **Overflow pages:** store what fits, and chain the rest through overflow pages, each pointing to the next (SQLite does this).
  3. **Out-of-line values:** move big values (a long text, eventually a big list) to separate storage and leave a pointer (Postgres calls this TOAST, and may compress them).
  - A related rule: make sure a page holds at least a few records (SQLite guarantees 4), so a huge record doesn't wreck B+tree fan-out. That's why big values get pushed out of line.
  - **Leaning:** start with the size limit, and leave room in the record header for an overflow pointer. Nested objects, and later lists, are what make records big, so this will matter.

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

### How the B+tree works

- **It's always balanced: every leaf is at the same depth.** The tree never grows in the middle of a branch; it only gets taller at the root.
- **Levels are counted up from the leaves** (leaf = 0), so **a node's level never changes** once it's created. Counting down from the root would mean renumbering every node whenever the tree got taller.
- **Internal nodes:** a node with `n` keys has `n + 1` children. The keys are dividers between children. The leftmost child lives in the page header's link field, and each entry after it is a key plus the child to its right.

  ```
    leftmost: page 8 | 100 -> page 9 | 200 -> page 12 | 300 -> page 15

    value 42   -> below 100          -> page 8
    value 150  -> 100 <= v < 200     -> page 9
    value 300  -> >= 300             -> page 15   (equal goes right)
  ```

  Finding the right child is a binary search over the keys; the header's count gives the upper bound.
- **Leaves:** sorted key-to-RID entries, found by binary search. If the key isn't there, the record doesn't exist, and no data page is ever read. Each leaf links to the next leaf (the header's link field), so a range scan finds its first key, then walks sideways.
- **Inserting:**
  1. Find the leaf the key belongs in, and insert it there.
  2. If the leaf is full, it splits into two leaves *at the same level*, and the parent gets one more divider key.
  3. If the parent is full too, it splits the same way, pushing a divider to its parent. This can ripple up one level at a time, but every new node matches the level of the node it split from.
  4. **Only a root split makes the tree taller:** with no parent to push into, a new root is created at `old level + 1`. Nothing below is renumbered. The index file's header page records which page is the root, and that pointer is updated.
- **Deleting** is the mirror image: underfull nodes can merge with or borrow from siblings, and the tree only gets shorter at the root (a root left with one child is replaced by that child). **Leaning:** skip merging at first.
- **Capacity (4 KB pages, 8-byte keys):**
  - an internal node fits `(4096 - 16) / 12 = 340` entries (8-byte key + 4-byte child)
  - a leaf fits about `4080 / 14 = 290` entries (8-byte key + 6-byte RID)
  - the header's count is 2 bytes for this reason: one byte only counts to 255

  | Levels | Holds up to about |
  |---|---|
  | 1 | 290 keys (a single leaf) |
  | 2 | 84,000 |
  | 3 | 24 million |
  | 4 | 7 billion |

  Nearly every lookup is 3 or 4 page reads, and the top levels are almost always already in the buffer pool.
- **The level byte** is mostly for checks and convenience (knowing the next page down is a leaf, debugging). The page type byte alone would be enough to navigate.
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

**Decided: split the metadata by how often it changes.**

| Kind | Examples | Changes | Stored in |
|---|---|---|---|
| **Definitions** | collection names, fields and types, nested blocks, `@id` / `@auto` / `@optional`, which indexes exist | rarely (defining a collection) | the database-wide **catalog** |
| **Running state** | `nextID`, `@auto` counters, page count, free-space hint, each index's root page | potentially every insert | **page 0 of each collection's (and index's) own file** |

- Keeping the counters out of the catalog means an insert doesn't rewrite the catalog. They change in the same batch of page writes as the insert that bumped them, which is what the write-ahead log will make atomic.
- Keeping the definitions in one catalog means listing collections doesn't open every file, and cross-collection information (e.g. `@collection` relationships) has a home.
- A collection file's page 0 holds: magic + version + page type, the collection name (a sanity check), `nextID`, `@auto` counters, page count, and a free-space hint.

**Decided: how the catalog itself is stored, in two steps:**

1. **First, a simple file replaced atomically:** write `catalog.tmp`, fsync it, then rename it over `catalog`. A rename either happens completely or not at all, so a crash never leaves a half-written catalog. Readable, easy to inspect, and plenty for something that rarely changes (e.g. JSON).
2. **Later, the catalog as a built-in collection** (e.g. `_collections`, with a hard-coded schema), stored with the engine's own pages and records, the way SQLite keeps `sqlite_schema` and Postgres keeps `pg_class` / `pg_attribute`. It gets paging, crash safety, and the write-ahead log for free, and makes the catalog the engine's first real user. Only where the catalog lives changes, not what's in it.

**Why records store field names (section 2) matters here:** with names, the schema is needed to *validate* records but not to *decode* them, so records are readable on their own. Storing fields by position would make every record unreadable without its schema, forcing the catalog to load first and schema changes to be versioned.

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

## Worked example: one lookup, byte by byte

Illustrative only: these are the layouts sketched above, not a final format.

```
user { id: int @id name: text };
>> user {id: 41 name: "Sam"} & {id: 42 name: "Matt"};
<< user(id: 42) => {*};
```

```
  catalog:  user's primary index root = page 3
  page 3  (index internal):  42 < 100           -> page 8
  page 8  (index leaf):      binary search 42   -> RID (17, 3)
  page 17 (data):            slot 3             -> offset 3994, length 26 -> decode
  result:   {id: 42 name: "Matt"}
```

**Page 3, index internal (the root):**

```
  offset  bytes                     meaning
  0       02                        page type: index internal
  1       01                        level 1
  2-3     01 00                     1 key (so 2 children)
  4-7     ?? ?? ?? ??               checksum
  8-11    08 00 00 00               leftmost child: page 8 (keys < 100)
  16-23   80 00 00 00 00 00 00 64   key 100 (big-endian, sign bit flipped)
  24-27   09 00 00 00               child: page 9 (keys >= 100)
  28-     00 ...                    free space
```

**Page 8, index leaf:**

```
  offset  bytes                     meaning
  0       03                        page type: index leaf
  1       00                        level 0
  2-3     02 00                     2 entries
  8-11    0C 00 00 00               next leaf: page 12
  16-23   80 00 00 00 00 00 00 29   key 41
  24-29   11 00 00 00  02 00        -> page 17, slot 2
  30-37   80 00 00 00 00 00 00 2A   key 42
  38-43   11 00 00 00  03 00        -> page 17, slot 3
```

**Page 17, data page:**

```
  offset  bytes                     meaning
  0       04                        page type: data
  2-3     04 00                     4 slots
  12-13   9A 0F                     free end: 3994
  16-19   E5 0F 1B 00               slot 0: offset 4069, 27 bytes
  20-23   CD 0F 18 00               slot 1: offset 4045, 24 bytes
  24-27   B4 0F 19 00               slot 2: offset 4020, 25 bytes (Sam)
  28-31   9A 0F 1A 00               slot 3: offset 3994, 26 bytes (Matt)
  32-3993 00 ...                    free space

  record at 3994 (26 bytes), with field names and type tags:
  02 00                             2 fields
  02 69 64  01  2A 00 00 00 00 00 00 00      "id", int, 42 (little-endian value)
  04 6E 61 6D 65  03  04 00  4D 61 74 74     "name", text, length 4, "Matt"
```

Because the record carries its own field names and type tags, it decodes without the schema. The schema is for validation.

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
