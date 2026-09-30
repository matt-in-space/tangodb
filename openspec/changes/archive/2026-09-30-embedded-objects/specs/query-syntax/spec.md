## ADDED Requirements

### Requirement: Nested record literals
The system SHALL accept a record literal, `{ ... }`, as a value, nested to any depth, anywhere a record's field value is expected.

#### Scenario: Insert with a nested record
- **WHEN** collection `user { id: int @id address: { street: text city: text } }` exists and a client sends `>> user {id: 1 address: {street: "1 Main" city: "MSP"}};`
- **THEN** the system stores the record with its nested `address`

### Requirement: Projections flatten embedded objects into dotted columns
In any projection (read, delete, merge, or insert), the system SHALL show an embedded object's fields as separate columns named by their dotted path. `*` SHALL expand to every leaf path in the schema, sorted. Naming an object field SHALL expand to its leaf paths. A dotted path to a leaf SHALL be its own column. A path not in the schema SHALL be an error. A leaf with no value, including every leaf of an absent optional object, SHALL display as `null`.

#### Scenario: Wildcard flattens objects
- **WHEN** collection `user { id: int @id address: { street: text city: text } }` holds `{id: 1 address: {street: "1 Main" city: "MSP"}}` and a client sends `<< user => {*};`
- **THEN** the system prints columns `address.city`, `address.street`, `id` with the row `MSP`, `1 Main`, `1`

#### Scenario: Naming an object field
- **WHEN** a client sends `<< user => {id address};` against the same record
- **THEN** the system prints columns `id`, `address.city`, `address.street`

#### Scenario: Dotted projection
- **WHEN** a client sends `<< user => {address.city};` against the same record
- **THEN** the system prints the single column `address.city` with the value `MSP`

#### Scenario: Absent optional object shows null
- **WHEN** collection `user { id: int @id address: { city: text } @optional }` holds `{id: 1}` and a client sends `<< user => {*};`
- **THEN** the row shows `null` in the `address.city` column

#### Scenario: Unknown path
- **WHEN** a client sends `<< user => {address.zip};` and `address` has no `zip` field
- **THEN** the system returns `field "address.zip" not found in schema for collection "user"`

### Requirement: Filters on embedded objects are not supported yet
The system SHALL reject a filter condition (in read, delete, or merge) that names an embedded object field or uses a dotted path, with an error saying it is not supported yet, and SHALL change nothing.

#### Scenario: Filtering on an object field
- **WHEN** collection `user { id: int @id address: { city: text } }` exists and a client sends `<< user(address: {city: "MSP"});`
- **THEN** the system returns `filtering on embedded object field "address" is not supported yet`

#### Scenario: Filtering on a dotted path
- **WHEN** a client sends `!> user(address.city: "MSP");` against the same collection
- **THEN** the system returns `filtering on embedded object field "address" is not supported yet` and deletes nothing
