## Why

Embedded objects can be declared, inserted, projected, and filtered, but a merge payload can't touch them at all: any object field is rejected, even `{address: null}`. So there's no way to update an address, fix one field of it, or clear it. Merge is already defined as a partial update, and this change extends that rule into embedded objects.

## What Changes

- **Deep merge:** a payload object merges into the stored object field by field, at every depth. Fields the payload doesn't mention are left alone. Removing a value takes an explicit `null`:
  - `{address: {city: "STP"}}` updates `city` and keeps everything else
  - `{address: {zip: null}}` clears `zip`
  - `{address: null}` clears the whole (optional) address
- **Dotted payload keys:** `{address.city: "STP"}` is the same deep update, written as a path. Dotted keys work anywhere in a merge payload, including inside nested payload objects. Inserts still don't accept them, since an insert writes a whole record.
- **Merging into a missing object creates it**, from the nested or the dotted form.
- **Conflicting keys are errors:**
  - two keys setting the same field: `{address.city: "A" address: {city: "B"}}` gives `field "address.city" is given more than once`
  - clearing an object while setting inside it: `{address: null address.city: "X"}` gives `field "address.city" conflicts with "address"`
- **Payload checks before anything runs:** unknown paths, wrong types, wrong shapes, `null` on a required field, `{}`, and `{*}` are errors naming the full path.
- **Results are validated after the merge, before anything is written.** Every matched record's new state is computed and checked against the schema. If any would be invalid (for example, a newly created address missing its required `street`), nothing changes, and every problem is listed with its record: `record with id 2: ...`, or `matched record 2: ...` for a collection without a primary key.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `merge`: removes "Merge payload cannot set an embedded object yet"; adds deep merge, dotted payload keys and conflicts, and validating results.
- `schema-enforcement`: "An invalid statement changes nothing" gains a scenario where one matched record's result would be invalid.

## Impact

- `core/parse_insert.go` / `core/parse_merge.go`: a payload context in which record-literal keys may be dotted; conflict detection for payloads.
- `core/validate.go`: `validatePayload` validates object values and dotted keys at their paths instead of rejecting objects.
- `core/merge.go`: compute each matched record's merged copy, validate all of them, then commit; deep-merge helper that copies payload objects.
- Tests; README merge and embedded-objects sections; `QUERY_LANGUAGE.md` note.
