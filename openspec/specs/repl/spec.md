## Purpose

Defines how the interactive REPL reads input: when it decides a statement is complete, and how it reports a statement that is wrong.

## Requirements

### Requirement: A statement is parsed only once its brackets balance
The system SHALL NOT parse a statement while any `{` or `(` opened in it is still unclosed. It SHALL keep reading lines, showing the continuation prompt, and parse the full statement once every opened bracket is closed. Brackets inside quoted strings SHALL NOT count.

#### Scenario: Multi-line statement with a mistake reports one error
- **WHEN** a client enters, line by line, `>> user {`, `  id: 1`, `  name: Matt`, `  age: 3`, `};`
- **THEN** the REPL shows continuation prompts until `};`, then prints the single error `expected a value, got "Matt"`, and does not report errors for `age: 3` or `};`

#### Scenario: Brace inside a string does not hold the statement open
- **WHEN** collection `user { id: int @id name: text }` exists and a client enters `>> user {id: 1 name: "{"};`
- **THEN** the REPL runs the insert immediately and prints `1`

#### Scenario: Unmatched closing bracket is reported immediately
- **WHEN** a client enters `};`
- **THEN** the REPL prints an error right away and returns to the main prompt

### Requirement: Nothing after an error is re-parsed as a new statement
When a complete statement fails to parse, the system SHALL discard that entire statement, and the next line SHALL begin a new statement.

#### Scenario: Next statement runs normally after an error
- **WHEN** a client enters a multi-line insert containing a mistake, then `<< user;`
- **THEN** the REPL prints one error for the insert, then the count for `<< user;`
