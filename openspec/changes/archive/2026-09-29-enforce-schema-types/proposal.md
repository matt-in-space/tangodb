## Why

Schemas declare a type for every field, but nothing enforces them. `>> user {age: "abc"}` into an `int` field is accepted, fields the schema never declared are stored silently (so a typo like `nmae` vanishes without a signal), and `bool` fields can be declared but never written, because the parser has no `true`/`false` literals. Filters have a related silent failure: `(price: 10)` against a stored `10.0` compares an int to a float and quietly never matches. The language is meant to be structured and explicit; this makes the schema actually mean something, and it's the foundation the upcoming nested-object work hooks into.

## What Changes

- Add boolean literals: `true` and `false` parse as `bool` values wherever a value is expected (record literals, filters, merge payloads).
- Add one schema validator that every value passes through before it's used: insert records, merge payloads, and filter values (read, delete, merge).
- A value's type must match its field's declared type **exactly** — no implicit conversions. A literal's type comes from its syntax: `10` is an `int`, `10.0` is a `float`, so `10` into a `float` field is an error.
- A field the collection's schema doesn't declare is an error — for inserts and merge payloads, not just filters and projections as today.
- Validation is all-or-nothing and runs before any write: if anything in a statement is invalid, the whole statement fails and nothing is stored or changed.

## Capabilities

### New Capabilities
- `schema-enforcement`: every value written or filtered on must match its field's declared type exactly, undeclared fields are rejected, and an invalid statement changes nothing.

### Modified Capabilities
- `query-syntax`: adds boolean literals to the value syntax.

## Impact

- New `core/validate.go` holding the validator; `core/parse_insert.go` (`parseValue`) gains the boolean case.
- `core/insert.go`, `core/merge.go`, `core/read.go`, `core/delete.go` route their record/payload/filter values through the validator.
- Tests that build operations directly in Go with untyped `int` constants (e.g. `Entity{"age": 39}`) switch to `int64`, the type the parser actually produces.
- Docs: `README.md` and `QUERY_LANGUAGE.md` describe type enforcement and boolean literals.
