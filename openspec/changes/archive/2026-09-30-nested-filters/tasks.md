## 1. Parsing

- [x] 1.1 Parser `inFilter` state set by `parseFilter`; `parseValue` turns `{*}` into a `wildcardObject` only in filters, and otherwise rejects it with `{*} is only allowed in a filter`; `{*` followed by anything but `}` is a parse error
- [x] 1.2 `parseFilter`: expand conditions to the paths they constrain and reject overlaps with `field "<path>" is given more than once`
- [x] 1.3 Parser tests: `{*}` in a filter, `{*}` in an insert and a merge payload, `{* city: ...}`, the overlap cases

## 2. Validation

- [x] 2.1 Path resolution helper returning the field's type, its block schema, and whether it's optional along the path
- [x] 2.2 `validateFilter`: replace the "not supported yet" guard with path-aware validation per the design (null, `{*}`, subset objects recursively, empty object error, shapes, types)
- [x] 2.3 Tests for every scenario in the `schema-enforcement` delta, plus the empty object error

## 3. Matching

- [x] 3.1 `recordMatchesFilter`: resolve each key's path, then match by subset for objects, presence for `{*}`, and equality otherwise, never comparing two maps
- [x] 3.2 Tests for every scenario in the `query-syntax` delta, in read, delete, and merge
- [x] 3.3 Remove or rewrite the tests that expected "filtering on embedded object field ... is not supported yet"

## 4. Docs and verification

- [x] 4.1 README embedded-objects section: filtering (dotted, subset, null forms, `{*}`, the `{}` error) with real REPL output, and update "not supported yet"; `QUERY_LANGUAGE.md`: update the "implemented so far" note
- [x] 4.2 Run `go vet ./...` and `go test ./...`
