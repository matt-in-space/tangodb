## Context

`InsertResult{Record}` and `DefineCollectionResult{Collection}` have no `String()` method, so the REPL's `%v` prints Go's struct and map syntax. Read, delete, and merge already share a result shape, `{Count, Projection, Records}`, rendered by `renderCountOrTable`: a bare integer when the projection is nil, `no records found` when a projection matches nothing, and a table otherwise. Delete's parser shows how to handle an optional trailing `=>`: at EOF where `=>` could still follow, it returns `ErrIncompleteInput`.

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Insert follows the same count-or-`=>` rule as the other operations.
- No Go formatting reaches the REPL for any result.

**Non-Goals:**
- Batch inserts (`&`). The result shape is chosen so a batch simply reports a larger count and more rows.
- Changing how tables look (quoting text, etc.). That's a separate output-formatting pass.

## Decisions

1. **Parse insert exactly like delete's tail.** After the record literal:
   - `=>` → `parseProjection`, normalizing nil to `[]string{}` so `=> {}` stays distinguishable from no `=>`, as delete does
   - EOF → `ErrIncompleteInput`
   - anything else → `expectEndOfStatement`, which accepts the optional `;`

2. **`InsertOperation` gains `Projection []string`**, with nil meaning count only, the same convention as the others.

3. **`InsertResult` becomes `{Count, Projection, Records}`**, with `String()` delegating to `renderCountOrTable`. For a single insert, `Count` is always 1 and `Records` holds the one stored record when a projection was given. A batch will use the same shape with more of each. Leaving the struct out of line with the others would just mean redoing it when batches land.

4. **The projection is validated before anything is written.** `expandProjection` and the "field not found in schema" check run alongside the other insert validation, before counters advance or the record is stored, so `>> user {...} => {nope}` stores nothing.

5. **Returned records are the stored record**, after `null`s are stripped and auto values assigned, so `=> {id}` returns the generated id and `=> {*}` shows `null` for any optional field with no value.

6. **`DefineCollectionResult.String()` returns `r.Collection.String()`.** That's the whole fix for the define output.

## Risks / Trade-offs

- [Every bare insert now needs `;` in the REPL] → Accepted: it's the same rule delete and merge already have, and it's what lets `=>` follow. Tests and docs gain `;`.
- [Seeing a generated id now takes `=> {id}`] → Deliberate, in line with the explicit-over-implicit principle and Postgres's `INSERT ... RETURNING`.
- [Many tests read `InsertResult.Record`] → Mechanical: they add `=> {id}` or `=> {*}` and read `Records[0]`.
