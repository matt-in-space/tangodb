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
