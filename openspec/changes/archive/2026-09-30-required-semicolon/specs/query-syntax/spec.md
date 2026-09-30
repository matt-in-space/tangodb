## MODIFIED Requirements

### Requirement: Commas between list entries are optional
The system SHALL accept entries in schema blocks, record literals, filters, and projections separated by commas, by whitespace alone, or by any mix of the two. A comma SHALL carry no meaning beyond separating entries.

#### Scenario: Record literal without commas
- **WHEN** a client sends `>> user {id: 1 name: "Matt"};`
- **THEN** the system inserts a record with `id` 1 and `name` "Matt", exactly as it would for `>> user {id: 1, name: "Matt"};`

#### Scenario: Filter without commas
- **WHEN** a client sends `<< user(id: 1 name: "Matt") => {id};`
- **THEN** the system applies both filter conditions, exactly as it would for `<< user(id: 1, name: "Matt") => {id};`

#### Scenario: Projection without commas
- **WHEN** a client sends `<< user() => {id name};`
- **THEN** the system returns the `id` and `name` fields, exactly as it would for `<< user() => {id, name};`

#### Scenario: Schema block with commas
- **WHEN** a client sends `user { id: int @id, name: text };`
- **THEN** the system defines the collection exactly as it would for `user { id: int @id name: text };`

#### Scenario: Mixed and trailing commas
- **WHEN** a client sends `>> user {id: 1, name: "Matt" age: 39,};`
- **THEN** the system inserts a record with all three fields

### Requirement: Commas inside strings are not separators
The system SHALL treat a comma inside a quoted string as part of the string's value.

#### Scenario: String value containing a comma
- **WHEN** a client sends `>> user {name: "Smith, Matt"};`
- **THEN** the stored `name` is exactly `Smith, Matt`

### Requirement: Boolean literals
The system SHALL accept the bare words `true` and `false` as boolean values anywhere a value is expected: record literals, filters, and merge payloads. They SHALL be distinct from the quoted strings `"true"` and `"false"`, which remain text.

#### Scenario: Boolean in a record literal
- **WHEN** collection `user { id: int @id active: bool }` exists and a client sends `>> user {id: 1 active: false};`
- **THEN** the system stores `active` as the boolean `false`

#### Scenario: Boolean in a filter
- **WHEN** a client sends `<< user(active: true) => {id};`
- **THEN** the system filters on the boolean `true`

#### Scenario: Quoted true is text, not a boolean
- **WHEN** collection `user { id: int @id active: bool }` exists and a client sends `>> user {id: 1 active: "true"};`
- **THEN** the system returns a type error, because `"true"` is text

#### Scenario: Other bare words are not values
- **WHEN** a client sends `>> user {id: 1 active: yes};`
- **THEN** the system returns a parse error saying a value was expected

### Requirement: Null literal
The system SHALL accept the bare word `null` as a value anywhere a value is expected: record literals, filters, and merge payloads. It means "no value". The quoted string `"null"` SHALL remain text.

#### Scenario: Null in a record literal
- **WHEN** collection `user { id: int @id nickname: text @optional }` exists and a client sends `>> user {id: 1 nickname: null};`
- **THEN** the system parses `nickname` as having no value

#### Scenario: Null in a filter
- **WHEN** a client sends `<< user(nickname: null);`
- **THEN** the system filters for records with no value in `nickname`

#### Scenario: Quoted null is text
- **WHEN** collection `user { id: int @id nickname: text @optional }` exists and a client sends `>> user {id: 1 nickname: "null"};`
- **THEN** the system stores the text `null` as the value of `nickname`

## ADDED Requirements

### Requirement: Every statement ends with a semicolon
Every statement SHALL end with `;`, including collection definitions and statements with a `=>` projection. A statement SHALL be complete at its first `;` outside any brackets. Until then it SHALL be treated as incomplete, and not parsed, however many lines it spans, even if it would otherwise be a valid statement. Only one statement SHALL be accepted at a time: anything after the `;` SHALL be an error, and nothing SHALL run.

#### Scenario: Definition waits for a semicolon
- **WHEN** a client enters `user { id: int @id }` in the REPL
- **THEN** the REPL shows a continuation prompt and defines nothing yet

#### Scenario: Semicolon completes a definition
- **WHEN** a client enters `user { id: int @id };`
- **THEN** the system defines the collection

#### Scenario: Statement with a projection still needs a semicolon
- **WHEN** collection `user { id: int @id }` exists and a client enters `<< user => {*}` in the REPL
- **THEN** the REPL shows a continuation prompt, and runs the read once `;` is entered

#### Scenario: Semicolon on a later line
- **WHEN** a client enters `>> user {id: 1}`, then an empty line, then `;`
- **THEN** the REPL keeps reading until the `;` line, then runs the insert once

#### Scenario: Mistakes are reported once the statement is complete
- **WHEN** a client enters `>> user {id: 1} nonsense` in the REPL
- **THEN** the REPL shows a continuation prompt instead of an error, and reports the error once `;` is entered

#### Scenario: One statement at a time
- **WHEN** a client sends `>> user {id: 1}; << user;`
- **THEN** the system returns `unexpected input after statement: "<<"` and runs neither statement
