## Purpose

Lets a client bulk-update every record in a collection matching a filter by merging new field values into each match, while making it structurally hard to update an entire collection by accident or to corrupt a record's identity through the update itself.

## Requirements

### Requirement: Merge statement requires explicit filter parens
The system SHALL require a `(...)` filter clause immediately after the collection name in a `~>` statement, even when the filter is empty. A `~>` statement with no filter parens at all SHALL be rejected as invalid, rather than treated as an implicit "update everything" shorthand.

#### Scenario: Explicit empty filter matches everything
- **WHEN** a client sends `~> user() {status: "inactive"};`
- **THEN** every record in the `user` collection has `status` merged in as `"inactive"`

#### Scenario: Omitted filter parens is rejected
- **WHEN** a client sends `~> user {status: "inactive"};` (no `(...)` at all)
- **THEN** the system returns an error and does not update anything

### Requirement: Merge updates every record matching the filter
The system SHALL merge the payload's fields into every record in the named collection whose fields match all of the given `field: value` equality filter conditions, and SHALL leave non-matching records untouched.

#### Scenario: Filter matches a subset of records
- **WHEN** a client sends `~> user(active: true) {plan: "pro"};` against a collection where some records have `active: true` and others have `active: false`
- **THEN** every record with `active: true` has `plan` merged in as `"pro"`, and records with `active: false` are unchanged

### Requirement: Merge is a partial update
The system SHALL only change the fields named in the payload on a matched record. Fields already present on that record but not named in the payload SHALL remain unchanged.

#### Scenario: Unmentioned fields survive the merge
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"};` against a record `{id: 1, name: "Sam", age: 40}`
- **THEN** the record becomes `{id: 1, name: "Matt", age: 40}` — `age` is untouched

### Requirement: Merge never creates records
The system SHALL NOT create a new record when a merge's filter matches nothing. A merge matching zero records SHALL be a no-op: no update, no creation, and no error.

#### Scenario: Filter matches nothing
- **WHEN** a client sends `~> user(id: 999) {name: "Matt"};` and no record has `id: 999`
- **THEN** the system returns success with an updated count of 0, and no record is created or changed

### Requirement: Merge payload must not set the primary key
The system SHALL reject, with an error and no mutation, any merge whose payload includes the collection's declared primary key field — regardless of whether that field is `@auto` or manually assigned. This restriction applies only to the payload; filtering on the primary key is unaffected.

#### Scenario: Payload includes the primary key
- **WHEN** a client sends `~> user(name: "Matt") {id: 5};` and `id` is the collection's declared primary key
- **THEN** the system returns an error and does not update anything

#### Scenario: Filtering on the primary key is still allowed
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"};` and `id` is the collection's declared primary key
- **THEN** the matching record's `name` is updated normally

### Requirement: Merge result defaults to a count
The system SHALL, when a merge statement has no `=>` projection clause, return only the number of records updated, not the updated records themselves, displayed as a bare integer with no surrounding words.

#### Scenario: Merge with no projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"};` and one record matches
- **THEN** the system returns an updated count of 1, displayed as the bare text `1` — not `"1 updated"` or any other wording — without including that record's field values

### Requirement: Merge result can opt into returning updated records
The system SHALL, when a merge statement includes an `=> {...}` projection clause, return the updated records themselves — reflecting their state after the merge — limited to the projected fields, in addition to the count. The projection MAY be a wildcard `*` in place of a field list, meaning every field currently declared in the collection's schema; `*` MUST be the sole content of the projection — combining it with named fields SHALL be rejected as a parse error.

#### Scenario: Merge with a projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"} => {id, name};` and the matching record was `{id: 1, name: "Sam"}` before the merge
- **THEN** the system returns `{id: 1, name: "Matt"}` (the post-merge state) alongside an updated count of 1

#### Scenario: Merge with a wildcard projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"} => {*};` against a schema with fields `id`, `name`, and `age`, and the matching record was `{id: 1, name: "Sam", age: 40}` before the merge
- **THEN** the system returns all three fields reflecting the post-merge state (`{id: 1, name: "Matt", age: 40}`) alongside an updated count of 1

#### Scenario: Wildcard cannot be combined with named fields
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"} => {*, id};` or `~> user(id: 1) {name: "Matt"} => {id, *};`
- **THEN** the system returns a parse error and does not update anything

### Requirement: Merge validates the collection and field names
The system SHALL return an error, without updating anything, if the named collection does not exist, or if any field referenced in the filter or projection is not declared in that collection's schema.

#### Scenario: Unknown collection
- **WHEN** a client sends `~> ghost(id: 1) {name: "Matt"};` and no `ghost` collection has been defined
- **THEN** the system returns an error and updates nothing

#### Scenario: Unknown field in filter
- **WHEN** a client sends `~> user(nope: 1) {name: "Matt"};` and `user` has no `nope` field
- **THEN** the system returns an error and updates nothing

### Requirement: Merge payload must not set an auto-increment field
The system SHALL reject, with an error and no mutation, any merge whose payload includes an `@auto` field. If the field is also the primary key, the existing primary-key error SHALL be reported instead. Filtering on an `@auto` field is unaffected.

#### Scenario: Payload sets a non-id auto field
- **WHEN** collection `ticket { code: text @id number: int @auto }` holds a record and a client sends `~> ticket() {number: 5};`
- **THEN** the system returns `payload must not set auto-increment field "number" for collection "ticket"` and changes no record

#### Scenario: Filtering on an auto field
- **WHEN** collection `ticket { code: text @id number: int @auto title: text @optional }` holds `{code: "A" number: 1}` and a client sends `~> ticket(number: 1) {title: "hi"};`
- **THEN** the system updates that record's `title`

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
