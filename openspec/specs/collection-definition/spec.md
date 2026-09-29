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

### Requirement: Auto-increment works on any int field
The `@auto` annotation SHALL be allowed on any `int` field, whether or not it is also `@id`. Using it on a non-`int` field SHALL be an error. A collection MAY declare more than one `@auto` field.

#### Scenario: Auto field alongside a text primary key
- **WHEN** a client sends `ticket { code: text @id number: int @auto title: text }`
- **THEN** the system defines the collection

#### Scenario: Auto on a non-int field
- **WHEN** a client sends `ticket { number: text @auto }`
- **THEN** the system returns `@auto requires an int field (field "number")` and defines no collection

#### Scenario: Multiple auto fields
- **WHEN** a client sends `event { id: int @id @auto seq: int @auto }`
- **THEN** the system defines the collection

### Requirement: Auto-increment values are assigned by the database
Each `@auto` field SHALL have its own counter, starting at `1`, incremented by one for each record inserted. Values SHALL never be reused. Supplying an `@auto` field on insert, including as `null`, SHALL be an error. An insert that fails for any reason SHALL NOT consume a counter value.

#### Scenario: Sequential values on a non-id field
- **WHEN** collection `ticket { code: text @id number: int @auto }` exists and a client inserts `{code: "A"}` then `{code: "B"}`
- **THEN** the records receive `number` 1 and 2

#### Scenario: Independent counters
- **WHEN** collection `event { id: int @id @auto seq: int @auto }` exists and a client inserts two records
- **THEN** each field counts 1, 2 on its own

#### Scenario: Supplying an auto field
- **WHEN** collection `ticket { code: text @id number: int @auto }` exists and a client sends `>> ticket {code: "A" number: 7}`
- **THEN** the system returns `field "number" is auto-increment and must not be supplied for collection "ticket"`

#### Scenario: Failed insert does not consume a value
- **WHEN** collection `ticket { code: text @id number: int @auto }` holds `{code: "A" number: 1}` and a client inserts `{code: "A"}` (a duplicate key), then `{code: "B"}`
- **THEN** the first insert fails and the second receives `number` 2

### Requirement: An auto-increment field cannot be optional
The system SHALL reject a declaration that marks a field both `@auto` and `@optional`, since the field always has a value.

#### Scenario: Auto and optional together
- **WHEN** a client sends `ticket { code: text @id number: int @auto @optional }`
- **THEN** the system returns `@auto field "number" cannot be @optional` and defines no collection
