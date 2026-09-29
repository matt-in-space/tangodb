## Purpose

Defines the insert statement's observable behavior: when it's complete and what it returns. Insert follows the same result convention as read, delete, and merge: a bare count by default, and records only when a `=>` projection asks for them.

## Requirements

### Requirement: Insert result defaults to a count
The system SHALL, when an insert statement has no `=>` projection clause, return only the number of records inserted, displayed as a bare integer with no surrounding words and no field values.

#### Scenario: Insert with no projection
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"};`
- **THEN** the system stores the record and returns the bare text `1`

### Requirement: Insert result can opt into returning the inserted record
The system SHALL, when an insert statement includes an `=> {...}` projection clause, return the inserted record as stored, limited to the projected fields, including any values the database assigned. The projection MAY be the wildcard `*`, meaning every field in the collection's schema. A projection naming a field not in the schema SHALL be an error, and the record SHALL NOT be stored.

#### Scenario: Returning a generated id
- **WHEN** collection `user { id: int @id @auto name: text }` exists and a client sends `>> user {name: "Matt"} => {id}`
- **THEN** the system returns a table with the single column `id` and the value `1`

#### Scenario: Wildcard projection
- **WHEN** collection `user { id: int @id name: text nickname: text @optional }` exists and a client sends `>> user {id: 1 name: "Matt"} => {*}`
- **THEN** the system returns a table with columns `id`, `name`, `nickname` and the row `1`, `Matt`, `null`

#### Scenario: Unknown projection field stores nothing
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"} => {nope}`
- **THEN** the system returns `field "nope" not found in schema for collection "user"` and `<< user;` returns `0`

### Requirement: A bare insert needs a terminator to be complete
Because a `=>` projection may follow the record literal, the system SHALL treat an insert that ends right after its record literal as incomplete, and wait for more input. A trailing `;` SHALL complete it without a projection, as SHALL a completed `=>` clause.

#### Scenario: Insert without a terminator waits
- **WHEN** a client enters `>> user {id: 1 name: "Matt"}` in the REPL
- **THEN** the REPL shows a continuation prompt and stores nothing yet

#### Scenario: Semicolon completes the insert
- **WHEN** a client enters `>> user {id: 1 name: "Matt"};` in the REPL
- **THEN** the REPL runs the insert and prints `1`
