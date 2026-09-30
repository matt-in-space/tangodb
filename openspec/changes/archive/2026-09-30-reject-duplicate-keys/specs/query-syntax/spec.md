## ADDED Requirements

### Requirement: A field can be given only once
The system SHALL reject a record literal, at any depth, or a filter that gives the same field more than once, with `field "<path>" is given more than once` naming the field's full dotted path, and SHALL run nothing. This applies to insert records (every record in a batch), merge payloads, and read, delete, and merge filters.

#### Scenario: Repeated field in a record
- **WHEN** a client sends `>> user {id: 1 name: "A" name: "B"};`
- **THEN** the system returns `field "name" is given more than once` and stores nothing

#### Scenario: Repeated field in a nested record
- **WHEN** a client sends `>> user {id: 1 address: {city: "A" city: "B"}};`
- **THEN** the system returns `field "address.city" is given more than once`

#### Scenario: Repeated field in a batch record
- **WHEN** a client sends `>> user {id: 1 name: "A"} & {id: 2 name: "B" name: "C"};`
- **THEN** the system returns `field "name" is given more than once` and stores nothing

#### Scenario: Repeated field in a filter
- **WHEN** a client sends `!> user(id: 1 id: 2);`
- **THEN** the system returns `field "id" is given more than once` and deletes nothing

#### Scenario: Repeated field in a merge payload
- **WHEN** a client sends `~> user(id: 1) {name: "A" name: "B"};`
- **THEN** the system returns `field "name" is given more than once` and changes nothing
