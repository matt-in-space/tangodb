## Purpose

Makes a collection's schema binding: every value written to a collection or used to filter it must match its field's declared type exactly, fields the schema doesn't declare are rejected, and a statement that fails validation changes nothing.

## ADDED Requirements

### Requirement: Values must match the declared field type exactly
The system SHALL reject any insert record, merge payload, or filter (read, delete, merge) containing a value whose type does not exactly match its field's declared type. An integer literal SHALL match only `int`, a decimal literal only `float`, a quoted string only `text`, and `true`/`false` only `bool`. The system SHALL NOT convert between types, including from integer to float. The error SHALL name the field, the declared type, and the value's type.

#### Scenario: Wrong type on insert
- **WHEN** collection `user { id: int @id age: int }` exists and a client sends `>> user {id: 1 age: "old"}`
- **THEN** the system returns an error naming field `age`, expected type `int`, and actual type `text`, and stores nothing

#### Scenario: Integer into a float field is rejected
- **WHEN** collection `item { id: int @id price: float }` exists and a client sends `>> item {id: 1 price: 10}`
- **THEN** the system returns an error naming field `price`, expected type `float`, and actual type `int`

#### Scenario: Decimal into a float field is accepted
- **WHEN** collection `item { id: int @id price: float }` exists and a client sends `>> item {id: 1 price: 10.0}`
- **THEN** the system stores the record

#### Scenario: Wrong type in a filter
- **WHEN** collection `item { id: int @id price: float }` exists and a client sends `<< item(price: 10) => {id}`
- **THEN** the system returns a type error for field `price` rather than matching nothing

#### Scenario: Wrong type in a merge payload
- **WHEN** collection `user { id: int @id active: bool }` exists and a client sends `~> user(id: 1) {active: "yes"}`
- **THEN** the system returns a type error for field `active` and changes no record

#### Scenario: Boolean field round-trip
- **WHEN** collection `user { id: int @id active: bool }` exists and a client sends `>> user {id: 1 active: true}` followed by `<< user(active: true) => {id}`
- **THEN** the read returns the record with `id` 1

### Requirement: Undeclared fields are rejected
The system SHALL reject any insert record or merge payload containing a field that the collection's schema does not declare, with the same error already used for undeclared filter and projection fields: `field "<name>" not found in schema for collection "<collection>"`.

#### Scenario: Undeclared field on insert
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 nmae: "Matt"}`
- **THEN** the system returns `field "nmae" not found in schema for collection "user"` and stores nothing

#### Scenario: Undeclared field in a merge payload
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `~> user() {nickname: "M"}`
- **THEN** the system returns `field "nickname" not found in schema for collection "user"` and changes no record

### Requirement: An invalid statement changes nothing
The system SHALL validate an entire statement before making any change. If any field or value in the statement is invalid, the statement SHALL fail as a whole and no record SHALL be inserted, updated, or deleted.

#### Scenario: One bad field fails the whole insert
- **WHEN** collection `user { id: int @id name: text age: int }` exists and a client sends `>> user {id: 1 name: "Matt" age: "x"}`
- **THEN** the system returns an error and a subsequent `<< user;` returns `0`

#### Scenario: Bulk merge never half-applies
- **WHEN** collection `user` holds two records and a client sends `~> user() {name: "Sam" age: "x"}` where `age` is declared `int`
- **THEN** the system returns an error and neither record's `name` has changed

#### Scenario: Invalid delete filter deletes nothing
- **WHEN** collection `user { id: int @id }` holds one record and a client sends `!> user(id: "1")`
- **THEN** the system returns a type error and the record is still present
