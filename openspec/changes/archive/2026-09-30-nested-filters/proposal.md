## Why

Embedded objects can be declared, inserted, and projected, but not filtered on: any filter touching an object field returns "not supported yet". That leaves nested data stored but not queryable. The two filter forms were designed earlier: dotted paths and subset object filters. Their null semantics are now settled, so this makes them real.

## What Changes

- **Dotted-path filters:** `(address.city: "MSP")` matches when the value at that path equals the filter value. A path that runs through an absent object has no value, so `(address.city: null)` matches records with no city *for any reason*, including no address.
- **Subset object filters:** `(address: {city: "MSP"})` matches when `address` is **present** and every condition inside it matches. Fields not mentioned are unconstrained, and conditions nest. Because the form describes an object, it asserts the object exists:
  - `(address: {city: null})` means "has an address, with no city on it"
  - `(address: null)` means "has no address"
- **`{*}` in a filter** means "present, with any contents": `(address: {*})`. `*` must be alone in the braces.
- **`{}` in a filter is an error.** It's ambiguous (a subset with no conditions, or an object that's exactly empty), so the error points to `{*}`.
- **`{*}` is only allowed in filters.** In an insert record or merge payload it's an error.
- **Validation at the path:** unknown paths, wrong types, and wrong shapes are errors naming the full path. `null` on a path is allowed only if the field, or some block above it, is optional; otherwise it could never match.
- **One condition per field:** a field constrained twice, whether by repeating a dotted key (already an error from `reject-duplicate-keys`) or by a dotted key overlapping a subset filter (`(address.city: "A" address: {city: "B"})`), is `field "address.city" is given more than once`.
- Read, delete, and merge filters all support this. Merge payloads still can't set object fields (out of scope).

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `query-syntax`: removes "Filters on embedded objects are not supported yet"; adds dotted-path filters, subset filters (with `{*}` and the `{}` error), `{*}` only in filters, and one condition per field.
- `schema-enforcement`: adds validating filter values against the schema at their path.

## Impact

- `core/parse_insert.go` / `core/parse_read.go`: `{*}` parses only in filter context; filters flag overlapping conditions.
- `core/validate.go`: `validateFilter` replaces its "not supported yet" guard with path-aware validation.
- `core/read.go`: `recordMatchesFilter` resolves each key's path and matches objects by subset.
- Tests; README's embedded-objects section documents filtering; `QUERY_LANGUAGE.md` updates its "implemented so far" note.
- Depends on `reject-duplicate-keys` (the path-threaded record literal parser and the "given more than once" message).
