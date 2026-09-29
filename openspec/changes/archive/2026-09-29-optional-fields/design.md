## Context

`Collection.data` maps field names to a `DataType`, with no notion of required. `validateFields` (from `enforce-schema-types`) checks only the fields a statement mentions, so a missing field is never noticed. `parseValue` has no `null`, and `formatCell` renders a missing field as `""`, which looks the same as an empty string.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Required by default, `@optional` to opt out.
- One concept of "no value", written `null`, usable in inserts, merges, and filters.
- `null` visible in output.

**Non-Goals:**
- Telling the text `"null"` apart from real `null` in tables. Tables print text unquoted, so both show as `null`. Accepted for now; output formatting is a later change.
- Optional nested objects. Nesting will reuse `@optional` when it lands.
- `@auto` changes (next change, `auto-any-int-field`).

## Decisions

1. **"No value" is stored as an absent key.** A record never holds a `nil`. Inserting `{nickname: null}` stores the record without `nickname`, and merging `{nickname: null}` deletes the key. With one internal representation, a filter `(nickname: null)` needs no special case: `record[field]` on a missing key is `nil`, which equals the filter's `nil`.

2. **`null` parses to Go `nil`**, as a third bare word in `parseValue`'s `tokenIdent` case, next to `true` and `false`. The quoted `"null"` stays text.

3. **Optional fields are carried as a set of names**, `Optional map[string]bool` on `DefineCollectionOperation` and `optional map[string]bool` on `Collection`, next to `data`. Changing `data` into a field struct would be cleaner once there are more per-field properties, but it isn't needed yet.

4. **Annotation conflicts are checked in the parser and in `defineCollection`**, matching how `@auto` is handled today, so an operation built directly in Go is held to the same rules. `@id @optional` fails with `@id field %q cannot be @optional`.

5. **`validateFields` handles `nil` before the type check.** A `nil` value on an optional field is valid. On a required field it fails with `field %q is required and cannot be null`. Because filters go through `validateFields`, `(name: null)` on a required field is rejected rather than silently matching nothing.

6. **A new `validateRequired(collection, record)` runs on insert only**, after `validateFields` and before anything is stored. Each required field missing from the record fails with `field %q is required for collection %q`, checked in sorted order for a deterministic error. The `@id @auto` field is skipped, since the database assigns it. For plain `@id`, the existing `record missing primary key` check runs before `validateFields` and treats `null` the same as absent, so `{id: null}` gets that same error. Merge never runs this check: a partial update only touches the fields it names.

7. **Check order on insert and merge.** Checks that give a more specific message run before generic validation:
   - insert: the `@auto` "must not be supplied" check runs before `validateFields`, so `{id: null}` on an auto id reports that it must not be supplied, not that it can't be null.
   - merge: the "payload must not set primary key" check runs before payload validation, so `{id: null}` reports the primary-key rule.

   Everything still runs before any mutation.

8. **Insert strips `null`s** from the record after validation, so no `nil` is ever stored (decision 1).

9. **`formatCell(nil)` returns `"null"`.** A projected field that a record doesn't have shows as `null`.

10. **`Collection.String()` prints `@optional`** after any other annotations.

## Risks / Trade-offs

- [Required by default breaks every test and doc example that leaves out declared fields] → Expected. Each one either supplies the field or marks it `@optional`, whichever the test actually means.
- [The text `"null"` and real `null` look the same in a table] → Accepted for now (see Non-Goals).
