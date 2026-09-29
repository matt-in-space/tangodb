## ADDED Requirements

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
