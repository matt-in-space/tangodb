## MODIFIED Requirements

### Requirement: Values must match the declared field type exactly
The system SHALL reject any insert record, merge payload, or filter (read, delete, merge) containing a value whose type does not exactly match its field's declared type. An integer literal SHALL match only `int`, a decimal literal only `float`, a quoted string only `text`, and `true`/`false` only `bool`. The system SHALL NOT convert between types, including from integer to float. The one exception is `null`, which is valid for an optional field and SHALL NOT be treated as a type mismatch there. The error SHALL name the field, the declared type, and the value's type.

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


#### Scenario: Null is not a type mismatch on an optional field
- **WHEN** collection `user { id: int @id nickname: text @optional }` exists and a client sends `>> user {id: 1 nickname: null}`
- **THEN** the system stores the record with no value for `nickname`

## ADDED Requirements

### Requirement: Required fields must have a value
Every declared field not marked `@optional` is required. The system SHALL reject an insert whose record omits a required field, or sets it to `null`, naming the field. A primary key that is `@auto` is exempt from supplying a value, because the database assigns it. Merge SHALL NOT require fields it doesn't name, since it is a partial update, but SHALL reject `null` for a required field.

#### Scenario: Insert omits a required field
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1}`
- **THEN** the system returns `field "name" is required for collection "user"` and stores nothing

#### Scenario: Insert sets a required field to null
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: null}`
- **THEN** the system returns `field "name" is required and cannot be null` and stores nothing

#### Scenario: Insert omits an optional field
- **WHEN** collection `user { id: int @id name: text nickname: text @optional }` exists and a client sends `>> user {id: 1 name: "Matt"}`
- **THEN** the system stores the record with no value for `nickname`

#### Scenario: Auto-increment primary key is not required on insert
- **WHEN** collection `user { id: int @id @auto name: text }` exists and a client sends `>> user {name: "Matt"}`
- **THEN** the system stores the record with an assigned `id`

#### Scenario: Merge does not require unnamed fields
- **WHEN** collection `user { id: int @id name: text age: int }` holds record `{id: 1 name: "Matt" age: 39}` and a client sends `~> user(id: 1) {age: 40};`
- **THEN** the system updates `age` and `name` is unchanged

#### Scenario: Merge rejects null for a required field
- **WHEN** collection `user { id: int @id name: text }` holds a record and a client sends `~> user() {name: null};`
- **THEN** the system returns `field "name" is required and cannot be null` and changes no record

### Requirement: Null clears an optional field
Writing `null` to an optional field SHALL leave it with no value. On merge, this SHALL clear any value the matched records had. Omitting an optional field and writing `null` for it SHALL be indistinguishable afterward.

#### Scenario: Merge clears a value
- **WHEN** collection `user { id: int @id nickname: text @optional }` holds `{id: 1 nickname: "M"}` and a client sends `~> user(id: 1) {nickname: null};`
- **THEN** `<< user(nickname: null);` returns `1`

#### Scenario: Omitted and null are the same
- **WHEN** a client inserts `{id: 1}` and `{id: 2 nickname: null}` into collection `user { id: int @id nickname: text @optional }`
- **THEN** `<< user(nickname: null);` returns `2`

### Requirement: Filtering on null
A filter condition `field: null` SHALL match exactly the records where that optional field has no value, using ordinary equality: `null` equals `null`. A `null` filter on a required field SHALL be an error, since it could never match.

#### Scenario: Null filter matches records without a value
- **WHEN** collection `user { id: int @id nickname: text @optional }` holds `{id: 1}` and `{id: 2 nickname: "M"}` and a client sends `<< user(nickname: null) => {id}`
- **THEN** the system returns only the record with `id` 1

#### Scenario: Null filter on a required field is rejected
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `<< user(name: null);`
- **THEN** the system returns `field "name" is required and cannot be null`

### Requirement: Missing values display as null
When a projected field has no value on a record, the system SHALL display it as `null` rather than as an empty cell.

#### Scenario: Projection shows null
- **WHEN** collection `user { id: int @id nickname: text @optional }` holds `{id: 1}` and a client sends `<< user => {id, nickname};`
- **THEN** the row for `id` 1 shows `null` in the `nickname` column
