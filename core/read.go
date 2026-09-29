package core

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

type ReadOperation struct {
	Collection string
	Filter     map[string]any
	Projection []string
}

type ReadResult struct {
	Count      int
	Projection []string
	Records    []Entity
}

func (db *Database) read(collectionName string, filter map[string]any, projection []string) (OperationResult, error) {
	collection, ok := db.collections[collectionName]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", collectionName)
	}

	if err := validateFields(collection, filter); err != nil {
		return nil, err
	}

	wantRecords := projection != nil

	if wantRecords {
		projection = expandProjection(collection, projection)

		for _, field := range projection {
			if _, ok := collection.data[field]; !ok {
				return nil, fmt.Errorf("field %q not found in schema for collection %q", field, collectionName)
			}
		}
	}

	ids := make([]uint64, 0, len(collection.records))
	for id := range collection.records {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	count := 0

	var matches []Entity
	if wantRecords {
		matches = []Entity{}
	}

	for _, id := range ids {
		record := collection.records[id]
		if !recordMatchesFilter(record, filter) {
			continue
		}

		count++

		if wantRecords {
			matches = append(matches, record)
		}
	}

	result := ReadResult{Count: count}

	if wantRecords {
		result.Projection = projection
		result.Records = matches
	}

	return result, nil
}

func recordMatchesFilter(record Entity, filter map[string]any) bool {
	for field, want := range filter {
		if record[field] != want {
			return false
		}
	}
	return true
}

func (r ReadResult) String() string {
	return renderCountOrTable(r.Count, r.Projection, r.Records)
}

// isWildcardProjection reports whether a projection is the `{*}` sentinel
// produced by parseProjection() — a single element equal to "*" — as
// opposed to nil (no projection given) or an explicit field list.
func isWildcardProjection(projection []string) bool {
	return len(projection) == 1 && projection[0] == "*"
}

// expandProjection resolves the `{*}` wildcard to every field currently
// declared in the collection's schema, alphabetically ordered. Any other
// projection (an explicit field list) is returned unchanged. Callers only
// invoke this once they've already decided a projection was given at all
// (projection != nil) — nil itself means "count only" and never reaches
// here.
func expandProjection(collection *Collection, projection []string) []string {
	if !isWildcardProjection(projection) {
		return projection
	}

	fields := make([]string, 0, len(collection.data))
	for field := range collection.data {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	return fields
}

// renderCountOrTable is the shared rendering rule for every operation's
// result: a bare integer when no projection was given, "no records found"
// when one was given but nothing matched, and a table otherwise. Shared by
// ReadResult, DeleteResult, and MergeResult's String() methods.
func renderCountOrTable(count int, projection []string, records []Entity) string {
	if projection == nil {
		return strconv.Itoa(count)
	}

	if len(records) == 0 {
		return "no records found"
	}

	return renderTable(projection, records)
}

// renderTable formats records as a tab-aligned table, one column per
// projected field, in the given order.
func renderTable(projection []string, records []Entity) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, strings.Join(projection, "\t"))

	for _, record := range records {
		cells := make([]string, len(projection))
		for i, field := range projection {
			cells[i] = formatCell(record[field])
		}
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}

	w.Flush()

	return strings.TrimRight(buf.String(), "\n")
}

func formatCell(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
