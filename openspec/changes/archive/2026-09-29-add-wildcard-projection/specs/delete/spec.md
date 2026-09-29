## MODIFIED Requirements

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

### Requirement: Delete result defaults to a count
The system SHALL, when a delete statement has no `=>` projection clause, return only the number of records deleted, not the deleted records themselves, displayed as a bare integer with no surrounding words.

#### Scenario: Delete with no projection
- **WHEN** a client sends `!> user(id: 1);` and one record matches
- **THEN** the system returns a deleted count of 1, displayed as the bare text `1` — not `"1 deleted"` or any other wording — without including that record's field values
