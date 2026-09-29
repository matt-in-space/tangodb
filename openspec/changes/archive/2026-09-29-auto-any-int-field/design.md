## Context

Today `@auto` is a single `autoIncrement bool` on `DefineCollectionOperation` and `Collection`, with one `nextAutoValue int64` counter, and it always refers to the primary key. Two places enforce that `@auto` needs `@id` and an `int` type: the parser (`@auto can only be used on the @id field`) and `defineCollection` (`@auto requires a primary key`). `db.insert` assigns the counter *before* the duplicate-key check. That's harmless today, because an auto primary key can't be a duplicate, but it won't be once other fields can be `@auto`.

Assumes `optional-fields` has landed.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- `@auto` on any `int` field, any number of them per collection.
- The database is the only writer of `@auto` fields.
- Counters are consumed only by successful inserts.

**Non-Goals:**
- `@auto` on non-`int` types.
- Resetting or seeding counters.
- A uniqueness index on non-id `@auto` fields. Uniqueness comes from nothing else being able to write them.

## Decisions

1. **`AutoIncrement bool` becomes `AutoFields map[string]bool`** on `DefineCollectionOperation`, matching the `Optional` set from `optional-fields`.

2. **Counters are `autoCounters map[string]int64` on `Collection`**, the next value for each auto field, each starting at `1`. This replaces `autoIncrement` and `nextAutoValue`.

3. **Declaration rules, checked in both the parser and `defineCollection`** as today:
   - an `@auto` field must be declared, and must be `int` (`@auto requires an int field (field %q)`)
   - `@auto @optional` fails with `@auto field %q cannot be @optional`
   - the "requires `@id`" / "requires a primary key" errors are removed

4. **Insert order:**
   1. For every auto field, a key present in the record, even as `null`, fails with the existing `field %q is auto-increment and must not be supplied for collection %q`.
   2. `validateFields`, the primary-key presence and duplicate checks, and `validateRequired` (which skips every auto field, not just the primary key).
   3. Only after all checks pass: assign each auto field its counter's value and increment it, iterating in sorted field order so assignment is deterministic.
   4. Store and index.

   The duplicate-key check for an `@id @auto` field keeps working, because the assigned value is always new.

5. **Merge rejects any auto field in the payload** with `payload must not set auto-increment field %q for collection %q`. The primary-key check runs first, so a field that is both `@id` and `@auto` still reports the primary-key error.

6. **`Collection.String()` prints annotations in a fixed order**, `@id`, `@auto`, `@optional`, on any field that has them.

## Risks / Trade-offs

- [A non-id `@auto` field has no index, so uniqueness depends on the insert and merge rules never having a gap] → Both write paths are covered by explicit requirements and tests. A future write path (such as batch insert) must go through the same checks.
- [Test churn from `AutoIncrement: true`] → Mechanical: those tests switch to `AutoFields: map[string]bool{"id": true}`.
