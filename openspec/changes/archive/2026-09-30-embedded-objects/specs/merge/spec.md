## ADDED Requirements

### Requirement: Merge payload cannot set an embedded object yet
The system SHALL reject, with an error and no mutation, a merge whose payload sets an embedded object field.

#### Scenario: Payload sets an object field
- **WHEN** collection `user { id: int @id address: { city: text } }` holds a record and a client sends `~> user(id: 1) {address: {city: "MSP"}};`
- **THEN** the system returns `merge payload cannot set embedded object field "address" yet` and changes no record
