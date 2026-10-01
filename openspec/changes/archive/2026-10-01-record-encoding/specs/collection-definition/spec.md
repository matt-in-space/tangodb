## ADDED Requirements

### Requirement: Field names and nesting have size limits
The system SHALL reject a collection definition containing a field name longer than 255 bytes, or embedded blocks nested more than 32 levels deep, so that every valid record can be stored.

#### Scenario: Field name too long
- **WHEN** a client defines a collection with a field whose name is 256 bytes long
- **THEN** the system returns `field name "<name>" is longer than 255 bytes` and defines no collection

#### Scenario: Nesting too deep
- **WHEN** a client defines a collection whose embedded blocks are nested 33 levels deep
- **THEN** the system returns an error naming the field that's nested more than 32 levels deep, and defines no collection
