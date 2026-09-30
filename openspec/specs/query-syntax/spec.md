## Purpose

Defines the lexical rules shared by every statement in the query language — starting with how entries are separated inside schema blocks, record literals, filters, and projections — so the same rule holds no matter which statement you're writing.

## Requirements

### Requirement: Commas between list entries are optional
The system SHALL accept entries in schema blocks, record literals, filters, and projections separated by commas, by whitespace alone, or by any mix of the two. A comma SHALL carry no meaning beyond separating entries.

#### Scenario: Record literal without commas
- **WHEN** a client sends `>> user {id: 1 name: "Matt"}`
- **THEN** the system inserts a record with `id` 1 and `name` "Matt", exactly as it would for `>> user {id: 1, name: "Matt"}`

#### Scenario: Filter without commas
- **WHEN** a client sends `<< user(id: 1 name: "Matt") => {id}`
- **THEN** the system applies both filter conditions, exactly as it would for `<< user(id: 1, name: "Matt") => {id}`

#### Scenario: Projection without commas
- **WHEN** a client sends `<< user() => {id name}`
- **THEN** the system returns the `id` and `name` fields, exactly as it would for `<< user() => {id, name}`

#### Scenario: Schema block with commas
- **WHEN** a client sends `user { id: int @id, name: text }`
- **THEN** the system defines the collection exactly as it would for `user { id: int @id name: text }`

#### Scenario: Mixed and trailing commas
- **WHEN** a client sends `>> user {id: 1, name: "Matt" age: 39,}`
- **THEN** the system inserts a record with all three fields

### Requirement: Commas inside strings are not separators
The system SHALL treat a comma inside a quoted string as part of the string's value.

#### Scenario: String value containing a comma
- **WHEN** a client sends `>> user {name: "Smith, Matt"}`
- **THEN** the stored `name` is exactly `Smith, Matt`

### Requirement: Boolean literals
The system SHALL accept the bare words `true` and `false` as boolean values anywhere a value is expected: record literals, filters, and merge payloads. They SHALL be distinct from the quoted strings `"true"` and `"false"`, which remain text.

#### Scenario: Boolean in a record literal
- **WHEN** collection `user { id: int @id active: bool }` exists and a client sends `>> user {id: 1 active: false}`
- **THEN** the system stores `active` as the boolean `false`

#### Scenario: Boolean in a filter
- **WHEN** a client sends `<< user(active: true) => {id}`
- **THEN** the system filters on the boolean `true`

#### Scenario: Quoted true is text, not a boolean
- **WHEN** collection `user { id: int @id active: bool }` exists and a client sends `>> user {id: 1 active: "true"}`
- **THEN** the system returns a type error, because `"true"` is text

#### Scenario: Other bare words are not values
- **WHEN** a client sends `>> user {id: 1 active: yes}`
- **THEN** the system returns a parse error saying a value was expected

### Requirement: Null literal
The system SHALL accept the bare word `null` as a value anywhere a value is expected: record literals, filters, and merge payloads. It means "no value". The quoted string `"null"` SHALL remain text.

#### Scenario: Null in a record literal
- **WHEN** collection `user { id: int @id nickname: text @optional }` exists and a client sends `>> user {id: 1 nickname: null}`
- **THEN** the system parses `nickname` as having no value

#### Scenario: Null in a filter
- **WHEN** a client sends `<< user(nickname: null);`
- **THEN** the system filters for records with no value in `nickname`

#### Scenario: Quoted null is text
- **WHEN** collection `user { id: int @id nickname: text @optional }` exists and a client sends `>> user {id: 1 nickname: "null"}`
- **THEN** the system stores the text `null` as the value of `nickname`

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
