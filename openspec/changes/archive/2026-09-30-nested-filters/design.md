## Context

After `embedded-objects` and `reject-duplicate-keys`:
- `parseFilter` reads `key: value` pairs, where keys may be dotted and values come from the shared `parseValue`, so a nested literal becomes an `Entity`. Repeated keys are rejected.
- `validateFilter` rejects any key touching an object field, then calls `validateFields`.
- `recordMatchesFilter` compares `record[key] != want`, which would panic on two maps; the guard is what prevents that today.
- `resolvePath(record, path)` walks dotted segments and returns `nil` when any step is missing.
- A stored record never holds `nil`: absent means no value, at every depth.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Dotted-path and subset filters with the agreed null semantics, in read, delete, and merge.
- Filter values validated at their path, with full-path errors.

**Non-Goals:**
- Merge payloads that set objects (deep merge).
- Operators other than equality (`!=`, `<`, and so on).
- Rejecting contradictory but distinct conditions like `(address: null address.city: "MSP")`. That never matches, but it isn't the same field given twice. See Risks.

## Decisions

1. **`{*}` is a sentinel, and it's only parsed in a filter.** The parser gets a `inFilter bool` that `parseFilter` sets while it reads values, including nested literals. In `parseValue`, `{` followed by `*` and `}` produces a `wildcardObject{}` value only when `inFilter`. Otherwise it's a parse error: `{*} is only allowed in a filter`. Anything else after `*` (`{*, city: "MSP"}`) is a parse error, the same as in projections.

2. **Resolving a key's path in the schema.** A new helper walks a dotted key through `Schema.Data` / `Schema.Objects` and returns the field's `DataType`, its block's schema, and `optionalAlong`: whether the field or any block above it is optional. A missing segment, or stepping through a scalar, gives `field "<key>" not found in schema for collection "<c>"`.

3. **Validating a filter value** against the resolved field (`filterValueProblem`):
   - `nil`: allowed if `optionalAlong` (the field is optional, or a block above it is); otherwise `field "<path>" is required and cannot be null`.
   - `wildcardObject`: allowed only on an object field; otherwise `field "<path>": expected <type>, got {*}`.
   - `Entity`: allowed only on an object field (`expected <type>, got object`). An empty `Entity` is `empty object filter for field "<path>"; use {*} to match any value`. Otherwise each key inside is validated recursively against the block's schema (undeclared fields, types, null, nested objects). Missing keys are fine: they just aren't constrained. Inside a subset, `null` is allowed only if that field itself is optional, since the object is asserted present.
   - Scalar on an object field: `field "<path>": expected object, got <type>`. Otherwise `validateValue` checks it as usual.

   `validateFilter` checks keys in sorted order and returns the first problem (filters report their first problem, as today).

4. **One condition per field, across forms.** After parsing, `parseFilter` expands each condition to the leaf-level paths it constrains: a scalar or `null` value constrains its key, a subset object constrains `key.` plus each of its keys (recursively), and `{*}` constrains its key. If any path is constrained twice, the filter is rejected with `field "<path>" is given more than once`. Different fields of the same object (`(address.city: "A" address: {street: "B"})`) don't collide.

5. **Matching.** `recordMatchesFilter` becomes: for each key, `actual := resolvePath(record, key)`, then `matches(actual, want)`:
   - `want` is `wildcardObject`: `actual` is an `Entity`
   - `want` is an `Entity`: `actual` is an `Entity`, and `matches(actual[k], w)` for every `k, w` in `want`
   - otherwise: `actual` is not an `Entity` and `actual == want` (so `nil == nil` matches absence)

   After validation, a scalar `want` is never compared against an `Entity`. The explicit `not an Entity` check keeps the comparison safe regardless, since it never compares two maps.

6. **Delete and merge get it through the shared filter path**, since both use `validateFilter` and `recordMatchesFilter`.

## Risks / Trade-offs

- [`(address: null address.city: "MSP")` is accepted and can never match] → Distinct paths, so not a duplicate. Rejecting contradictions in general is a bigger feature. Accepted for now.
- [`{*}` means "any value" in a filter but "every field" in a projection] → Both read as "don't make me list them". Documented side by side.
