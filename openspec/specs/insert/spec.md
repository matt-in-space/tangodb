## Purpose

Defines the insert statement's observable behavior: when it's complete and what it returns. Insert follows the same result convention as read, delete, and merge: a bare count by default, and records only when a `=>` projection asks for them.

## Requirements

### Requirement: Insert result defaults to a count
The system SHALL, when an insert statement has no `=>` projection clause, return only the number of records inserted, displayed as a bare integer with no surrounding words and no field values.

#### Scenario: Insert with no projection
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"};`
- **THEN** the system stores the record and returns the bare text `1`

### Requirement: Insert result can opt into returning the inserted record
The system SHALL, when an insert statement includes an `=> {...}` projection clause, return every inserted record as stored, in input order, limited to the projected fields, including any values the database assigned. The projection MAY be the wildcard `*`, meaning every field in the collection's schema. A projection naming a field not in the schema SHALL be an error, and no record SHALL be stored.

#### Scenario: Returning a generated id
- **WHEN** collection `user { id: int @id @auto name: text }` exists and a client sends `>> user {name: "Matt"} => {id};`
- **THEN** the system returns a table with the single column `id` and the value `1`

#### Scenario: Wildcard projection
- **WHEN** collection `user { id: int @id name: text nickname: text @optional }` exists and a client sends `>> user {id: 1 name: "Matt"} => {*};`
- **THEN** the system returns a table with columns `id`, `name`, `nickname` and the row `1`, `Matt`, `null`

#### Scenario: Unknown projection field stores nothing
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"} => {nope};`
- **THEN** the system returns `field "nope" not found in schema for collection "user"` and `<< user;` returns `0`

#### Scenario: Projection on a batch
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"} & {id: 2 name: "Sam"} => {id};`
- **THEN** the system returns a table with the column `id` and the rows `1` and `2`, in that order

### Requirement: Insert accepts a batch of records
An insert statement SHALL accept one or more record literals separated by `&`, all inserted into the one named collection. A trailing `&` SHALL mean the statement is incomplete. A `=>` projection, if present, SHALL come after the last record and apply to all of them. The result count SHALL be the number of records inserted.

#### Scenario: Batch insert returns the count
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"} & {id: 2 name: "Sam"};`
- **THEN** the system stores both records and returns the bare text `2`

#### Scenario: Batch spread over lines
- **WHEN** a client enters, line by line, `>> user {id: 1 name: "Matt"} &` and then `{id: 2 name: "Sam"};`
- **THEN** the REPL waits after the first line and stores both records after the second

#### Scenario: Dangling separator
- **WHEN** a client sends `>> user {id: 1 name: "Matt"} & ;`
- **THEN** the system returns a parse error and stores nothing

#### Scenario: Records in one batch may differ in optional fields
- **WHEN** collection `user { id: int @id name: text nickname: text @optional }` exists and a client sends `>> user {id: 1 name: "Matt" nickname: "M"} & {id: 2 name: "Sam"};`
- **THEN** the system stores both records

### Requirement: A batch is all-or-nothing
The system SHALL validate every record in a batch before storing any of them. If any record is invalid, or two records in the batch share a primary key value, the system SHALL store none of them. Auto-increment values SHALL be assigned in input order, and only once every record is valid.

#### Scenario: One bad record stores nothing
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"} & {id: 2};`
- **THEN** the system returns `record 2: field "name" is required for collection "user"` and `<< user;` returns `0`

#### Scenario: Duplicate key within the batch
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: "Matt"} & {id: 1 name: "Sam"};`
- **THEN** the system returns `record 2: duplicate primary key 1 for collection "user"` and stores nothing

#### Scenario: Auto values follow input order
- **WHEN** collection `user { id: int @id @auto name: text }` exists and a client sends `>> user {name: "Matt"} & {name: "Sam"} => {id name};`
- **THEN** the rows come back as `1 Matt` then `2 Sam`

#### Scenario: Failed batch consumes no auto values
- **WHEN** collection `user { id: int @id @auto name: text }` exists, a client sends `>> user {name: "Matt"} & {name: 5};`, and then `>> user {name: "Sam"} => {id};`
- **THEN** the batch fails and the second insert returns `id` `1`

### Requirement: Insert reports every problem
The system SHALL report every problem it finds in an insert statement, not only the first, with at most one problem per field. In a batch of more than one record, each problem SHALL be prefixed with `record N: `, counting from 1. A single insert's problems SHALL have no prefix. When there is exactly one problem, the error SHALL be that problem's message alone. When there are several, the error SHALL begin with `N problems, nothing inserted:` and list each problem on its own line.

#### Scenario: Several problems across a batch
- **WHEN** collection `user { id: int @id name: text age: int @optional }` exists and a client sends `>> user {id: 1 name: "Matt"} & {id: 2} & {id: 1 name: "Sam" age: "x"};`
- **THEN** the system returns an error beginning `3 problems, nothing inserted:` and listing `record 2: field "name" is required for collection "user"`, `record 3: field "age": expected int, got text`, and `record 3: duplicate primary key 1 for collection "user"`

#### Scenario: Several problems in a single insert
- **WHEN** collection `user { id: int @id name: text age: int }` exists and a client sends `>> user {id: 1 name: 5 age: "x"};`
- **THEN** the system returns an error beginning `2 problems, nothing inserted:` and listing `field "age": expected int, got text` and `field "name": expected text, got int`, with no `record` prefix

#### Scenario: One problem per field
- **WHEN** collection `user { id: int @id @auto }` exists and a client sends `>> user {id: null};`
- **THEN** the system returns only `field "id" is auto-increment and must not be supplied for collection "user"`
