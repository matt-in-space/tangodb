## Purpose

Lets a client bulk-update every record in a collection matching a filter by merging new field values into each match, while making it structurally hard to update an entire collection by accident or to corrupt a record's identity through the update itself.

## ADDED Requirements

### Requirement: Merge statement requires explicit filter parens
The system SHALL require a `(...)` filter clause immediately after the collection name in a `~>` statement, even when the filter is empty. A `~>` statement with no filter parens at all SHALL be rejected as invalid, rather than treated as an implicit "update everything" shorthand.

#### Scenario: Explicit empty filter matches everything
- **WHEN** a client sends `~> user() {status: "inactive"}`
- **THEN** every record in the `user` collection has `status` merged in as `"inactive"`

#### Scenario: Omitted filter parens is rejected
- **WHEN** a client sends `~> user {status: "inactive"}` (no `(...)` at all)
- **THEN** the system returns an error and does not update anything

### Requirement: Merge updates every record matching the filter
The system SHALL merge the payload's fields into every record in the named collection whose fields match all of the given `field: value` equality filter conditions, and SHALL leave non-matching records untouched.

#### Scenario: Filter matches a subset of records
- **WHEN** a client sends `~> user(active: true) {plan: "pro"}` against a collection where some records have `active: true` and others have `active: false`
- **THEN** every record with `active: true` has `plan` merged in as `"pro"`, and records with `active: false` are unchanged

### Requirement: Merge is a partial update
The system SHALL only change the fields named in the payload on a matched record. Fields already present on that record but not named in the payload SHALL remain unchanged.

#### Scenario: Unmentioned fields survive the merge
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"}` against a record `{id: 1, name: "Sam", age: 40}`
- **THEN** the record becomes `{id: 1, name: "Matt", age: 40}` — `age` is untouched

### Requirement: Merge never creates records
The system SHALL NOT create a new record when a merge's filter matches nothing. A merge matching zero records SHALL be a no-op: no update, no creation, and no error.

#### Scenario: Filter matches nothing
- **WHEN** a client sends `~> user(id: 999) {name: "Matt"}` and no record has `id: 999`
- **THEN** the system returns success with an updated count of 0, and no record is created or changed

### Requirement: Merge payload must not set the primary key
The system SHALL reject, with an error and no mutation, any merge whose payload includes the collection's declared primary key field — regardless of whether that field is `@auto` or manually assigned. This restriction applies only to the payload; filtering on the primary key is unaffected.

#### Scenario: Payload includes the primary key
- **WHEN** a client sends `~> user(name: "Matt") {id: 5}` and `id` is the collection's declared primary key
- **THEN** the system returns an error and does not update anything

#### Scenario: Filtering on the primary key is still allowed
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"}` and `id` is the collection's declared primary key
- **THEN** the matching record's `name` is updated normally

### Requirement: Merge result defaults to a count
The system SHALL, when a merge statement has no `=>` projection clause, return only the number of records updated, not the updated records themselves.

#### Scenario: Merge with no projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"}` and one record matches
- **THEN** the system returns a result reporting 1 record updated, without including that record's field values

### Requirement: Merge result can opt into returning updated records
The system SHALL, when a merge statement includes an `=> {...}` projection clause, return the updated records themselves — reflecting their state after the merge — limited to the projected fields, in addition to the count.

#### Scenario: Merge with a projection
- **WHEN** a client sends `~> user(id: 1) {name: "Matt"} => {id, name};` and the matching record was `{id: 1, name: "Sam"}` before the merge
- **THEN** the system returns `{id: 1, name: "Matt"}` (the post-merge state) alongside an updated count of 1

### Requirement: Merge validates the collection and field names
The system SHALL return an error, without updating anything, if the named collection does not exist, or if any field referenced in the filter or projection is not declared in that collection's schema.

#### Scenario: Unknown collection
- **WHEN** a client sends `~> ghost(id: 1) {name: "Matt"}` and no `ghost` collection has been defined
- **THEN** the system returns an error and updates nothing

#### Scenario: Unknown field in filter
- **WHEN** a client sends `~> user(nope: 1) {name: "Matt"}` and `user` has no `nope` field
- **THEN** the system returns an error and updates nothing
