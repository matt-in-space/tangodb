# tangodb

A database written in Go, with its own query language. Still early — this README grows alongside what's actually implemented.

## Running the REPL

```
go run .
```

This starts an interactive prompt (`>`) backed by a single in-memory database that lives for the session. It reads a statement, parses it, runs it against that database, and prints the result (or an error). Enter starts a new line rather than submitting — the REPL keeps reading until what you've typed forms a complete statement, then submits it automatically. While a statement is still incomplete, the prompt switches to a continuation prompt (`... `).

Exit with Ctrl+D, or by typing `exit` (case-insensitive) on its own. `exit` is a REPL command, not part of the query language — it isn't run against the database.

## Declaring a collection

A collection is a named block of typed fields:

```
user {
  id: int @id
  name: text
}
```

Type it into the REPL across as many lines as you like — it submits as soon as the closing `}` is read:

```
> user {
...   id: int @id
...   name: text
... }
{user {
  id: int @id
  name: text
}}
```

**Types:** `int`, `float`, `text`, `bool` (case-insensitive).

**`@id`** marks a field as the collection's primary key. It's optional — a collection can be defined with no primary key at all. At most one field may be marked `@id`.

**`@auto`** makes an `int @id` field auto-increment, starting at `1`. It can only be used alongside `@id`, and only on an `int` field:

```
> user { id: int @id @auto name: text }
{user {
  id: int @id @auto
  name: text
}}
```

With `@auto`, insert must *not* supply that field — the database assigns it and hands the value back in the result:

```
> >> user => {name: "Matt"}
{map[id:1 name:Matt]}
> >> user => {name: "Sam"}
{map[id:2 name:Sam]}
> >> user => {id: 99, name: "nope"}
error: field "id" is auto-increment and must not be supplied for collection "user"
```

A collection can also be written on a single line:

```
> user { name: text }
{user {
  name: text
}}
```

## Inserting a record

```
> >> user => {id: 1, name: "Matt"}
{map[id:1 name:Matt]}
```

Field values can be a quoted string (`"Matt"`, with `\"` and `\\` supported as escapes) or a number — a plain integer (`39`) or a decimal (`9.99`). Fields are separated by commas, with an optional trailing comma before the closing `}`.

Inserting into a collection with a declared `@id` field enforces it: the record must include that field, and a duplicate value is rejected rather than overwritten:

```
> >> user => {id: 1}
{map[id:1]}
> >> user => {id: 1}
error: duplicate primary key 1 for collection "user"
```

Inserting into a collection that hasn't been defined is also an error:

```
> >> ghost => {id: 1}
error: collection "ghost" does not exist
```

## Querying records

```
> << user(name: "Matt") => {id, name}
id  name
1   Matt
```

The filter in `(...)` matches on equality, and can hold zero or more comma-separated `field: value` conditions — all of them must match (there's no `or` yet). An empty filter (`()`) matches every record in the collection:

```
> << user() => {id, name}
id  name
1   Matt
2   Sam
```

The projection (`=> {...}`) picks which fields to show, and controls both the columns and their order in the printed table. Filtering or projecting on a field the collection doesn't declare is an error, same as an unknown collection:

```
> << user(nope: 1) => {id}
error: field "nope" not found in schema for collection "user"
```

A read that matches nothing prints a plain message rather than an empty table:

```
> << user(id: 99) => {id}
no records found
```

The filter parens are optional when a projection follows directly — `<< user => {id, name}` works the same as `<< user() => {id, name}`.

### Selecting everything

`<< collection` with no filter or projection is a valid statement on its own — but it's *also* a valid prefix of a longer one (`<< user(id: 1) => {...}`). Since the REPL submits the moment something parses successfully, it needs to know which you mean. Structurally, the safe default is to keep waiting — a bare `<< user` alone assumes more might still be coming, the same as any other unclosed statement:

```
> << user
... 
```

Add `;` to say "no, that's everything" explicitly — it returns every record with every schema field:

```
> << user;
id  name
1   Matt
2   Sam
```

`;` also works after a filter with no projection (`<< user(name: "Matt");`), and is harmlessly tolerated as an optional trailing marker at the end of any statement (`>> user => {id: 1};`, `user { id: int };`) — it's never required except to resolve this one ambiguity.

## Status

The REPL currently supports defining a collection, inserting a flat record, and reading records back with a basic equality filter and projection (including "select everything" via `;`). Not yet supported: nested/embedded values (in records, filters, or projections), the write's return-projection clause (`=> {id}`), batch inserts (`&`), pipeline stages like `order`/`limit`, and merge/delete — see `QUERY_LANGUAGE.md` for the full target language.
