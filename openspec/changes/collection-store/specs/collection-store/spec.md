## Purpose

Defines the contract between the query layer and storage for one collection: storing, finding, scanning, updating, and deleting its records; maintaining its primary index; and keeping its `@auto` counters. Any implementation, whether in memory or page-backed, must behave this way.

## ADDED Requirements

### Requirement: Records are stored encoded and returned as independent copies
A store SHALL keep each record as its encoded bytes, and SHALL return every record as a freshly decoded copy, so that changing a returned record never changes what is stored.

#### Scenario: Changing a returned record
- **WHEN** a record `{id: 1 name: "Matt"}` is inserted, looked up, and the returned record's `name` is changed to `"Sam"`
- **THEN** looking it up again returns `name` `"Matt"`

### Requirement: Insert is all-or-nothing
A store SHALL check every record in an insert before storing any of them. If any record is invalid for storage, the store SHALL return an error prefixed `storage: ` and store none of them. A record is invalid for storage if:
- it is missing its primary key, or the key has the wrong type
- its key duplicates a stored record's, or another record's in the same insert
- it can't be encoded
- it is larger than the maximum record size

#### Scenario: One record too large
- **WHEN** a store with no records is given an insert of three records, where the third encodes to more than 4076 bytes
- **THEN** the insert returns an error naming record 3 and the size limit, and a scan visits no records

#### Scenario: Duplicate key within the insert
- **WHEN** a store with primary key `id` is given `{id: 1}` and `{id: 1}` in one insert
- **THEN** the insert returns `storage: record 2: duplicate primary key 1 in collection "user"` and stores nothing

#### Scenario: Refs come back in input order
- **WHEN** three records are inserted together
- **THEN** the insert returns three refs, the first referring to the first record given

### Requirement: Records must fit in one page
A store SHALL reject any record whose encoding is larger than 4076 bytes (a 4096-byte page minus its 16-byte header and one 4-byte slot entry), until records spanning several pages are supported.

#### Scenario: Exactly the maximum
- **WHEN** a record whose encoding is exactly 4076 bytes is inserted
- **THEN** it is stored

#### Scenario: One byte over
- **WHEN** a record whose encoding is 4077 bytes is inserted on its own
- **THEN** the insert returns `storage: record is 4077 bytes; the maximum is 4076`

### Requirement: Records can be found by primary key
A store configured with a primary key of type `int` or `text` SHALL find a record by that key. A lookup of a key that isn't stored SHALL report not found, without an error. A lookup with a key of the wrong type, or on a store without a primary key, SHALL be an error.

#### Scenario: Found
- **WHEN** a store with an `int` primary key holds `{id: 42 name: "Matt"}` and key `42` is looked up
- **THEN** the lookup returns that record and a ref to it

#### Scenario: Not found
- **WHEN** key `7` is looked up in the same store
- **THEN** the lookup reports not found, with no error

#### Scenario: Wrong key type
- **WHEN** key `"42"` (text) is looked up in a store with an `int` primary key
- **THEN** the lookup returns an error

### Requirement: Scans visit every record once, in no promised order
A scan SHALL visit each record that existed when the scan started exactly once, unless it is deleted before the scan reaches it, and SHALL NOT visit records inserted during the scan. A scan SHALL NOT promise any order.

#### Scenario: Every record once
- **WHEN** a store holds five records and is scanned
- **THEN** the scan visits five records, each with a distinct ref

#### Scenario: Deleted during the scan
- **WHEN** a store holds three records, a scan starts, and a record the scan hasn't reached yet is deleted
- **THEN** the scan visits the other two

### Requirement: Records can be updated and deleted by ref
A store SHALL replace a record by its ref, and delete a record by its ref along with its index entry. An update SHALL be rejected if the ref is unknown, if it would change or remove the primary key, or if the new record can't be encoded or is too large, and the stored record SHALL then be unchanged. A delete of an unknown ref SHALL be an error.

#### Scenario: Update in place
- **WHEN** `{id: 1 name: "Matt"}` is updated by its ref to `{id: 1 name: "Sam"}`
- **THEN** looking up key `1` returns `name` `"Sam"`

#### Scenario: Update cannot change the key
- **WHEN** `{id: 1 name: "Matt"}` is updated by its ref to `{id: 2 name: "Matt"}`
- **THEN** the update returns an error, and key `1` still returns the original record

#### Scenario: Delete
- **WHEN** a record is deleted by its ref
- **THEN** looking up its key reports not found, and a scan doesn't visit it

### Requirement: Auto-increment counters advance only on success
A store SHALL keep a counter for each configured `@auto` field, starting at 1, and SHALL report its next value. After a successful insert, each counter SHALL be one past the highest value inserted for its field. A failed insert SHALL leave every counter unchanged.

#### Scenario: Counter after an insert
- **WHEN** a store with `@auto` field `id` is given records with `id` 1 and 2
- **THEN** the next value for `id` is 3

#### Scenario: Failed insert keeps the counter
- **WHEN** an insert whose second record is too large is given records with `id` 1 and 2
- **THEN** the insert fails, and the next value for `id` is still 1

### Requirement: Storage errors are prefixed
Every error a store returns SHALL begin with `storage: `, so it can be told apart from errors about the query language.

#### Scenario: Prefixed error
- **WHEN** a delete is given an unknown ref
- **THEN** the error begins with `storage: `
