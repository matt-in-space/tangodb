package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
)

type ReadOperation struct {
	Collection string
	Filter     map[string]any
	Projection []string
}

type ReadResult struct {
	Projection []string
	Records    []Entity
}

func (db *Database) read(collectionName string, filter map[string]any, projection []string) (OperationResult, error) {
	collection, ok := db.collections[collectionName]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", collectionName)
	}

	if projection == nil {
		projection = make([]string, 0, len(collection.data))
		for field := range collection.data {
			projection = append(projection, field)
		}
		sort.Strings(projection)
	}

	for field := range filter {
		if _, ok := collection.data[field]; !ok {
			return nil, fmt.Errorf("field %q not found in schema for collection %q", field, collectionName)
		}
	}

	for _, field := range projection {
		if _, ok := collection.data[field]; !ok {
			return nil, fmt.Errorf("field %q not found in schema for collection %q", field, collectionName)
		}
	}

	ids := make([]uint64, 0, len(collection.records))
	for id := range collection.records {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var matches []Entity

	for _, id := range ids {
		record := collection.records[id]
		if recordMatchesFilter(record, filter) {
			matches = append(matches, record)
		}
	}

	return ReadResult{Projection: projection, Records: matches}, nil
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
	if len(r.Records) == 0 {
		return "no records found"
	}

	return renderTable(r.Projection, r.Records)
}

// renderTable formats records as a tab-aligned table, one column per
// projected field, in the given order. Shared by any result type that
// needs to print a set of records the same way (ReadResult, DeleteResult).
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
