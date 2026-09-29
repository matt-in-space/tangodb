# Query Language Spec

A working spec for the schema and query syntax designed in conversation. Everything here is a design draft, not yet implemented against the storage engine in this repo.

## Philosophy

- Code-like, not English-like. No `SELECT`/`FROM`/`WHERE` sentence-reading. No pluralization games — collection and field names are always singular.
- Symbols carry direction/intent (`<<` out, `>>` in, `!>` destroy). `!` is physically distant from `<`, `>`, `~`, and `=` on a standard keyboard, so the destructive operation isn't one keystroke away from any of the others — a word-form keyword (e.g. `del`) was considered instead but rejected, since it would collide with the "a bare identifier could still become a collection name" ambiguity every schema definition already has to resolve.
- Commas are whitespace. They're optional between entries in every list — schema blocks, record literals, filters, projections — so `{id: 1, name: "Matt"}` and `{id: 1 name: "Matt"}` are the same thing. Use them for one-liners, drop them for multi-line.
- One shared grammar for "what shape of data am I looking at," reused across schema, filters, and projections — including the `*` wildcard, which means "every field currently in the schema" wherever a projection appears (`<< collection(filter) => {*}`, `!> collection(filter) => {*}`, `~> collection(filter) {payload} => {*}`).

---

## Schema Definition

A collection is a named block of typed fields.

```
user {
  id: int @id
  name: text
  age: int
}
```

**Types:** `INT`, `FLOAT`, `TEXT`, `BOOL`

**`@id`** marks a field as the collection's identity/primary key.

### Nested shapes: embedded vs. related

A nested block with no `@collection` annotation is **pure embedding** — a value shape with no identity of its own, no back-reference, and no independent query entry point. It only exists as part of its parent.

```
user {
  id: INT @id
  name: TEXT
  dimensions: { width: FLOAT, height: FLOAT }   // pure shape — only reachable via user.dimensions
}
```

A nested block **with `@collection`** is a real, independently-identified, independently-queryable collection. The annotation's value is the underlying collection name — this is what lets two differently-named fields share one collection (e.g. `billing_address` and `shipping_address` both backed by `address`).

```
user {
  id: INT @id
  name: TEXT
  address: { @collection: address, id: INT @id, street: TEXT, city: TEXT }
}
```

This single declaration wires up:
- `address` as a real, top-level queryable collection
- `user.address` — the field, as written
- `address.user` — the reverse reference, free, because nothing needs pluralizing: the collection name IS the relation name

### Cardinality

Default is **one-to-one**. Append `[]` to the block to make it **one-to-many**:

```
user {
  id: INT @id
  name: TEXT
  address: { @collection: address, id: INT @id, street: TEXT, city: TEXT }[]
}
```

The field name stays singular either way — cardinality lives in `[]`, never in the name. The "many" side always holds the actual reference (`address.user` is always singular); `user.address` as a list is resolved through `address`'s reverse index, not a stored list on `user`.

### Many-to-many

Neither side can hold a single clean reference, so it needs a real join collection — declared explicitly, not auto-generated, so it has somewhere to carry its own data later (timestamps, roles, etc.):

```
user_tag {
  user: { @collection: user, id: INT @id }
  tag: { @collection: tag, id: INT @id }
}
```

---

## Statements

Four kinds, distinguished by leading symbol/keyword:

| Form | Meaning |
|---|---|
| `<< ...` | read |
| `>> ...` | insert |
| `~> ...` | merge (bulk conditional update) |
| `!> ...` | delete |

### Read

```
<< collection(filter) => projection
```

**Filter** — `(field: value)` pairs, dotted paths allowed for nested fields:

```
<< user(id: 1) => {id, name, age}
<< user(address.city: "Minneapolis") => {id, name}
```

**Wildcard** — `*` alone in the projection means every field currently in the schema. It must be the sole content of the braces; combining it with named fields is a parse error:

```
<< user(id: 1) => {*}
```

**Omitting the projection entirely returns a count, not records** — `<< user(id: 1)` (with no `=>` at all) reports how many records matched, without materializing any of them. This is the same "count by default, `=>` opts into records" convention delete and merge already use; `=> {*}` is the explicit way to get everything back instead.

**Bound variables** — bind a name to reach into nested structure explicitly in the projection:

```
<< user(address.city: "Minneapolis") => u => {id: u.id, name: u.name, street: u.address.street}
```

**Querying a related collection directly**, navigating back up via its reverse reference:

```
<< address(city: "Minneapolis") => a => {a.street, a.city, owner: a.user.name}
```

**Pipeline stages** — post-processing after the match, separate from filtering:

```
<< user(address.city: "Minneapolis") => {id, name} | order(name) | limit(20)
```

### Insert

The record literal — the same nested-object-literal shape reads return — comes directly after the collection name, no `=>` in front of it:

```
>> user {id: 1, name: "Matt", age: 39, address: {street: "123 Main St", city: "Minneapolis"}}
```

This keeps `=>` meaning one single thing everywhere in the language: "the shape of what comes back," same job it does in a read's projection. `(...)`, correspondingly, always means "identify an existing record" (a read/delete/merge filter) — insert has nothing to identify, so it doesn't use `(...)` at all.

Batch form — separate with &:

```
>> user
  {id: 1, name: "Matt", address: {city: "Minneapolis"}} &
  {id: 2, name: "Sam", address: {city: "St. Paul"}}
```

**Returning a value from the write** — `=>` projects the result, the same meaning it has everywhere else:

```
>> user {name: "Matt", age: 39} => {id}
```

### Merge

Bulk conditional update — merge the payload's fields into every record matching the filter. There is no create path: a filter matching nothing is a no-op, not an implicit insert. Deliberately a different symbol from plain insert — insert should fail on a duplicate key, merge is the explicit opt-in for "overwrite whatever's already there."

The payload comes directly after the filter, no `=>` in front of it — same grammar shape as insert, for the same reason (`=>` only ever means "shape of what comes back"):

```
~> user(id: 1) {name: "Matt", age: 39}
```

Like delete, the filter parens are **mandatory**, even when empty — merge is a bulk mutating operation, so there's no bare `~> user {...}` shorthand that would apply to a whole collection by omission:

```
~> user() {status: "inactive"}   -- deliberately matches (and updates) everything
```

The update is a **partial merge**, not a full replace — only the fields named in the payload change; anything else already on a matched record is left untouched. Every record matching the filter is updated (no attempt to detect or reject multiple matches — the filter means the same thing here as it does for read and delete). The payload must not include the collection's declared primary key field, whether or not that field is `@auto` — the primary key is something you filter *on*, never something a merge payload sets, since a bulk update could otherwise assign the same key value to more than one row at once.

Return value follows the same count-vs-`RETURNING` convention as delete — and the count is a bare integer, no wording around it:

```
~> user(id: 1) {name: "Matt"}                  -- returns 1 (just the count)
~> user(id: 1) {name: "Matt"} => {id, name}    -- returns the updated record(s) too
~> user(id: 1) {name: "Matt"} => {*}           -- returns the updated record(s), every field
```

### Delete

Simple form — sugar for a match-then-delete pipeline. Unlike read, the filter parens are **mandatory**, even when empty — there's no bare `!> user` shorthand for "delete everything." The risk isn't confusing which operator was typed (the Philosophy section's `!>` keyboard-distance argument covers that); it's the classic "forgot the filter" mistake, which persists no matter how distinct the operator is. Requiring `()` — even empty — forces the filter position to be visibly acknowledged rather than silently skipped:

```
!> user(id: 1)
!> user()          -- deliberately matches (and deletes) everything
```

Full pipeline form — what the sugar expands to, and the escape hatch for conditions too complex for a bare filter:

```
<< user(address.city: "Minneapolis") => u | delete(u)
```

Return what was deleted, same `=>` convention as insert — but unlike insert, this is genuinely optional either way: with no `=>`, delete returns only a bare-integer count (Postgres plain-`DELETE` style); with `=>`, it also returns the deleted records, limited to the projected fields (Postgres `DELETE ... RETURNING` style):

```
!> user(id: 1)                  -- returns 1 (just the count)
!> user(id: 1) => {id, name}    -- returns the deleted record(s) too
!> user(id: 1) => {*}           -- returns the deleted record(s), every field
```

---

## Open questions / not yet decided

- Physical storage: whether a `@collection` nested inline is stored colocated with its parent (for locality) or fully separately. Logically it's the same either way — this is an optimization decision, not a semantics one.
- Full grammar for joins across two independently-queried collections, beyond the declared-relation traversal case.
