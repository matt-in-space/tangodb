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

**`@auto`** makes an `int` field auto-increment, starting at `1`. It's most often used on the primary key, but works on any `int` field:

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

`@auto` and `@id` are independent: `@id` is the record's identity, `@auto` means the database assigns the value. So a collection can use a key you choose alongside a generated sequence number, and a collection can have more than one `@auto` field — each keeps its own counter:

```
> ticket { code: text @id number: int @auto title: text }
{ticket {
  code: text @id
  number: int @auto
  title: text
}}
> >> ticket {code: "A" title: "first"}
{map[code:A number:1 title:first]}
> >> ticket {code: "B" title: "second"}
{map[code:B number:2 title:second]}
```

The database is the only thing that ever writes an `@auto` field — insert can't supply it (not even as `null`), and a merge payload can't set it:

```
> ~> ticket(code: "A") {number: 9};
error: payload must not set auto-increment field "number" for collection "ticket"
```

That's what keeps the values unique without needing an index. Counter values are never reused, and an insert that fails for any reason doesn't use one up. Since an `@auto` field always has a value, it can't be `@optional`.

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

Field values can be a quoted string (`"Matt"`, with `\"` and `\\` supported as escapes), a number — a plain integer (`39`) or a decimal (`9.99`) — or a boolean (`true` / `false`, unquoted; `"true"` is text).

### Types are enforced

Every value must match its field's declared type **exactly**, and a literal's type comes from how it's written: `39` is an `int`, `9.99` is a `float`, `"Matt"` is `text`, `true` is a `bool`. There are no conversions — not even integer to float, so a `float` field takes `10.0`, never `10`:

```
> item { id: int @id price: float }
{item {
  id: int @id
  price: float
}}
> >> item {id: 1 price: 10}
error: field "price": expected float, got int
> >> item {id: 1 price: 10.0}
{map[id:1 price:10]}
```

A field the schema doesn't declare is rejected too, so a typo can't silently vanish:

```
> >> user {id: 1 nmae: "Matt"}
error: field "nmae" not found in schema for collection "user"
```

The same rules apply everywhere a value appears — insert records, merge payloads, and filters on read, delete, and merge. A filter like `(price: 10)` on a `float` field is an error, not a query that quietly matches nothing.

Validation is all-or-nothing: the whole statement is checked before anything is written, so one bad field fails the entire insert, and a bulk merge with a bad payload changes no records at all.

### Required and optional fields

Every declared field is **required** unless it's marked `@optional`. An insert that leaves out a required field fails:

```
> >> user {id: 2}
error: field "name" is required for collection "user"
```

`@optional` marks a field that's allowed to have no value. "No value" is written `null` (unquoted; `"null"` is just text), and leaving an optional field out of an insert means exactly the same thing as writing `null` for it:

```
> profile { id: int @id name: text nickname: text @optional }
{profile {
  id: int @id
  name: text
  nickname: text @optional
}}
> >> profile {id: 1 name: "Matt"}
{map[id:1 name:Matt]}
> >> profile {id: 2 name: "Sam" nickname: "S"}
{map[id:2 name:Sam nickname:S]}
> << profile => {*};
id  name  nickname
1   Matt  null
2   Sam   S
```

A field with no value shows as `null` in a table, not as a blank cell, so it can't be mistaken for an empty string. (The text `"null"` also prints as `null` for now — output formatting will get its own pass later.)

`null` works everywhere a value does, but only for optional fields:

- **Merge** `{nickname: null}` clears the field's value.
- **Filter** `(nickname: null)` matches records where the field has no value — plain equality, so `null` equals `null`.
- On a required field, `null` is an error in an insert, a merge payload, or a filter (a `null` filter on a required field could never match anything):

```
> << profile(name: null);
error: field "name" is required and cannot be null
```

A primary key always has a value, so `@id` and `@optional` together are an error:

```
> x { id: int @id @optional }
error: @id field "id" cannot be @optional
```

Commas between fields are optional — here and everywhere else a list appears (schema blocks, filters, projections). Use them when they make a one-liner easier to read, or leave them out, especially across multiple lines. These are all the same insert:

```
>> user {id: 1, name: "Matt"}

>> user {id: 1 name: "Matt"}

>> user {
  id: 1
  name: "Matt"
}
```

A comma inside a quoted string is part of the value, not a separator (`"Smith, Matt"`).

Inserting into a collection with a declared `@id` field enforces it: the record must include that field, and a duplicate value is rejected rather than overwritten:

```
> >> user {id: 1 name: "Matt"}
{map[id:1 name:Matt]}
> >> user {id: 1 name: "Matt"}
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

**Wildcard** — `*` alone in the projection means every field currently in the schema, without having to name them:

```
> << user(name: "Matt") => {*}
id  name
1   Matt
```

`*` must be the only thing in the braces — combining it with named fields (`{*, id}` or `{id, *}`) is a parse error, not silently merged.

**Omitting the projection entirely returns a count, not records:**

```
> << user(id: 1);
1
```

This is the same "count by default, `=>` opts into records" convention delete and merge use below — `<< user(id: 1) => {*};` is the explicit way to get the record itself back instead of just knowing it exists.

### Selecting everything

`<< collection` with no filter or projection is a valid statement on its own — but it's *also* a valid prefix of a longer one (`<< user(id: 1) => {...}`). Since the REPL submits the moment something parses successfully, it needs to know which you mean. Structurally, the safe default is to keep waiting — a bare `<< user` alone assumes more might still be coming, the same as any other unclosed statement:

```
> << user
... 
```

Add `;` to say "no, that's everything" explicitly — it returns a count of every record in the collection:

```
> << user;
2
```

Combine it with `=> {*}` to get the records themselves, not just how many there are:

```
> << user => {*};
id  name
1   Matt
2   Sam
```

`;` also works after a filter with no projection (`<< user(name: "Matt");`), and is harmlessly tolerated as an optional trailing marker at the end of any statement (`>> user {id: 1};`, `user { id: int };`) — it's never required except to resolve this one ambiguity.

## Deleting records

```
> !> user(name: "Matt");
1
```

Delete uses `!>`, the same filter syntax as read — but unlike read, **the filter parens are always required, even when empty**. There's no bare `!> user;` shorthand for "delete everything," the way `<< user;` works for read: forgetting a filter is the single most common way to accidentally wipe out an entire collection, so the parens can't be silently skipped. To genuinely delete every record in a collection, say so explicitly with empty parens:

```
> !> user();
1
```

Omitting the parens entirely is a hard error, not a shortcut:

```
> !> user;
error: delete requires an explicit filter, e.g. !> user() to match everything
```

Deleting with a filter that matches nothing is a no-op, not an error:

```
> !> user(id: 99);
0
```

By default, delete returns only a bare-integer count. Add `=> {...}` (or `=> {*}` for every field) to also get the deleted records back, limited to the projected fields — this is opt-in, unlike insert and read, where a return shape is either implicit or required:

```
> !> user(id: 1);
1
> !> user(id: 1) => {id, name};
id  name
1   Matt
```

## Merging records

```
> ~> user(id: 1) {name: "Matt"};
1
```

Merge (`~>`) bulk-updates every record matching a filter by merging new field values into each match — it's a **partial** update, so any existing field not named in the payload is left untouched:

```
> << user => {*};
age  id  name
40   1   Matt
40   2   Pat
```

(`age` above wasn't touched by the merge — only `name` was in the payload.)

Like delete, **the filter parens are always required, even when empty** — there's no bare `~> user {...}` shorthand, since merge can update an entire collection at once just as easily as delete can remove one:

```
> ~> user() {age: 41};
1
> ~> user {name: "Matt"};
error: merge requires an explicit filter, e.g. ~> user() to match everything
```

Unlike delete, there's no create path — a merge whose filter matches nothing is a no-op, not an insert:

```
> ~> user(id: 99) {name: "Matt"};
0
```

The payload can't include the collection's declared primary key field, whether or not it's `@auto` — the primary key is something you filter *on*, never something a merge payload sets (a bulk update could otherwise assign the same key to multiple rows at once):

```
> ~> user(name: "Sam") {id: 5};
error: payload must not set primary key "id" for collection "user"
```

Same count-vs-`RETURNING` result as delete: a bare-integer count by default, or the updated records (reflecting their state *after* the merge) with `=> {...}` or `=> {*}`:

```
> ~> user(id: 1) {name: "Matt"};
1
> ~> user(id: 1) {name: "Matt"} => {id, name};
id  name
1   Matt
```

## Status

The REPL currently supports defining a collection, inserting a flat record, reading records back with a basic equality filter and projection (a `*` wildcard for every field, or omit the projection entirely for just a count), deleting records with a mandatory filter and the same count-vs-returning shape, and merging (bulk partial-updating) records the same way. Not yet supported: nested/embedded values (in records, filters, or projections), the write's return-projection clause (`=> {id}`), batch inserts (`&`), and pipeline stages like `order`/`limit` — see `QUERY_LANGUAGE.md` for the full target language.
