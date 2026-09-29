## 1. Declaration

- [x] 1.1 Parser: allow `@auto` without `@id`, keep the `int` check, add the `@auto @optional` error; collect auto fields into `AutoFields map[string]bool` on `DefineCollectionOperation`
- [x] 1.2 `defineCollection`: same rules for operations built in Go; replace `autoIncrement` and `nextAutoValue` on `Collection` with `autoCounters map[string]int64`, each starting at 1
- [x] 1.3 `Collection.String()`: print `@id`, `@auto`, `@optional` in that order on any field
- [x] 1.4 Tests: `@auto` on a non-id field, on a non-int field, several `@auto` fields, `@auto @optional`; flip the old "`@auto` requires `@id`" tests

## 2. Insert

- [x] 2.1 Reject any supplied auto field (including `null`) before validation; have `validateRequired` skip every auto field
- [x] 2.2 Assign every auto field's counter only after all checks pass, in sorted field order
- [x] 2.3 Switch tests from `AutoIncrement: true` to `AutoFields`
- [x] 2.4 Tests: sequential values on a non-id field, independent counters, supplying an auto field, a failed insert (duplicate key) consumes no value

## 3. Merge

- [x] 3.1 Reject payloads setting any auto field, after the primary-key check
- [x] 3.2 Tests: payload sets a non-id auto field, filtering on an auto field

## 4. Docs and verification

- [x] 4.1 Update `README.md` and `QUERY_LANGUAGE.md`: `@auto` on any `int` field, database-owned, not optional
- [x] 4.2 Run `go vet ./...` and `go test ./...`
