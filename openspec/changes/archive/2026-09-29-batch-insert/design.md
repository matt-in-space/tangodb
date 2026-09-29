## Context

`db.insert` validates one record and stops at the first problem, in this order: a supplied `@auto` field, a missing manual primary key, `validateFields` (undeclared fields, `null` on required fields, type mismatches, in sorted field order), `validateRequired` (missing required fields, sorted), then the duplicate-key check against `primaryIndex`. After that it strips `null`s, assigns auto counters, and stores. `InsertOperation` carries a single `Record`. `InsertResult` already has `Count` and `Records`.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- `&`-separated batches, validated as a unit.
- Every problem in the statement reported at once.
- One insert code path.

**Non-Goals:**
- Collecting every problem for merge payloads or filters. They keep reporting the first.
- Best-effort or partial batches.
- A size limit.

## Decisions

1. **A `tokenAmp` token for `&`.** It has no other meaning in the language yet.

2. **Parsing.** After the collection name, `parseInsert` reads a record literal, then while the next token is `&`: consume it; at EOF return `ErrIncompleteInput` (more records are coming); otherwise read another record literal. The existing tail follows: optional `=>`, EOF means incomplete, then `expectEndOfStatement`. So `=>` can only follow the last record, and `& ;` fails with the record literal's "expected {" error.

3. **`InsertOperation.Records []Entity`** replaces `Record`. A single insert has one element. `db.insert` takes the slice.

4. **Problems are collected per field, first check wins.** Each record goes through the same checks in the same order as today, but instead of returning, each problem is appended to a list. Once a field has a problem, later checks skip that field. So `{id: null}` on an `@auto` id reports only "must not be supplied", not also "cannot be null", and single-problem messages stay exactly what they are today. The duplicate-key check only runs if the primary key field had no problem.

   The field-level checks move into a helper that returns the record's problems as a `[]string`, e.g. `recordProblems(collection, record) []string`. `validateFields` stays as-is for merge and filters.

5. **Duplicates within the batch.** While validating, a `seen` map records each valid manual primary key value and its record's position. A later record with the same key gets `duplicate primary key 1 for collection "user"`, the same message as a clash with a stored record. The `record N:` prefix says which record it is.

6. **The projection is validated once**, after the records, and a bad projection field is one more problem.

7. **Reporting.** With no problems, continue. Otherwise return one error:
   - Each problem is prefixed with `record N: ` (1-based) when the batch has more than one record; a single insert gets no prefix.
   - Exactly one problem: the error message is that line alone, which keeps every current single-insert error unchanged.
   - More than one: `N problems, nothing inserted:` followed by each problem on its own line, indented two spaces.

8. **Writing happens only after every record is valid.** Then, in input order, for each record: strip `null`s, assign auto counters (sorted field order within the record), store, and index. `Count` is `len(records)`. With a projection, `Records` holds the stored records in input order.

## Risks / Trade-offs

- [A single insert with several problems now gets a multi-line error instead of just the first] → Intended: that's the "report everything" decision. Single-problem messages don't change, so existing tests still hold.
- [Merge and filters report differently from insert] → Accepted for now (see Non-Goals); easy to align later using the same helper.
