# 💃🏻 TangoDB 💃🏻

A database written in Go, with its own query language. Still early — this README grows alongside what's actually implemented.

## Running the REPL

```
go run .
```

This starts an interactive prompt (`tango>`) backed by a single in-memory database that lives for the session. It reads a statement, parses it, runs it against that database, and prints the result (or an error).

**Every statement ends with `;`.** That's the one rule for when something runs: Enter just starts a new line, and the REPL keeps reading (however many lines, blank ones included) until it sees the `;`. While a statement is still in progress, the prompt switches to a continuation prompt (`...>`), padded to line up with `tango>`. If you're sitting at `...>` and expected something to happen, you've probably left off the `;`. One statement at a time: anything after the `;` on the same line is an error. The prompts deliberately avoid the query language's operator characters, so `tango> >> user {...}` can't be misread.

Nothing is parsed until the statement is finished, so a mistake partway through a multi-line statement is reported once, after the `;`, and the rest of the statement is discarded with it (rather than each leftover line being read as a statement of its own):

```
tango> >> user {
  ...>   id: 1
  ...>   name: Matt
  ...>   age: 3
  ...> };
error: expected a value, got "Matt"
tango>
```

A `;` inside a quoted string or inside open brackets doesn't end the statement. An unmatched closing bracket is reported right away.

Exit with Ctrl+D, or by typing `exit` (case-insensitive) on its own — no `;`, since `exit` is a REPL command, not part of the query language; it isn't run against the database.

## Declaring a collection

A collection is a named block of typed fields:

```
user {
  id: int @id
  name: text
};
```

Type it into the REPL across as many lines as you like — like every statement, it runs once the `;` is read:

```
tango> user {
  ...>   id: int @id
  ...>   name: text
  ...> };
user {
  id: int @id
  name: text
}
```

**Types:** `int`, `float`, `text`, `bool` (case-insensitive).

**`@id`** marks a field as the collection's primary key. It's optional — a collection can be defined with no primary key at all. At most one field may be marked `@id`.

**`@auto`** makes an `int` field auto-increment, starting at `1`. It's most often used on the primary key, but works on any `int` field:

```
tango> user { id: int @id @auto name: text };
user {
  id: int @id @auto
  name: text
}
```

With `@auto`, insert must *not* supply that field — the database assigns it. Ask for it back with `=> {id}` (see [Inserting a record](#inserting-a-record)):

```
tango> >> user {name: "Matt"} => {id};
id
1
tango> >> user {name: "Sam"} => {id};
id
2
tango> >> user {id: 99, name: "nope"};
error: field "id" is auto-increment and must not be supplied for collection "user"
```

`@auto` and `@id` are independent: `@id` is the record's identity, `@auto` means the database assigns the value. So a collection can use a key you choose alongside a generated sequence number, and a collection can have more than one `@auto` field — each keeps its own counter:

```
tango> ticket { code: text @id number: int @auto title: text };
ticket {
  code: text @id
  number: int @auto
  title: text
}
tango> >> ticket {code: "A" title: "first"} => {code number};
code  number
A     1
tango> >> ticket {code: "B" title: "second"} => {code number};
code  number
B     2
```

The database is the only thing that ever writes an `@auto` field — insert can't supply it (not even as `null`), and a merge payload can't set it:

```
tango> ~> ticket(code: "A") {number: 9};
error: payload must not set auto-increment field "number" for collection "ticket"
```

That's what keeps the values unique without needing an index. Counter values are never reused, and an insert that fails for any reason doesn't use one up. Since an `@auto` field always has a value, it can't be `@optional`.

A collection can also be written on a single line:

```
tango> user { name: text };
user {
  name: text
}
```

## Inserting a record

```
tango> >> user {id: 1, name: "Matt"};
1
```

An insert returns a bare count — `1` — the same "count by default" rule read, delete, and merge follow (and what a batch insert will report as more than one). To get the stored record back, add a projection with `=>`; it reflects what was actually stored, including any values the database assigned:

```
tango> >> user {id: 2, name: "Sam"} => {*};
id  name
2   Sam
```

The record literal comes directly after the collection name — no `=>` before it. That keeps `=>`'s meaning consistent across the whole language: it always means "the shape of what comes back," the same job it does in a read's projection. `(...)`, in turn, always means "identify an existing record" (a read/delete/merge filter) — never "here are values for a new one." Insert has no existing record to identify, so it doesn't use `(...)` at all.

Field values can be a quoted string (`"Matt"`, with `\"` and `\\` supported as escapes), a number — a plain integer (`39`) or a decimal (`9.99`) — or a boolean (`true` / `false`, unquoted; `"true"` is text).

### Types are enforced

Every value must match its field's declared type **exactly**, and a literal's type comes from how it's written: `39` is an `int`, `9.99` is a `float`, `"Matt"` is `text`, `true` is a `bool`. There are no conversions — not even integer to float, so a `float` field takes `10.0`, never `10`:

```
tango> item { id: int @id price: float };
item {
  id: int @id
  price: float
}
tango> >> item {id: 1 price: 10};
error: field "price": expected float, got int
tango> >> item {id: 1 price: 10.0};
1
```

A field the schema doesn't declare is rejected too, so a typo can't silently vanish:

```
tango> >> user {id: 1 nmae: "Matt"};
error: 2 problems, nothing inserted:
  field "nmae" not found in schema for collection "user"
  field "name" is required for collection "user"
```

An insert reports every problem it finds, not just the first — here, the typo'd field *and* the required field it left missing. When there's only one problem, you get just that line.

The same rules apply everywhere a value appears — insert records, merge payloads, and filters on read, delete, and merge. A filter like `(price: 10)` on a `float` field is an error, not a query that quietly matches nothing.

Validation is all-or-nothing: the whole statement is checked before anything is written, so one bad field fails the entire insert, and a bulk merge with a bad payload changes no records at all.

### Required and optional fields

Every declared field is **required** unless it's marked `@optional`. An insert that leaves out a required field fails:

```
tango> >> user {id: 2};
error: field "name" is required for collection "user"
```

`@optional` marks a field that's allowed to have no value. "No value" is written `null` (unquoted; `"null"` is just text), and leaving an optional field out of an insert means exactly the same thing as writing `null` for it:

```
tango> profile { id: int @id name: text nickname: text @optional };
profile {
  id: int @id
  name: text
  nickname: text @optional
}
tango> >> profile {id: 1 name: "Matt"};
1
tango> >> profile {id: 2 name: "Sam" nickname: "S"};
1
tango> << profile => {*};
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
tango> << profile(name: null);
error: field "name" is required and cannot be null
```

A primary key always has a value, so `@id` and `@optional` together are an error:

```
tango> x { id: int @id @optional };
error: @id field "id" cannot be @optional
```

Commas between fields are optional — here and everywhere else a list appears (schema blocks, filters, projections). Use them when they make a one-liner easier to read, or leave them out, especially across multiple lines. These are all the same insert:

```
>> user {id: 1, name: "Matt"};

>> user {id: 1 name: "Matt"};

>> user {
  id: 1
  name: "Matt"
};
```

A comma inside a quoted string is part of the value, not a separator (`"Smith, Matt"`).

A field can only be given once — in a record, a filter, or a schema block. A repeat is an error rather than silently keeping one of the values:

```
tango> >> user {id: 1 name: "A" name: "B"};
error: field "name" is given more than once
tango> user2 { id: int id: text };
error: field "id" is declared more than once
```

Inserting into a collection with a declared `@id` field enforces it: the record must include that field, and a duplicate value is rejected rather than overwritten:

```
tango> >> user {id: 1 name: "Matt"};
1
tango> >> user {id: 1 name: "Matt"};
error: duplicate primary key 1 for collection "user"
```

Inserting into a collection that hasn't been defined is also an error:

```
tango> >> ghost {id: 1};
error: collection "ghost" does not exist
```

### Inserting several records at once

Separate records with `&` to insert them in one statement. The result is the number of records inserted:

```
tango> >> user {id: 1 name: "Matt"} & {id: 2 name: "Sam"};
2
```

A trailing `&` means more records are coming, so a batch can span lines. A `=>` projection goes after the last record and returns one row per record, in the order you wrote them:

```
tango> >> user {id: 3 name: "Pat"} &
  ...> {id: 4 name: "Ann"} => {id name};
id  name
3   Pat
4   Ann
```

A batch is **all-or-nothing**: every record is checked before any is stored — including duplicate keys *within* the batch — and if anything is wrong, nothing is inserted. Every problem is listed, each tagged with the record it came from:

```
tango> >> user {id: 5 name: "Kim"} & {id: 6} & {id: 5 name: 7};
error: 3 problems, nothing inserted:
  record 2: field "name" is required for collection "user"
  record 3: field "name": expected text, got int
  record 3: duplicate primary key 5 for collection "user"
```

Records in a batch can differ in which optional fields they supply, and `@auto` values are assigned in input order, only once the whole batch is valid.

## Embedded objects

A field's type can be a block of fields of its own: an **embedded object**, stored as part of its parent record. Blocks can nest, and a whole block can be `@optional`:

```
tango> user {
  ...>   id: int @id
  ...>   name: text
  ...>   address: {
  ...>     street: text
  ...>     city: text
  ...>     zip: text @optional
  ...>   } @optional
  ...> };
user {
  address: {
    city: text
    street: text
    zip: text @optional
  } @optional
  id: int @id
  name: text
}
```

An embedded object has no identity of its own, so `@id` and `@auto` aren't allowed inside a block (`@id is not allowed inside an embedded block (field "address.id")`). They may come back when related collections (`@collection`) do.

Insert an object as a nested record literal. Everything inside is validated with the same rules as top-level fields, and problems name the full dotted path. Required fields inside an optional block are only checked when the block is there:

```
tango> >> user {id: 1 name: "Matt" address: {street: "1 Main" city: "MSP"}} & {id: 2 name: "Sam"};
2
tango> >> user {id: 3 name: "Pat" address: {city: 5}};
error: 2 problems, nothing inserted:
  field "address.city": expected text, got int
  field "address.street" is required for collection "user"
```

In a table, an object is **flattened into dotted columns**. `{*}` lists every leaf field, naming an object (`=> {address}`) lists its fields, and a dotted path (`=> {address.city}`) is a column of its own. A field with no value, including every field of an absent optional object, shows `null`:

```
tango> << user => {*};
address.city  address.street  address.zip  id  name
MSP           1 Main          null         1   Matt
null          null            null         2   Sam
tango> << user => {name address.city};
name  address.city
Matt  MSP
Sam   null
```

### Filtering on embedded objects

There are two ways to filter on an embedded object, and they mean slightly different things:

- A **dotted path**, `(address.city: "MSP")`, asks about one value. A path through an object that isn't there has no value, so `(address.zip: null)` matches records with no zip *for any reason*, including no address at all.
- A **subset filter**, `(address: {city: "MSP"})`, describes the object: it matches when the address is **present** and every field you name matches. Fields you don't name don't matter. So `(address: {zip: null})` means "has an address, with no zip on it".

```
tango> user { id: int @id name: text address: { city: text zip: text @optional } @optional };
user {
  address: {
    city: text
    zip: text @optional
  } @optional
  id: int @id
  name: text
}
tango> >> user {id: 1 name: "Matt" address: {city: "MSP" zip: "55401"}} & {id: 2 name: "Sam" address: {city: "MSP"}} & {id: 3 name: "Pat"};
3
tango> << user(address.city: "MSP") => {name};
name
Matt
Sam
tango> << user(address.zip: null) => {name};
name
Sam
Pat
tango> << user(address: {zip: null}) => {name};
name
Sam
```

For the object as a whole, `(address: null)` matches records with no address, and `(address: {*})` matches records that have one, whatever it contains. An empty `{}` is an error, since it's ambiguous between "any address" and "an address with nothing in it":

```
tango> << user(address: null) => {name};
name
Pat
tango> << user(address: {*}) => {name};
name
Matt
Sam
tango> << user(address: {});
error: empty object filter for field "address"; use {*} to match any value
```

Filter values are checked against the schema at their full path (`field "address.city": expected text, got int`), and a field can only be constrained once: `(address.city: "A" address: {city: "B"})` is `field "address.city" is given more than once`. `{*}` only means something in a filter; in an insert record it's an error. All of this works the same in read, delete, and merge filters.

### Merging into embedded objects

A merge into an embedded object is a **deep merge**: it updates the fields you give, at any depth, and leaves every other field alone — the same "partial update" rule merge follows at the top level. You can write the update as a nested object or as a dotted path; they mean the same thing. To remove a value, set it to `null`:

```
tango> ~> user(id: 1) {address: {city: "STP"}} => {*};
address.city  address.street  address.zip  id  name
STP           1 Main          55401        1   Matt
tango> ~> user(id: 1) {address.zip: null} => {*};
address.city  address.street  address.zip  id  name
STP           1 Main          null         1   Matt
```

`{address: null}` clears a whole optional object, and merging into a record that has no address creates one. A payload can't set the same field twice, or clear an object while also setting something inside it:

```
tango> ~> user(id: 1) {address: null address.city: "X"};
error: field "address.city" conflicts with "address"
```

Because a merge can land differently on each record — creating an address on one, updating it on another — the database works out every matched record's result first and checks each against the schema. If any result would be invalid, nothing changes, and each problem names its record:

```
tango> ~> user() {address.city: "MPLS"};
error: record with id 2: field "address.street" is required for collection "user"
```

(Record 2 had no address, so the merge would have created one with only a `city`.) For a collection with no primary key, records are named by position instead: `matched record 2: ...`. Dotted keys are a merge thing only — an insert writes a whole record, so it still takes nested literals.

## Querying records

```
tango> << user(name: "Matt") => {id, name};
id  name
1   Matt
```

The filter in `(...)` matches on equality, and can hold zero or more comma-separated `field: value` conditions — all of them must match (there's no `or` yet). An empty filter (`()`) matches every record in the collection:

```
tango> << user() => {id, name};
id  name
1   Matt
2   Sam
```

The projection (`=> {...}`) picks which fields to show, and controls both the columns and their order in the printed table. Filtering or projecting on a field the collection doesn't declare is an error, same as an unknown collection:

```
tango> << user(nope: 1) => {id};
error: field "nope" not found in schema for collection "user"
```

A read that matches nothing prints a plain message rather than an empty table:

```
tango> << user(id: 99) => {id};
no records found
```

The filter parens are optional when a projection follows directly — `<< user => {id, name}` works the same as `<< user() => {id, name}`.

**Wildcard** — `*` alone in the projection means every field currently in the schema, without having to name them:

```
tango> << user(name: "Matt") => {*};
id  name
1   Matt
```

`*` must be the only thing in the braces — combining it with named fields (`{*, id}` or `{id, *}`) is a parse error, not silently merged.

**Omitting the projection entirely returns a count, not records:**

```
tango> << user(id: 1);
1
```

This is the same "count by default, `=>` opts into records" convention delete and merge use below — `<< user(id: 1) => {*};` is the explicit way to get the record itself back instead of just knowing it exists.

### Reading everything

With no filter at all, a read covers the whole collection — a count by default, or every record with `=> {*}`:

```
tango> << user;
2
tango> << user => {*};
id  name
1   Matt
2   Sam
```

## Deleting records

```
tango> !> user(name: "Matt");
1
```

Delete uses `!>`, the same filter syntax as read — but unlike read, **the filter parens are always required, even when empty**. There's no bare `!> user;` shorthand for "delete everything," the way `<< user;` works for read: forgetting a filter is the single most common way to accidentally wipe out an entire collection, so the parens can't be silently skipped. To genuinely delete every record in a collection, say so explicitly with empty parens:

```
tango> !> user();
1
```

Omitting the parens entirely is a hard error, not a shortcut:

```
tango> !> user;
error: delete requires an explicit filter, e.g. !> user() to match everything
```

Deleting with a filter that matches nothing is a no-op, not an error:

```
tango> !> user(id: 99);
0
```

By default, delete returns only a bare-integer count. Add `=> {...}` (or `=> {*}` for every field) to also get the deleted records back, limited to the projected fields — this is opt-in, unlike insert and read, where a return shape is either implicit or required:

```
tango> !> user(id: 1);
1
tango> !> user(id: 1) => {id, name};
id  name
1   Matt
```

## Merging records

```
tango> ~> user(id: 1) {name: "Matt"};
1
```

Merge (`~>`) bulk-updates every record matching a filter by merging new field values into each match — it's a **partial** update, so any existing field not named in the payload is left untouched:

```
tango> << user => {*};
age  id  name
40   1   Matt
40   2   Pat
```

(`age` above wasn't touched by the merge — only `name` was in the payload.)

Like delete, **the filter parens are always required, even when empty** — there's no bare `~> user {...}` shorthand, since merge can update an entire collection at once just as easily as delete can remove one:

```
tango> ~> user() {age: 41};
1
tango> ~> user {name: "Matt"};
error: merge requires an explicit filter, e.g. ~> user() to match everything
```

Unlike delete, there's no create path — a merge whose filter matches nothing is a no-op, not an insert:

```
tango> ~> user(id: 99) {name: "Matt"};
0
```

The payload can't include the collection's declared primary key field, whether or not it's `@auto` — the primary key is something you filter *on*, never something a merge payload sets (a bulk update could otherwise assign the same key to multiple rows at once):

```
tango> ~> user(name: "Sam") {id: 5};
error: payload must not set primary key "id" for collection "user"
```

Same count-vs-`RETURNING` result as delete: a bare-integer count by default, or the updated records (reflecting their state *after* the merge) with `=> {...}` or `=> {*}`:

```
tango> ~> user(id: 1) {name: "Matt"};
1
tango> ~> user(id: 1) {name: "Matt"} => {id, name};
id  name
1   Matt
```

## Status

The REPL currently supports defining a collection, inserting records, including embedded objects, one at a time or in `&` batches (a count by default, or the stored records with `=>`), reading records back with an equality filter (including dotted paths and subset filters on embedded objects) and projection (a `*` wildcard for every field, dotted columns for embedded objects, or omit the projection entirely for just a count), deleting records with a mandatory filter and the same count-vs-returning shape, and merging (bulk partial-updating) records the same way, including deep merges into embedded objects. Not yet supported: related collections (`@collection`), and pipeline stages like `order`/`limit` — see `QUERY_LANGUAGE.md` for the full target language.
