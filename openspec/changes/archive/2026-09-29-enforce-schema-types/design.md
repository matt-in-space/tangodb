## Context

The parser produces four value types: `int64` (integer literals), `float64` (decimal literals), `string` (quoted), and — after this change — `bool`. `Collection.data` maps each declared field name to a `DataType` (`TypeInt`, `TypeFloat`, `TypeText`, `TypeBool`). Today nothing connects the two: `db.insert` stores whatever record it's given, `db.merge` merges whatever payload it's given, and `db.read`/`db.delete`/`db.merge` only check that filter *field names* exist, never their values' types. `recordMatchesFilter` compares with `!=` on `any`, so an `int64` filter value against a stored `float64` never matches.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- One validation entry point, shared by insert records, merge payloads, and filter values.
- Exact type matching — no conversions of any kind.
- Designed as the hook for nested objects: the next change adds an `object` case that recurses.

**Non-Goals:**
- Required fields. A record missing a declared field (`>> user {id: 1}` with a `name: text` field) is still accepted; whether fields should be mandatory is a separate decision.
- Nested objects themselves.

## Decisions

1. **Boolean literals in `parseValue`.** `true`/`false` already lex as `tokenIdent`. `parseValue` gains a `tokenIdent` case: `"true"` → `true`, `"false"` → `false`, anything else falls through to the existing "expected a value" error. They're only special in value position, so an identifier named `true` elsewhere (a field name) is unaffected.

2. **A new `core/validate.go` with two functions:**
   ```go
   // validateValue reports whether value's runtime type matches dataType exactly.
   func validateValue(dataType DataType, value any) error

   // validateFields checks every field against the collection's schema: it must
   // be declared, and its value must match the declared type.
   func validateFields(collection *Collection, fields map[string]any) error
   ```
   `validateValue` is a switch on `dataType` checking the concrete Go type (`int64`, `float64`, `string`, `bool`). Its error names both types (`expected float, got int`); `validateFields` prefixes the field name. `validateFields` iterates field names in sorted order so the reported error is deterministic when several fields are bad.

   Nesting will add a `TypeObject` case to `validateValue` that calls `validateFields` on the sub-record — that recursion is the hook.

3. **Validate, don't normalize.** Since no conversions are allowed, the validator only accepts or rejects; values are stored and compared exactly as parsed. This also makes the int-vs-float filter bug impossible rather than patched: a filter value that doesn't match its field's type is an error, so `!=` only ever compares like with like.

4. **Where each call site validates, always before any mutation:**
   - `db.insert`: right after the collection lookup, before the primary-key/auto-increment logic. `Entity` and `map[string]any` share an underlying type, so the record is passed directly.
   - `db.merge`: the filter and the payload are both validated up front, before the existing primary-key-in-payload check and before any record is touched — so a bulk merge can never half-apply.
   - `db.read`, `db.delete`, `db.merge` (filter): `validateFields` replaces each operation's existing "filter field must exist in schema" loop, since it checks both existence and type.

5. **Error wording for undeclared fields stays the existing one** — `field %q not found in schema for collection %q` — so filters, projections, inserts, and payloads all report unknown fields identically.

## Risks / Trade-offs

- [Strictness means `10` can't go into a `float` field — you write `10.0`] → Deliberate (explicit over implicit); trivially relaxed later if it proves annoying.
- [Tests that construct operations in Go with untyped constants (`39` is a Go `int`, not `int64`) will now fail validation] → Expected churn: `int64` is the only integer type the parser produces, so those tests switch to `int64(39)`. The validator does not accept Go `int`, since no parsed input can produce one.
