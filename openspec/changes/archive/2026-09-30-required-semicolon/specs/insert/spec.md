## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: A bare insert needs a terminator to be complete
**Reason**: Replaced by the general rule that every statement ends with `;` (query-syntax: "Every statement ends with a semicolon"), which covers inserts along with every other statement.
**Migration**: End every insert with `;`, whether or not it has a `=>` projection.
