## ADDED Requirements

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
