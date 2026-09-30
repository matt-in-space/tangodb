## REMOVED Requirements

### Requirement: Filters on embedded objects are not supported yet
**Reason**: Filtering on embedded objects is now supported, through dotted-path and subset object filters.
**Migration**: Use `(address.city: "MSP")` to filter on a value inside an object, or `(address: {city: "MSP"})` to require the object and filter inside it.

## ADDED Requirements

### Requirement: Filtering on a dotted path
A filter condition SHALL accept a dotted path to a field inside an embedded object (`address.city`) and SHALL match when the value at that path equals the filter value. A path that runs through an object with no value SHALL itself have no value, so a `null` condition on it SHALL match records that are missing the object.

#### Scenario: Dotted path equality
- **WHEN** collection `user { id: int @id address: { city: text } @optional }` holds `{id: 1 address: {city: "MSP"}}` and `{id: 2 address: {city: "STP"}}` and a client sends `<< user(address.city: "MSP") => {id};`
- **THEN** the system returns only the record with `id` 1

#### Scenario: Null on a dotted path matches a missing object
- **WHEN** collection `user { id: int @id address: { city: text zip: text @optional } @optional }` holds `{id: 1}`, `{id: 2 address: {city: "MSP"}}`, and `{id: 3 address: {city: "MSP" zip: "55401"}}` and a client sends `<< user(address.zip: null) => {id};`
- **THEN** the system returns the records with `id` 1 and 2

#### Scenario: Dotted paths work in delete and merge filters
- **WHEN** the same collection holds `{id: 1 address: {city: "MSP"}}` and a client sends `!> user(address.city: "MSP");`
- **THEN** the system deletes that record and returns `1`

### Requirement: Subset filters on embedded objects
A filter condition whose value is an object, e.g. `(address: {city: "MSP"})`, SHALL match when the record's object is present and every condition inside it matches. Fields it doesn't mention SHALL be unconstrained, and conditions MAY nest. A `null` inside a subset filter SHALL match a present object that has no value for that field. `(address: null)` SHALL match records with no value for the object. `(address: {*})` SHALL match records where the object is present, whatever it contains; `*` MUST be the only thing in the braces. An empty object filter, `(address: {})`, SHALL be an error.

#### Scenario: Subset match ignores unmentioned fields
- **WHEN** collection `user { id: int @id address: { street: text city: text } @optional }` holds `{id: 1 address: {street: "1 Main" city: "MSP"}}` and a client sends `<< user(address: {city: "MSP"}) => {id};`
- **THEN** the system returns the record with `id` 1

#### Scenario: Null inside a subset requires the object
- **WHEN** collection `user { id: int @id address: { city: text zip: text @optional } @optional }` holds `{id: 1}` and `{id: 2 address: {city: "MSP"}}` and a client sends `<< user(address: {zip: null}) => {id};`
- **THEN** the system returns only the record with `id` 2

#### Scenario: Null object
- **WHEN** the same collection holds `{id: 1}` and `{id: 2 address: {city: "MSP"}}` and a client sends `<< user(address: null) => {id};`
- **THEN** the system returns only the record with `id` 1

#### Scenario: Wildcard object
- **WHEN** the same collection holds `{id: 1}` and `{id: 2 address: {city: "MSP"}}` and a client sends `<< user(address: {*}) => {id};`
- **THEN** the system returns only the record with `id` 2

#### Scenario: Nested subset
- **WHEN** collection `user { id: int @id address: { city: text geo: { lat: float } } }` holds `{id: 1 address: {city: "MSP" geo: {lat: 44.9}}}` and a client sends `<< user(address: {geo: {lat: 44.9}});`
- **THEN** the system returns `1`

#### Scenario: Empty object filter
- **WHEN** a client sends `<< user(address: {});`
- **THEN** the system returns `empty object filter for field "address"; use {*} to match any value`

#### Scenario: Wildcard combined with conditions
- **WHEN** a client sends `<< user(address: {* city: "MSP"});`
- **THEN** the system returns a parse error

### Requirement: The {*} wildcard is only allowed in a filter
The system SHALL reject `{*}` as a value anywhere other than a filter, such as in an insert record or a merge payload.

#### Scenario: Wildcard in an insert
- **WHEN** a client sends `>> user {id: 1 address: {*}};`
- **THEN** the system returns `{*} is only allowed in a filter` and stores nothing

### Requirement: A field can be constrained only once in a filter
The system SHALL reject a filter that constrains the same field more than once, including through a dotted key and a subset filter together, with `field "<path>" is given more than once`. Conditions on different fields of the same object SHALL be allowed together.

#### Scenario: Dotted key overlaps a subset
- **WHEN** a client sends `<< user(address.city: "A" address: {city: "B"});`
- **THEN** the system returns `field "address.city" is given more than once`

#### Scenario: Different fields of the same object
- **WHEN** collection `user { id: int @id address: { street: text city: text } }` holds `{id: 1 address: {street: "1 Main" city: "MSP"}}` and a client sends `<< user(address.city: "MSP" address: {street: "1 Main"});`
- **THEN** the system returns `1`
