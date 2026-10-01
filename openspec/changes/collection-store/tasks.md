## 1. Package and types

- [x] 1.1 Create package `storage`: `KeyType`, `Config`, opaque `RecordRef`, the `Store` and `Cursor` interfaces, and the size limit constants (`PageSize`, `PageHeaderSize`, `SlotSize`, `MaxRecordSize = 4076`)
- [x] 1.2 Error helpers that prefix every message with `storage: `, and use a record's position for batches of more than one

## 2. Memory store

- [x] 2.1 `NewMemoryStore(cfg)` with config validation; the bytes-keeping state (records, index, `@auto` counters starting at 1)
- [x] 2.2 `Insert` in two phases: prepare (key presence and type, duplicates against the index and within the batch, encode, size) then apply (store, index, advance counters); refs in input order
- [x] 2.3 `Lookup`, `Update` (unknown ref, key change or removal, encode, size; unchanged on failure), `Delete` (with its index entry), and `NextAuto`
- [x] 2.4 `Scan` returning a cursor over a snapshot of refs that skips records deleted mid-scan; `Record()` decodes a fresh copy

## 3. Tests

- [x] 3.1 Tests for every scenario in the `collection-store` delta spec
- [x] 3.2 Edge cases: a store without a primary key (inserts and scans work; lookups error), a text key, an empty insert, config validation, and `Next` after the end
- [x] 3.3 A test that a record holding every value type (including nested objects) round-trips through insert, lookup, scan, and update

## 4. Docs and verification

- [x] 4.1 `docs/PERSISTENCE.md`: in the collection store section, note the `Store` interface, `MemoryStore`, refs, the snapshot scan, and the 4076-byte limit
- [x] 4.2 Run `go vet ./...` and `go test ./...`
