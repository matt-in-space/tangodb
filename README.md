# 💃🏻 TangoDB 💃🏻

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
> >> user {name: "Matt"}
{map[id:1 name:Matt]}
> >> user {name: "Sam"}
{map[id:2 name:Sam]}
> >> user {id: 99, name: "nope"}
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
> >> user {id: 1, name: "Matt"}
{map[id:1 name:Matt]}
```

The record literal comes directly after the collection name — no `=>` before it. That keeps `=>`'s meaning consistent across the whole language: it always means "the shape of what comes back," the same job it does in a read's projection. `(...)`, in turn, always means "identify an existing record" (a read/delete/merge filter) — never "here are values for a new one." Insert has no existing record to identify, so it doesn't use `(...)` at all.

Field values can be a quoted string (`"Matt"`, with `\"` and `\\` supported as escapes) or a number — a plain integer (`39`) or a decimal (`9.99`). Fields are separated by commas, with an optional trailing comma before the closing `}`.

Inserting into a collection with a declared `@id` field enforces it: the record must include that field, and a duplicate value is rejected rather than overwritten:

```
> >> user {id: 1}
{map[id:1]}
> >> user {id: 1}
error: duplicate primary key 1 for collection "user"
```

Inserting into a collection that hasn't been defined is also an error:

```
> >> ghost {id: 1}
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

`;` also works after a filter with no projection (`<< user(name: "Matt");`), and is harmlessly tolerated as an optional trailing marker at the end of any statement (`>> user {id: 1};`, `user { id: int };`) — it's never required except to resolve this one ambiguity.

## Deleting records

```
> !> user(name: "Matt");
1 deleted
```

Delete uses `!>`, the same filter syntax as read — but unlike read, **the filter parens are always required, even when empty**. There's no bare `!> user;` shorthand for "delete everything," the way `<< user;` works for read: forgetting a filter is the single most common way to accidentally wipe out an entire collection, so the parens can't be silently skipped. To genuinely delete every record in a collection, say so explicitly with empty parens:

```
> !> user();
1 deleted
```

Omitting the parens entirely is a hard error, not a shortcut:

```
> !> user;
error: delete requires an explicit filter, e.g. !> user() to match everything
```

Deleting with a filter that matches nothing is a no-op, not an error:

```
> !> user(id: 99);
0 deleted
```

By default, delete returns only a count. Add `=> {...}` to also get the deleted records back, limited to the projected fields — this is opt-in, unlike insert and read, where a return shape is either implicit or required:

```
> !> user(id: 1);
1 deleted
> !> user(id: 1) => {id, name};
id  name
1   Matt
```

## Merging records

```
> ~> user(id: 1) {name: "Matt"};
1 updated
```

Merge (`~>`) bulk-updates every record matching a filter by merging new field values into each match — it's a **partial** update, so any existing field not named in the payload is left untouched:

```
> << user;
age  id  name
40   1   Matt
40   2   Pat
```

(`age` above wasn't touched by the merge — only `name` was in the payload.)

Like delete, **the filter parens are always required, even when empty** — there's no bare `~> user {...}` shorthand, since merge can update an entire collection at once just as easily as delete can remove one:

```
> ~> user() {status: "inactive"};
1 updated
> ~> user {name: "Matt"};
error: merge requires an explicit filter, e.g. ~> user() to match everything
```

Unlike delete, there's no create path — a merge whose filter matches nothing is a no-op, not an insert:

```
> ~> user(id: 99) {name: "Matt"};
0 updated
```

The payload can't include the collection's declared primary key field, whether or not it's `@auto` — the primary key is something you filter *on*, never something a merge payload sets (a bulk update could otherwise assign the same key to multiple rows at once):

```
> ~> user(name: "Sam") {id: 5};
error: payload must not set primary key "id" for collection "user"
```

Same count-vs-`RETURNING` result as delete: a count by default, or the updated records (reflecting their state *after* the merge) with `=> {...}`:

```
> ~> user(id: 1) {name: "Matt"};
1 updated
> ~> user(id: 1) {name: "Matt"} => {id, name};
id  name
1   Matt
```

## Status

The REPL currently supports defining a collection, inserting a flat record, reading records back with a basic equality filter and projection (including "select everything" via `;`), deleting records with a mandatory filter and an optional count-vs-returning result, and merging (bulk partial-updating) records with the same mandatory-filter and count-vs-returning shape. Not yet supported: nested/embedded values (in records, filters, or projections), the write's return-projection clause (`=> {id}`), batch inserts (`&`), and pipeline stages like `order`/`limit` — see `QUERY_LANGUAGE.md` for the full target language.
