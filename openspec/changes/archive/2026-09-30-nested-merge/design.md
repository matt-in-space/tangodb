## Context

- **Payload parsing:** `parseMerge` parses the payload with `parseRecordLiteral(path)`, whose keys must be plain names (`expectPlainName`). Duplicate keys are already rejected.
- **Payload validation:** `validatePayload` rejects any object field, then calls `validateFields`: first problem only, with `null` allowed on optional fields.
- **Applying:** `db.merge` validates the filter and payload once, checks the primary-key and `@auto` rules, then writes the payload into each matched record in place (`record[field] = value`, or `delete` for `null`). That's safe only because every record ends up the same way.
- **Existing helpers:**
  - `resolveFilterPath` walks a dotted key through the schema.
  - `objectProblems` / `recordProblems` validate a whole record against the schema and collect every problem.
  - `stripNulls` removes nulls at every depth.
  - `constrainedPaths` expands a filter condition to the paths it touches.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Deep merge into embedded objects, through nested or dotted payload keys.
- A merge leaves every matched record valid, or changes nothing.

**Non-Goals:**
- A "replace this object" syntax. Clear with `null`, then set.
- Dotted keys in inserts.
- One-to-many (`[]`) or `@collection`.

## Decisions

1. **Payload context allows dotted keys.** The parser gets `inPayload bool`, set by `parseMerge` around the payload (parallel to `inFilter`). While it's set, `parseRecordLiteral` reads keys with `expectIdent` instead of `expectPlainName`, so they may be dotted, at any depth of the payload. Outside it, inserts keep rejecting dotted names.

2. **Normalize the payload into one nested object first.** Before validation, the payload is rewritten so every dotted key becomes nested: `{address.city: "STP"}` becomes `{address: {city: "STP"}}`, merging with any sibling object for the same prefix. While rewriting, two conflicts are detected:
   - the same path set twice gives `field "<path>" is given more than once`
   - a path set to `null` (or a scalar) while a path beneath it is also set gives `field "<child>" conflicts with "<parent>"`

   After this step, the rest of merge only ever sees plain nested objects.

3. **Payload validation, before any record is looked at.** `validatePayload` keeps its primary-key and `@auto` checks at the top level. It then walks the normalized payload against the schema:
   - unknown fields: not found (full path)
   - scalars: `validateValue`
   - an object on a scalar field, or a scalar on an object field: shape errors
   - `null`: only on an optional field
   - an empty object: `empty object in merge payload for field "<path>"`
   - `{*}`: already a parse error outside filters

   Missing fields aren't errors here: a partial update doesn't have to name them. First problem wins, as payload errors do today.

4. **Deep merge.** `deepMerge(stored, payload Entity) Entity` returns a new record:
   - it copies `stored`, then for each payload key:
     - `null` deletes the key
     - an object merges recursively into the stored object at that key (starting from an empty object if none is there)
     - anything else is set
   - every object it writes is a fresh copy, so no record ever shares a map with the payload or with another record.

5. **Compute, validate, then commit.**
   - `db.merge` first builds the merged copy of every matched record (in id order, the same order `=>` rows use).
   - It then validates each copy with the existing whole-record check (`objectProblems` against the root schema; the primary key and `@auto` fields can't have changed).
   - Problems are prefixed with `record with id <key>: ` when the collection has a primary key, otherwise `matched record <n>: ` (1-based, in match order).
   - One problem gives that line alone; several give `<N> problems, nothing changed:` followed by an indented list, like batch insert.
   - Only if there are no problems are the copies stored in place of the originals. The primary index doesn't change, since merge can't touch the key.

6. **Top-level merges go through the same path.** A payload with only scalar fields deep-merges trivially, so there's one code path. Existing merge behavior and messages stay the same, since top-level payload problems are still caught first by `validatePayload`.

## Risks / Trade-offs

- [Merging copies every matched record before writing] → Fine for an in-memory learning database, and it's what makes all-or-nothing possible when outcomes differ per record.
- [Payload errors report the first problem, while result errors report all of them] → Payload problems are about the statement itself. Result problems can span many records, where seeing all of them matters. This matches the split between merge's existing checks and batch insert.
