## Why

Inserting several records means one statement per record, and nothing ties them together: if the fifth of ten fails, the first four are already stored. `QUERY_LANGUAGE.md` already designs a batch form (`>> user {..} & {..}`), and insert's result shape (`Count`, `Records`) was chosen with batches in mind. This implements it, keeping the "an invalid statement changes nothing" rule for the whole batch.

## What Changes

- An insert can list several records separated by `&`: `>> user {..} & {..} & {..}`. A trailing `&` means more records are coming, so the REPL waits for the next line.
- An optional `=> {...}` projection comes after the last record only, and applies to every record.
- **A batch is all-or-nothing.** Every record is validated before anything is stored, including duplicate primary keys within the batch itself. If anything is wrong, nothing is inserted.
- **Insert reports every problem, not just the first.** For a batch, each problem is prefixed with its record's position (`record 3: ...`). A single insert keeps its current messages, with no prefix, and lists all of its problems if it has more than one.
- The count is the number of records inserted. Auto-increment values are assigned in input order, and `=>` rows come back in input order.
- Internally there is one insert path: every insert is a batch, and a single insert is a batch of one.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `insert`: batch syntax, all-or-nothing batches, reporting every problem, and projections returning one row per record.
- `schema-enforcement`: "An invalid statement changes nothing" gains a batch scenario.

## Impact

- `core/lexer.go`: a `&` token.
- `core/parse_insert.go`: parse `&`-separated records; `InsertOperation.Record` becomes `Records []Entity`.
- `core/insert.go`: validate every record into a list of problems, check duplicates within the batch, then assign counters and store all.
- `core/validate.go`: per-record checks that collect problems instead of stopping at the first.
- Tests building `InsertOperation{Record: ...}` or reading `o.Record` switch to `Records`.
- Docs: README gains a batch section; `QUERY_LANGUAGE.md`'s batch example is updated; the Status line no longer lists batches as unsupported.
- Merge payloads and filters still report only their first error. Changing them is out of scope.
