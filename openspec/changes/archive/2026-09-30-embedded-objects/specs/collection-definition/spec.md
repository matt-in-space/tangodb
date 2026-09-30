## ADDED Requirements

### Requirement: A field can be an embedded block
A field's type SHALL be allowed to be an embedded block, `{ ... }`, containing field declarations of its own, which MAY include further blocks. An embedded block has no identity of its own and is stored as part of its parent record. A block MAY be marked `@optional` after its closing brace, meaning the whole object may have no value. Fields inside a block follow the usual required/`@optional` rule.

#### Scenario: Declaring an embedded block
- **WHEN** a client sends `user { id: int @id address: { street: text city: text } }`
- **THEN** the system defines the collection with an `address` field holding `street` and `city`

#### Scenario: Nested blocks
- **WHEN** a client sends `user { id: int @id address: { city: text geo: { lat: float lng: float } } }`
- **THEN** the system defines the collection

#### Scenario: Optional block
- **WHEN** a client sends `user { id: int @id address: { city: text } @optional }`
- **THEN** the system defines the collection with `address` optional

### Requirement: Identity annotations are not allowed inside an embedded block
The system SHALL reject `@id` or `@auto` on any field inside an embedded block, naming the field's full dotted path, since an embedded block has no identity of its own.

#### Scenario: Id inside a block
- **WHEN** a client sends `user { id: int @id address: { id: int @id } }`
- **THEN** the system returns `@id is not allowed inside an embedded block (field "address.id")` and defines no collection

#### Scenario: Auto inside a block
- **WHEN** a client sends `user { id: int @id address: { n: int @auto } }`
- **THEN** the system returns `@auto is not allowed inside an embedded block (field "address.n")` and defines no collection

### Requirement: Schema output shows embedded blocks
A collection's schema output SHALL show each embedded block in the declaration syntax, with its fields indented one level deeper and any annotation after its closing brace.

#### Scenario: Schema output with a block
- **WHEN** a client sends `user { id: int @id address: { street: text city: text } @optional }`
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
