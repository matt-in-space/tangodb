## Purpose

Defines how a collection's fields are declared, including the annotations that change how a field behaves (`@id`, `@auto`, `@optional`) and which combinations are allowed.

## Requirements

### Requirement: Fields are required unless marked optional
Every field in a collection declaration SHALL be required by default. The `@optional` annotation SHALL mark a field that may have no value. Declaring `@optional` twice on the same field SHALL be an error. A collection's schema output SHALL show `@optional` on the fields that have it.

#### Scenario: Declaring an optional field
- **WHEN** a client sends `user { id: int @id nickname: text @optional };`
- **THEN** the system defines the collection, and its schema output shows `nickname: text @optional`

#### Scenario: Duplicate optional annotation
- **WHEN** a client sends `user { nickname: text @optional @optional };`
- **THEN** the system returns an error and defines no collection

### Requirement: A primary key cannot be optional
The system SHALL reject a declaration that marks the same field both `@id` and `@optional`, with or without `@auto`.

#### Scenario: Id and optional together
- **WHEN** a client sends `user { id: int @id @optional };`
- **THEN** the system returns `@id field "id" cannot be @optional` and defines no collection

#### Scenario: Auto id and optional together
- **WHEN** a client sends `user { id: int @id @auto @optional };`
- **THEN** the system returns `@id field "id" cannot be @optional` and defines no collection

### Requirement: Auto-increment works on any int field
The `@auto` annotation SHALL be allowed on any `int` field, whether or not it is also `@id`. Using it on a non-`int` field SHALL be an error. A collection MAY declare more than one `@auto` field.

#### Scenario: Auto field alongside a text primary key
- **WHEN** a client sends `ticket { code: text @id number: int @auto title: text };`
- **THEN** the system defines the collection

#### Scenario: Auto on a non-int field
- **WHEN** a client sends `ticket { number: text @auto };`
- **THEN** the system returns `@auto requires an int field (field "number")` and defines no collection

#### Scenario: Multiple auto fields
- **WHEN** a client sends `event { id: int @id @auto seq: int @auto };`
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
- **WHEN** collection `ticket { code: text @id number: int @auto }` exists and a client sends `>> ticket {code: "A" number: 7};`
- **THEN** the system returns `field "number" is auto-increment and must not be supplied for collection "ticket"`

#### Scenario: Failed insert does not consume a value
- **WHEN** collection `ticket { code: text @id number: int @auto }` holds `{code: "A" number: 1}` and a client inserts `{code: "A"}` (a duplicate key), then `{code: "B"}`
- **THEN** the first insert fails and the second receives `number` 2

### Requirement: An auto-increment field cannot be optional
The system SHALL reject a declaration that marks a field both `@auto` and `@optional`, since the field always has a value.

#### Scenario: Auto and optional together
- **WHEN** a client sends `ticket { code: text @id number: int @auto @optional };`
- **THEN** the system returns `@auto field "number" cannot be @optional` and defines no collection

### Requirement: Defining a collection returns its schema
The system SHALL, when a collection is defined, return the collection's schema written in the schema syntax, with no surrounding characters beyond the schema itself.

#### Scenario: Schema output has no wrapper
- **WHEN** a client sends `user { id: int @id name: text };`
- **THEN** the system prints exactly:
  ```
  user {
    id: int @id
    name: text
  }
  ```

### Requirement: A field can be an embedded block
A field's type SHALL be allowed to be an embedded block, `{ ... }`, containing field declarations of its own, which MAY include further blocks. An embedded block has no identity of its own and is stored as part of its parent record. A block MAY be marked `@optional` after its closing brace, meaning the whole object may have no value. Fields inside a block follow the usual required/`@optional` rule.

#### Scenario: Declaring an embedded block
- **WHEN** a client sends `user { id: int @id address: { street: text city: text } };`
- **THEN** the system defines the collection with an `address` field holding `street` and `city`

#### Scenario: Nested blocks
- **WHEN** a client sends `user { id: int @id address: { city: text geo: { lat: float lng: float } } };`
- **THEN** the system defines the collection

#### Scenario: Optional block
- **WHEN** a client sends `user { id: int @id address: { city: text } @optional };`
- **THEN** the system defines the collection with `address` optional

### Requirement: Identity annotations are not allowed inside an embedded block
The system SHALL reject `@id` or `@auto` on any field inside an embedded block, naming the field's full dotted path, since an embedded block has no identity of its own.

#### Scenario: Id inside a block
- **WHEN** a client sends `user { id: int @id address: { id: int @id } };`
- **THEN** the system returns `@id is not allowed inside an embedded block (field "address.id")` and defines no collection

#### Scenario: Auto inside a block
- **WHEN** a client sends `user { id: int @id address: { n: int @auto } };`
- **THEN** the system returns `@auto is not allowed inside an embedded block (field "address.n")` and defines no collection

### Requirement: Schema output shows embedded blocks
A collection's schema output SHALL show each embedded block in the declaration syntax, with its fields indented one level deeper and any annotation after its closing brace.

#### Scenario: Schema output with a block
- **WHEN** a client sends `user { id: int @id address: { street: text city: text } @optional };`
- **THEN** the system prints:
  ```
  user {
    address: {
      city: text
      street: text
    } @optional
    id: int @id
  }
  ```

### Requirement: A field can be declared only once
The system SHALL reject a collection definition that declares the same field more than once in one block, at any depth, with `field "<path>" is declared more than once` naming the field's full dotted path, and SHALL define no collection.

#### Scenario: Repeated top-level field
- **WHEN** a client sends `user { id: int id: text };`
- **THEN** the system returns `field "id" is declared more than once` and defines no collection

#### Scenario: Repeated field inside a block
- **WHEN** a client sends `user { id: int @id address: { city: text city: text } };`
- **THEN** the system returns `field "address.city" is declared more than once`

#### Scenario: Same name in different blocks is fine
- **WHEN** a client sends `user { name: text address: { name: text } };`
- **THEN** the system defines the collection
