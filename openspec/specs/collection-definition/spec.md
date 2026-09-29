## Purpose

Defines how a collection's fields are declared, including the annotations that change how a field behaves (`@id`, `@auto`, `@optional`) and which combinations are allowed.

## Requirements

### Requirement: Fields are required unless marked optional
Every field in a collection declaration SHALL be required by default. The `@optional` annotation SHALL mark a field that may have no value. Declaring `@optional` twice on the same field SHALL be an error. A collection's schema output SHALL show `@optional` on the fields that have it.

#### Scenario: Declaring an optional field
- **WHEN** a client sends `user { id: int @id nickname: text @optional }`
- **THEN** the system defines the collection, and its schema output shows `nickname: text @optional`

#### Scenario: Duplicate optional annotation
- **WHEN** a client sends `user { nickname: text @optional @optional }`
- **THEN** the system returns an error and defines no collection

### Requirement: A primary key cannot be optional
The system SHALL reject a declaration that marks the same field both `@id` and `@optional`, with or without `@auto`.

#### Scenario: Id and optional together
- **WHEN** a client sends `user { id: int @id @optional }`
- **THEN** the system returns `@id field "id" cannot be @optional` and defines no collection

#### Scenario: Auto id and optional together
- **WHEN** a client sends `user { id: int @id @auto @optional }`
- **THEN** the system returns `@id field "id" cannot be @optional` and defines no collection
