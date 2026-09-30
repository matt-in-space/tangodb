## 1. Parsing

- [x] 1.1 Parser `inPayload` state set by `parseMerge` around the payload; while set, `parseRecordLiteral` accepts dotted keys at any depth
- [x] 1.2 Payload normalization: rewrite dotted keys into nested objects, rejecting the same path twice (`is given more than once`) and a value set alongside a path beneath it (`conflicts with`)
- [x] 1.3 Parser tests: dotted payload keys (top level and nested), both conflict kinds, dotted keys still rejected in inserts

## 2. Payload validation

- [x] 2.1 `validatePayload`: replace the object guard with path-aware checks of the normalized payload (unknown paths, types, shapes, `null` on required fields, empty objects), keeping the primary-key and `@auto` checks
- [x] 2.2 Tests for every scenario in "Merge payload values are validated at their path", plus the empty object error

## 3. Deep merge and result validation

- [x] 3.1 `deepMerge(stored, payload)` returning a fresh record, copying every object it writes
- [x] 3.2 `db.merge`: build merged copies of all matched records, validate each with the whole-record check, report every problem with `record with id <key>:` / `matched record <n>:`, and commit only when there are none
- [x] 3.3 Tests for every deep-merge, dotted-key, and result-validation scenario in the `merge` delta, the new `schema-enforcement` scenario, and that records never share a nested object after a merge
- [x] 3.4 Confirm existing merge tests and messages are unchanged

## 4. Docs and verification

- [x] 4.1 README: merging into embedded objects (deep merge, `null`, dotted keys, conflicts, result validation) with real REPL output; remove the "not supported yet" note and update the Status line; `QUERY_LANGUAGE.md`: update the merge section and the "implemented so far" note
- [x] 4.2 Run `go vet ./...` and `go test ./...`
