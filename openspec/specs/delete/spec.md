## Purpose

Lets a client remove records from a collection that match a filter, using the `!>` statement, while making it structurally hard to delete an entire collection by accident.

## Requirements

### Requirement: Delete statement requires explicit filter parens
The system SHALL require a `(...)` filter clause immediately after the collection name in a `!>` statement, even when the filter is empty. A `!>` statement with no filter parens at all SHALL be rejected as invalid, rather than treated as an implicit "delete everything" shorthand.

#### Scenario: Explicit empty filter matches everything
- **WHEN** a client sends `!> user();`
- **THEN** every record in the `user` collection is deleted

#### Scenario: Omitted filter parens is rejected
- **WHEN** a client sends `!> user;` (no `(...)` at all)
- **THEN** the system returns an error and does not delete anything

### Requirement: Delete removes records matching the filter
The system SHALL delete every record in the named collection whose fields match all of the given `field: value` equality conditions, and SHALL leave non-matching records untouched.

#### Scenario: Filter matches a subset of records
- **WHEN** a client sends `!> user(name: "Matt");` against a collection containing records named "Matt" and "Sam"
- **THEN** the record named "Matt" is deleted and the record named "Sam" remains

### Requirement: Deleting zero matches is a no-op
The system SHALL treat a delete whose filter matches no records as a successful no-op, returning a count of 0, rather than as an error.

#### Scenario: Filter matches nothing
- **WHEN** a client sends `!> user(id: 999);` and no record has `id: 999`
- **THEN** the system returns success with a deleted count of 0 and no records are removed

### Requirement: Delete result defaults to a count
The system SHALL, when a delete statement has no `=>` projection clause, return only the number of records deleted, not the deleted records themselves, displayed as a bare integer with no surrounding words.

#### Scenario: Delete with no projection
- **WHEN** a client sends `!> user(id: 1);` and one record matches
- **THEN** the system returns a deleted count of 1, displayed as the bare text `1` — not `"1 deleted"` or any other wording — without including that record's field values

### Requirement: Delete result can opt into returning deleted records
The system SHALL, when a delete statement includes an `=> {...}` projection clause, return the deleted records themselves, limited to the projected fields, in addition to the count. The projection MAY be a wildcard `*` in place of a field list, meaning every field currently declared in the collection's schema; `*` MUST be the sole content of the projection — combining it with named fields SHALL be rejected as a parse error.

#### Scenario: Delete with a projection
- **WHEN** a client sends `!> user(id: 1) => {id, name};` and the matching record is `{id: 1, name: "Matt"}`
- **THEN** the system returns that record's `id` and `name` values alongside a deleted count of 1

#### Scenario: Delete with a wildcard projection
- **WHEN** a client sends `!> user(id: 1) => {*};` against a schema with fields `id`, `name`, and `age`, and the matching record is `{id: 1, name: "Matt", age: 39}`
- **THEN** the system returns all three fields of that record alongside a deleted count of 1

#### Scenario: Wildcard cannot be combined with named fields
- **WHEN** a client sends `!> user(id: 1) => {*, id};` or `!> user(id: 1) => {id, *};`
- **THEN** the system returns a parse error and does not delete anything

### Requirement: Deleted ids are never reused
The system SHALL NOT reuse a deleted record's internal row id or an auto-incremented `@id` value for any subsequently inserted record in the same collection.

#### Scenario: Insert after delete does not reuse an id
- **WHEN** a record with an auto-incremented `id` of 3 is deleted, and a new record is then inserted into the same collection
- **THEN** the new record's auto-incremented `id` is greater than 3, never 3

### Requirement: Delete validates the collection and field names
The system SHALL return an error, without deleting anything, if the named collection does not exist, or if any field referenced in the filter or projection is not declared in that collection's schema.

#### Scenario: Unknown collection
- **WHEN** a client sends `!> ghost(id: 1);` and no `ghost` collection has been defined
- **THEN** the system returns an error and deletes nothing

#### Scenario: Unknown field in filter
- **WHEN** a client sends `!> user(nope: 1);` and `user` has no `nope` field
- **THEN** the system returns an error and deletes nothing
