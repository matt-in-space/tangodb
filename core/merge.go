package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type MergeOperation struct {
	Collection string
	Filter     map[string]any
	Payload    Entity
	Projection []string
}

type MergeResult struct {
	Count      int
	Projection []string
	Records    []Entity
}

func (db *Database) merge(collectionName string, filter map[string]any, payload Entity, projection []string) (OperationResult, error) {
	collection, ok := db.collections[collectionName]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", collectionName)
	}

	if err := validateFilter(collection, filter); err != nil {
		return nil, err
	}

	if collection.primaryKey != "" {
		if _, ok := payload[collection.primaryKey]; ok {
			return nil, fmt.Errorf("payload must not set primary key %q for collection %q", collection.primaryKey, collectionName)
		}
	}

	autoFields := collection.sortedAutoFields()

	for _, field := range autoFields {
		if _, ok := payload[field]; ok {
			return nil, fmt.Errorf("payload must not set auto-increment field %q for collection %q", field, collectionName)
		}
	}

	if err := validatePayload(collection, payload); err != nil {
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

	// Work out every matched record's new state first, and check each one
	// against the schema. How a record ends up can depend on what it held
	// before (merging into a missing object creates it, which may leave it
	// incomplete), so the results are what get validated. Nothing is written
	// unless every result is valid.
	var matchedIDs []uint64
	var merged []Entity
	var problems []string

	for _, id := range ids {
		record := collection.records[id]
		if !recordMatchesFilter(record, filter) {
			continue
		}

		next := deepMerge(record, payload)
		label := mergedRecordLabel(collection, next, len(merged)+1)

		for _, problem := range objectProblems(collectionName, collection.rootSchema(), "", next, nil) {
			problems = append(problems, label+problem.err.Error())
		}

		matchedIDs = append(matchedIDs, id)
		merged = append(merged, next)
	}

	if len(problems) == 1 {
		return nil, errors.New(problems[0])
	}

	if len(problems) > 1 {
		return nil, fmt.Errorf("%d problems, nothing changed:\n  %s", len(problems), strings.Join(problems, "\n  "))
	}

	for i, id := range matchedIDs {
		collection.records[id] = merged[i]
	}

	result := MergeResult{Count: len(merged)}

	if wantRecords {
		result.Projection = projection
		result.Records = merged
	}

	return result, nil
}

func (r MergeResult) String() string {
	return renderCountOrTable(r.Count, r.Projection, r.Records)
}

// deepMerge returns a new record: stored with the payload merged in at every
// depth. A null removes a value, an object merges into the object already
// there (or into a new one if there isn't one), and anything else is set.
// Every object it writes is a fresh copy, so records never share a nested
// object with the payload or with each other.
func deepMerge(stored, payload Entity) Entity {
	merged := Entity{}
	for field, value := range stored {
		merged[field] = value
	}

	for field, value := range payload {
		switch v := value.(type) {
		case nil:
			delete(merged, field)
		case Entity:
			existing, _ := merged[field].(Entity)
			merged[field] = deepMerge(existing, v)
		default:
			merged[field] = value
		}
	}

	return merged
}

// mergedRecordLabel names a matched record in a problem: by its primary key
// when the collection has one, otherwise by its position among the matched
// records, counting from 1 (the order they'd be returned in).
func mergedRecordLabel(collection *Collection, record Entity, position int) string {
	if collection.primaryKey == "" {
		return fmt.Sprintf("matched record %d: ", position)
	}

	key := record[collection.primaryKey]
	if text, ok := key.(string); ok {
		return fmt.Sprintf("record with %s %q: ", collection.primaryKey, text)
	}
	return fmt.Sprintf("record with %s %v: ", collection.primaryKey, key)
}
