# tangodb

A database written in Go, with its own query language. Still early — this README grows alongside what's actually implemented.

## Running the REPL

```
go run .
```

This starts an interactive prompt (`>`) backed by a single in-memory database that lives for the session. It reads a statement, parses it, runs it against that database, and prints the result (or an error). Enter starts a new line rather than submitting — the REPL keeps reading until what you've typed forms a complete statement, then submits it automatically. While a statement is still incomplete, the prompt switches to a continuation prompt (`... `).

Exit with Ctrl+D.

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

## Status

The REPL currently supports defining a collection and inserting a flat record into one. Not yet supported: nested/embedded values in a record, the write's return-projection clause (`=> {id}`), batch inserts (`&`), and the rest of the query language (read, merge, delete) described in `QUERY_LANGUAGE.md`.
