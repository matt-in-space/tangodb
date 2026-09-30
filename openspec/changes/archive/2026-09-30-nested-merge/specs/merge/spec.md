## REMOVED Requirements

### Requirement: Merge payload cannot set an embedded object yet
**Reason**: Merge payloads can now update embedded objects, with deep merge.
**Migration**: Use `~> user(id: 1) {address: {city: "STP"}};` or `~> user(id: 1) {address.city: "STP"};` to update a field inside an object, and `{address: null}` to clear an optional object.

## ADDED Requirements

### Requirement: Merge into an embedded object is a deep merge
When a merge payload gives an object for an embedded object field, the system SHALL merge it into the stored object field by field, at every depth. Fields the payload doesn't mention SHALL be left unchanged. A `null` SHALL remove a value, including a whole optional object. If the record has no value for the object, merging into it SHALL create it. An empty payload object SHALL be an error.

#### Scenario: Updating one field of an object
- **WHEN** collection `user { id: int @id address: { street: text city: text zip: text @optional } }` holds `{id: 1 address: {street: "1 Main" city: "MSP" zip: "55401"}}` and a client sends `~> user(id: 1) {address: {city: "STP"}};`
- **THEN** the record's address becomes `{street: "1 Main" city: "STP" zip: "55401"}`

#### Scenario: Clearing a field inside an object
- **WHEN** the same record exists and a client sends `~> user(id: 1) {address: {zip: null}};`
- **THEN** the record's address has no `zip`, and `street` and `city` are unchanged

#### Scenario: Clearing an optional object
- **WHEN** collection `user { id: int @id address: { city: text } @optional }` holds `{id: 1 address: {city: "MSP"}}` and a client sends `~> user(id: 1) {address: null};`
- **THEN** `<< user(address: null);` returns `1`

#### Scenario: Creating a missing object
- **WHEN** collection `user { id: int @id address: { city: text } @optional }` holds `{id: 1}` and a client sends `~> user(id: 1) {address: {city: "MSP"}};`
- **THEN** `<< user(address.city: "MSP");` returns `1`

#### Scenario: Empty payload object
- **WHEN** a client sends `~> user(id: 1) {address: {}};`
- **THEN** the system returns `empty object in merge payload for field "address"` and changes nothing

### Requirement: Merge payloads accept dotted keys
A merge payload key SHALL be allowed to be a dotted path (`address.city`), anywhere in the payload, meaning the same deep update as the nested form. The system SHALL reject a payload that sets the same field twice (`field "<path>" is given more than once`), or that sets a value while also setting a path beneath it (`field "<child>" conflicts with "<parent>"`). Dotted keys SHALL remain an error in insert records.

#### Scenario: Dotted deep update
- **WHEN** collection `user { id: int @id address: { street: text city: text } }` holds `{id: 1 address: {street: "1 Main" city: "MSP"}}` and a client sends `~> user(id: 1) {address.city: "STP"};`
- **THEN** the record's address becomes `{street: "1 Main" city: "STP"}`

#### Scenario: Same field set twice
- **WHEN** a client sends `~> user(id: 1) {address.city: "A" address: {city: "B"}};`
- **THEN** the system returns `field "address.city" is given more than once` and changes nothing

#### Scenario: Clearing an object while setting inside it
- **WHEN** a client sends `~> user(id: 1) {address: null address.city: "X"};`
- **THEN** the system returns `field "address.city" conflicts with "address"` and changes nothing

#### Scenario: Dotted keys are still rejected in inserts
- **WHEN** a client sends `>> user {id: 1 address.city: "MSP"};`
- **THEN** the system returns `field name "address.city" cannot contain "."`

### Requirement: Merge payload values are validated at their path
The system SHALL validate every value in a merge payload against the schema at its full path before changing anything: unknown paths, type mismatches, wrong shapes, and `null` on a required field SHALL be errors naming the full path. Fields the payload leaves out SHALL NOT be required.

#### Scenario: Wrong type at a nested path
- **WHEN** collection `user { id: int @id address: { city: text } }` holds a record and a client sends `~> user() {address.city: 5};`
- **THEN** the system returns `field "address.city": expected text, got int` and changes nothing

#### Scenario: Unknown nested path
- **WHEN** a client sends `~> user() {address: {zip: "55401"}};` against the same collection
- **THEN** the system returns `field "address.zip" not found in schema for collection "user"`

#### Scenario: Clearing a required object
- **WHEN** a client sends `~> user() {address: null};` against the same collection
- **THEN** the system returns `field "address" is required and cannot be null`

### Requirement: Merge leaves every matched record valid
The system SHALL compute the result of a merge for every matched record and validate each result against the collection's schema before changing anything. If any result would be invalid, the system SHALL change no record and SHALL report every problem, each prefixed with the record it belongs to: `record with id <key>: ` for a collection with a primary key, otherwise `matched record <n>: `, counting matched records from 1 in the order they'd be returned. One problem SHALL be reported alone; several SHALL be listed under `<N> problems, nothing changed:`.

#### Scenario: Creating an object that would be incomplete
- **WHEN** collection `user { id: int @id address: { street: text city: text } @optional }` holds `{id: 1 address: {street: "1 Main" city: "MSP"}}` and `{id: 2}` and a client sends `~> user() {address: {city: "STP"}};`
- **THEN** the system returns `record with id 2: field "address.street" is required for collection "user"` and neither record changes

#### Scenario: Several invalid results
- **WHEN** the same collection holds `{id: 2}` and `{id: 3}` and a client sends `~> user(address: null) {address: {city: "STP"}};`
- **THEN** the system returns an error beginning `2 problems, nothing changed:` listing `record with id 2: field "address.street" is required for collection "user"` and `record with id 3: field "address.street" is required for collection "user"`

#### Scenario: Collection without a primary key
- **WHEN** collection `log { message: text meta: { source: text level: text } @optional }` holds `{message: "a"}` and a client sends `~> log() {meta.level: "warn"};`
- **THEN** the system returns `matched record 1: field "meta.source" is required for collection "log"`
