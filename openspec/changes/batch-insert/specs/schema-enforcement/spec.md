## MODIFIED Requirements

### Requirement: An invalid statement changes nothing
The system SHALL validate an entire statement before making any change. If any field or value in the statement is invalid, the statement SHALL fail as a whole and no record SHALL be inserted, updated, or deleted.

#### Scenario: One bad field fails the whole insert
- **WHEN** collection `user { id: int @id name: text age: int }` exists and a client sends `>> user {id: 1 name: "Matt" age: "x"};`
- **THEN** the system returns an error and a subsequent `<< user;` returns `0`

#### Scenario: Bulk merge never half-applies
- **WHEN** collection `user` holds two records and a client sends `~> user() {name: "Sam" age: "x"}` where `age` is declared `int`
- **THEN** the system returns an error and neither record's `name` has changed

#### Scenario: Invalid delete filter deletes nothing
- **WHEN** collection `user { id: int @id }` holds one record and a client sends `!> user(id: "1")`
- **THEN** the system returns a type error and the record is still present

#### Scenario: One bad record fails the whole batch
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"} & {id: 2 name: 7};`
- **THEN** the system returns an error and a subsequent `<< user;` returns `0`
