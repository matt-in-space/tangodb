## ADDED Requirements

### Requirement: Merge payload must not set an auto-increment field
The system SHALL reject, with an error and no mutation, any merge whose payload includes an `@auto` field. If the field is also the primary key, the existing primary-key error SHALL be reported instead. Filtering on an `@auto` field is unaffected.

#### Scenario: Payload sets a non-id auto field
- **WHEN** collection `ticket { code: text @id number: int @auto }` holds a record and a client sends `~> ticket() {number: 5};`
- **THEN** the system returns `payload must not set auto-increment field "number" for collection "ticket"` and changes no record

#### Scenario: Filtering on an auto field
- **WHEN** collection `ticket { code: text @id number: int @auto title: text @optional }` holds `{code: "A" number: 1}` and a client sends `~> ticket(number: 1) {title: "hi"};`
- **THEN** the system updates that record's `title`
