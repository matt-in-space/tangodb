## ADDED Requirements

### Requirement: Exit is a REPL command, not a statement
The REPL SHALL end the session when a line contains only the word `exit` (ignoring case and surrounding whitespace), with no `;`, when no statement is in progress. `exit` is not part of the query language and is never run against the database.

#### Scenario: Exit without a semicolon
- **WHEN** a client enters `exit` at the main prompt
- **THEN** the REPL ends the session

#### Scenario: Exit is not a statement
- **WHEN** a client enters `EXIT` with surrounding spaces at the main prompt
- **THEN** the REPL ends the session without running anything
