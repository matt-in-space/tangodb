# tangodb

A database written in Go, with its own query language. Still early — this README grows alongside what's actually implemented.

## Running the REPL

```
go run .
```

This starts an interactive prompt (`>`) that reads a statement, parses it, and prints the result. Enter starts a new line rather than submitting — the REPL keeps reading until what you've typed forms a complete statement, then submits it automatically. While a statement is still incomplete, the prompt switches to a continuation prompt (`... `).

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
{Name:user Data:map[id:0 name:2] PrimaryKey:id}
```

**Types:** `int`, `float`, `text`, `bool` (case-insensitive).

**`@id`** marks a field as the collection's primary key. It's optional — a collection can be defined with no primary key at all. At most one field may be marked `@id`.

A collection can also be written on a single line:

```
> user { name: text }
{Name:user Data:map[name:2] PrimaryKey:}
```

## Status

The REPL currently only parses collection definitions — it doesn't yet run them against a database or support the rest of the query language (insert, read, delete) described in `QUERY_LANGUAGE.md`. Those operations exist at the Go level (see `insert.go`) but aren't wired into the parser or REPL yet.
