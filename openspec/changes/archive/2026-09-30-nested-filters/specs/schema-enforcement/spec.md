## ADDED Requirements

### Requirement: Filter values are validated at their path
The system SHALL validate each filter condition against the schema at its path, naming the full path in any error:
- a path not in the schema is an error
- a scalar value must match the field's type exactly
- an object value or `{*}` is only allowed on an embedded object field, and a scalar value is not
- fields inside a subset filter are validated recursively, but fields it leaves out are not required
- `null` on a dotted path is allowed only if the field, or some block above it, is optional
- `null` inside a subset filter is allowed only if that field is optional

#### Scenario: Unknown path
- **WHEN** collection `user { id: int @id address: { city: text } }` exists and a client sends `<< user(address.zip: "55401");`
- **THEN** the system returns `field "address.zip" not found in schema for collection "user"`

#### Scenario: Wrong type at a path
- **WHEN** a client sends `<< user(address.city: 5);` against the same collection
- **THEN** the system returns `field "address.city": expected text, got int`

#### Scenario: Wrong type inside a subset
- **WHEN** a client sends `<< user(address: {city: 5});` against the same collection
- **THEN** the system returns `field "address.city": expected text, got int`

#### Scenario: Scalar on an object field
- **WHEN** a client sends `<< user(address: "MSP");` against the same collection
- **THEN** the system returns `field "address": expected object, got text`

#### Scenario: Null where it can never match
- **WHEN** collection `user { id: int @id address: { city: text } }` exists (both required) and a client sends `<< user(address.city: null);`
- **THEN** the system returns `field "address.city" is required and cannot be null`

#### Scenario: Null allowed through an optional block
- **WHEN** collection `user { id: int @id address: { city: text } @optional }` exists and a client sends `<< user(address.city: null);`
- **THEN** the system accepts the filter
