## Why

A key given twice is silently resolved to its last value. That applies to a record literal (`{name: "A" name: "B"}`), a filter (`(id: 1 id: 2)`), and a schema block (`user { id: int id: text }`). It's quiet data loss of exactly the kind the language's explicit-over-implicit rule is meant to prevent: a typo or a copy-paste slip changes what's stored, filtered on, or declared, with no signal. Nested filters are about to make keys more varied, so this should be an error first.

## What Changes

- A field given more than once in a record literal, at any depth, is an error: `field "name" is given more than once`. Nested repeats name the full path: `field "address.city" is given more than once`. This applies to inserts (every record of a batch) and merge payloads.
- A field given more than once in a filter is an error, with the same message.
- A field declared more than once in a collection definition, at any depth, is an error: `field "id" is declared more than once`.
- These are parse errors: the statement is rejected and nothing runs.
- Unchanged: a column named twice in a projection is still shown once (projections choose columns; they don't assign values).

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `query-syntax`: a field can be given only once in a record literal or a filter.
- `collection-definition`: a field can be declared only once.

## Impact

- `core/parse_insert.go`: `parseRecordLiteral` tracks keys it has seen and takes the literal's path prefix for messages.
- `core/parse_read.go`: `parseFilter` tracks keys it has seen.
- `core/parse_define_collection.go`: `parseFieldBlock` tracks names it has seen.
- Tests; README notes the rule.
