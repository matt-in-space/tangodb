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

	if err := validateFilter(collection, filter); err != nil {
		return nil, err
	}

	wantRecords := projection != nil

	if wantRecords {
		var problems []error
		projection, problems = resolveProjection(collection, projection)
		if len(problems) > 0 {
			return nil, problems[0]
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
	// A field with no value is absent from the record, so record[field] is nil
	// and matches a null filter value by plain equality.
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

// resolveProjection turns a projection into the table's columns, one per
// leaf field, named by dotted path. `{*}` becomes every leaf path in the
// schema, sorted; naming an embedded object field expands to its leaf paths;
// a dotted path to a leaf is its own column. Repeated columns are kept once,
// in the order first given. Every field that isn't in the schema is reported.
// Callers only invoke this once they've decided a projection was given at all
// (projection != nil); nil itself means "count only".
func resolveProjection(collection *Collection, projection []string) ([]string, []error) {
	root := collection.rootSchema()

	if isWildcardProjection(projection) {
		// Never nil: a nil projection means "count only" to the renderer.
		columns := append([]string{}, leafPaths(root, "")...)
		sort.Strings(columns)
		return columns, nil
	}

	columns := []string{} // never nil, as above
	var problems []error
	seen := map[string]bool{}

	for _, field := range projection {
		leaves, ok := projectField(root, field)
		if !ok {
			problems = append(problems, fmt.Errorf("field %q not found in schema for collection %q", field, collection.name))
			continue
		}

		for _, leaf := range leaves {
			if !seen[leaf] {
				seen[leaf] = true
				columns = append(columns, leaf)
			}
		}
	}

	return columns, problems
}

// projectField resolves one projected field, which may be a dotted path, to
// the leaf paths it covers: itself for a scalar field, or every leaf beneath
// it, sorted, for an embedded object. ok is false if the path isn't in the
// schema.
func projectField(schema *Schema, field string) (leaves []string, ok bool) {
	segments := strings.Split(field, ".")

	for i, segment := range segments {
		dataType, declared := schema.Data[segment]
		if !declared {
			return nil, false
		}

		last := i == len(segments)-1

		if dataType != TypeObject {
			if !last {
				return nil, false
			}
			return []string{field}, true
		}

		schema = schema.Objects[segment]

		if last {
			leaves := leafPaths(schema, field+".")
			sort.Strings(leaves)
			return leaves, true
		}
	}

	return nil, false
}

// leafPaths lists the dotted paths of every scalar field in a block,
// recursing into embedded blocks, each prefixed with path.
func leafPaths(schema *Schema, path string) []string {
	var leaves []string

	for field, dataType := range schema.Data {
		if dataType == TypeObject {
			leaves = append(leaves, leafPaths(schema.Objects[field], path+field+".")...)
			continue
		}
		leaves = append(leaves, path+field)
	}

	return leaves
}

// resolvePath looks up a dotted path in a record, stepping through embedded
// objects. It returns nil when any step has no value, so a field of an absent
// optional object reads as null.
func resolvePath(record Entity, path string) any {
	var value any = record

	for _, segment := range strings.Split(path, ".") {
		object, ok := value.(Entity)
		if !ok {
			return nil
		}
		value = object[segment]
	}

	return value
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
			cells[i] = formatCell(resolvePath(record, field))
		}
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}

	w.Flush()

	return strings.TrimRight(buf.String(), "\n")
}

func formatCell(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%v", v)
}
