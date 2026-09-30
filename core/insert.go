package core

import (
	"errors"
	"fmt"
	"strings"
)

type InsertOperation struct {
	Collection string
	Records    []Entity
	Projection []string
}

type InsertResult struct {
	Count      int
	Projection []string
	Records    []Entity
}

// insert stores a batch of records (a single insert is a batch of one). It's
// all-or-nothing: every record is validated, along with duplicate keys within
// the batch and the projection, and every problem found is reported. Nothing
// is written unless there are none.
func (db *Database) insert(collectionName string, records []Entity, projection []string) (OperationResult, error) {
	collection, ok := db.collections[collectionName]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", collectionName)
	}

	batch := len(records) > 1
	var problems []string

	addProblem := func(index int, problem string) {
		if batch {
			problem = fmt.Sprintf("record %d: %s", index+1, problem)
		}
		problems = append(problems, problem)
	}

	manualKey := collection.primaryKey != "" && !collection.isAuto(collection.primaryKey)
	seenKeys := map[any]bool{}

	for i, record := range records {
		recordErrs, bad := recordProblems(collection, record)
		for _, problem := range recordErrs {
			addProblem(i, problem)
		}

		if !manualKey || bad[collection.primaryKey] {
			continue
		}

		key := record[collection.primaryKey]
		_, stored := collection.primaryIndex[key]
		if stored || seenKeys[key] {
			addProblem(i, fmt.Sprintf("duplicate primary key %v for collection %q", key, collectionName))
		}
		seenKeys[key] = true
	}

	wantRecords := projection != nil

	if wantRecords {
		var projectionProblems []error
		projection, projectionProblems = resolveProjection(collection, projection)
		for _, err := range projectionProblems {
			problems = append(problems, err.Error())
		}
	}

	if len(problems) == 1 {
		return nil, errors.New(problems[0])
	}

	if len(problems) > 1 {
		return nil, fmt.Errorf("%d problems, nothing inserted:\n  %s", len(problems), strings.Join(problems, "\n  "))
	}

	// Every record is valid, so write them all, in input order. Counters
	// advance only now, so a failed insert never consumes a value, and auto
	// values follow the order the records were written in.
	for _, record := range records {
		stripNulls(record)

		for _, field := range collection.sortedAutoFields() {
			record[field] = collection.autoCounters[field]
			collection.autoCounters[field]++
		}

		id := collection.nextID
		collection.nextID++
		collection.records[id] = record

		if collection.primaryKey != "" {
			collection.primaryIndex[record[collection.primaryKey]] = id
		}
	}

	result := InsertResult{Count: len(records)}

	if wantRecords {
		result.Projection = projection
		result.Records = records
	}

	return result, nil
}

func (r InsertResult) String() string {
	return renderCountOrTable(r.Count, r.Projection, r.Records)
}

// stripNulls removes null values from a record and from every embedded
// object inside it, so "no value" is stored one way at every depth: as an
// absent field.
func stripNulls(record Entity) {
	for field, value := range record {
		switch v := value.(type) {
		case nil:
			delete(record, field)
		case Entity:
			stripNulls(v)
		}
	}
}
