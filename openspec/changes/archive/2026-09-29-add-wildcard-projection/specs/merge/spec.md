## MODIFIED Requirements

### Requirement: Merge result can opt into returning updated records
The system SHALL, when a merge statement includes an `=> {...}` projection clause, return the updated records themselves — reflecting their state after the merge — limited to the projected fields, in addition to the count. The projection MAY be a wildcard `*` in place of a field list, meaning every field currently declared in the collection's schema; `*` MUST be the sole content of the projection — combining it with named fields SHALL be rejected as a parse error.

#### Scenario: Merge with a projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"} => {id, name};` and the matching record was `{id: 1, name: "Sam"}` before the merge
- **THEN** the system returns `{id: 1, name: "Matt"}` (the post-merge state) alongside an updated count of 1

#### Scenario: Merge with a wildcard projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"} => {*};` against a schema with fields `id`, `name`, and `age`, and the matching record was `{id: 1, name: "Sam", age: 40}` before the merge
- **THEN** the system returns all three fields reflecting the post-merge state (`{id: 1, name: "Matt", age: 40}`) alongside an updated count of 1

#### Scenario: Wildcard cannot be combined with named fields
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"} => {*, id};` or `~> user(id: 1) {name: "Matt"} => {id, *};`
- **THEN** the system returns a parse error and does not update anything

### Requirement: Merge result defaults to a count
The system SHALL, when a merge statement has no `=>` projection clause, return only the number of records updated, not the updated records themselves, displayed as a bare integer with no surrounding words.

#### Scenario: Merge with no projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"};` and one record matches
- **THEN** the system returns an updated count of 1, displayed as the bare text `1` — not `"1 updated"` or any other wording — without including that record's field values
