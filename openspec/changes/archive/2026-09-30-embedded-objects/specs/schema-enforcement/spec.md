## ADDED Requirements

### Requirement: Embedded values are validated recursively
The system SHALL validate a value for an embedded block field against that block's fields, using the same rules as top-level fields: undeclared fields are rejected, types must match exactly, `null` is only allowed on optional fields, and required fields must have a value. Required fields inside an optional block SHALL only be checked when the block has a value. Every problem SHALL name the field's full dotted path.

#### Scenario: Wrong type inside a block
- **WHEN** collection `user { id: int @id address: { city: text } }` exists and a client sends `>> user {id: 1 address: {city: 5}};`
- **THEN** the system returns `field "address.city": expected text, got int` and stores nothing

#### Scenario: Missing required field inside a block
- **WHEN** collection `user { id: int @id address: { street: text city: text } }` exists and a client sends `>> user {id: 1 address: {city: "MSP"}};`
- **THEN** the system returns `field "address.street" is required for collection "user"`

#### Scenario: Undeclared field inside a block
- **WHEN** collection `user { id: int @id address: { city: text } }` exists and a client sends `>> user {id: 1 address: {city: "MSP" zip: "55401"}};`
- **THEN** the system returns `field "address.zip" not found in schema for collection "user"`

#### Scenario: Optional block omitted
- **WHEN** collection `user { id: int @id address: { city: text } @optional }` exists and a client sends `>> user {id: 1};`
- **THEN** the system stores the record with no value for `address`

#### Scenario: Required block omitted
- **WHEN** collection `user { id: int @id address: { city: text } }` exists and a client sends `>> user {id: 1};`
- **THEN** the system returns `field "address" is required for collection "user"`

### Requirement: Values must have the declared shape
The system SHALL reject a scalar value on an embedded block field, and an object value on a scalar field.

#### Scenario: Scalar on a block field
- **WHEN** collection `user { id: int @id address: { city: text } }` exists and a client sends `>> user {id: 1 address: 5};`
- **THEN** the system returns `field "address": expected object, got int`

#### Scenario: Object on a scalar field
- **WHEN** collection `user { id: int @id name: text }` exists and a client sends `>> user {id: 1 name: {first: "Matt"}};`
- **THEN** the system returns `field "name": expected text, got object`
