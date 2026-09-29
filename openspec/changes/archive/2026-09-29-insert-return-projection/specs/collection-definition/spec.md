## ADDED Requirements

### Requirement: Defining a collection returns its schema
The system SHALL, when a collection is defined, return the collection's schema written in the schema syntax, with no surrounding characters beyond the schema itself.

#### Scenario: Schema output has no wrapper
- **WHEN** a client sends `user { id: int @id name: text }`
- **THEN** the system prints exactly:
  ```
  user {
    id: int @id
    name: text
  }
  ```
