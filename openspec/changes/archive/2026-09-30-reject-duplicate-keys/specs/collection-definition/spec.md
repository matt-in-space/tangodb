## ADDED Requirements

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
