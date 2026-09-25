# Query Language Spec

A working spec for the schema and query syntax designed in conversation. Everything here is a design draft, not yet implemented against the storage engine in this repo.

## Philosophy

- Code-like, not English-like. No `SELECT`/`FROM`/`WHERE` sentence-reading. No pluralization games — collection and field names are always singular.
- Symbols carry direction/intent (`<<` out, `>>` in). The one exception is `del`, which is spelled out on purpose — it's the one destructive, irreversible operation, and it shouldn't be one keystroke away from anything else.
- One shared grammar for "what shape of data am I looking at," reused across schema, filters, and projections.

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
| `~> ...` | merge / upsert (match-or-create) |
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

Reuses the same nested-object-literal shape as reads return:

```
>> user => {id: 1, name: "Matt", age: 39, address: {street: "123 Main St", city: "Minneapolis"}}
```

Batch form — separate with &:

```
>> user => 
  {id: 1, name: "Matt", address: {city: "Minneapolis"}} &
  {id: 2, name: "Sam", address: {city: "St. Paul"}}
```

**Returning a value from the write** — second `=>` projects the result, same operator, new meaning in context:

```
>> user => {name: "Matt", age: 39} => {id}
```

### Merge / Upsert

Match first; update if found, create if not. Deliberately a different symbol from plain insert — insert should fail on a duplicate key, merge is the explicit opt-in for "overwrite if it's already there."

```
~> user(id: 1) => {name: "Matt", age: 39}
```

### Delete

Simple form — sugar for a match-then-delete pipeline:

```
!> user(id: 1)
```

Full pipeline form — what the sugar expands to, and the escape hatch for conditions too complex for a bare filter:

```
<< user(address.city: "Minneapolis") => u | delete(u)
```

Return what was deleted, same `=>` convention as insert:

```
!> user(id: 1) => {id, name}
```

---

## Open questions / not yet decided

- Physical storage: whether a `@collection` nested inline is stored colocated with its parent (for locality) or fully separately. Logically it's the same either way — this is an optimization decision, not a semantics one.
- Ceremony around unbounded bulk deletes (e.g. `del user(address.city: "Minneapolis")` with no id) — should this require something extra before it runs?
- Full grammar for joins across two independently-queried collections, beyond the declared-relation traversal case.
