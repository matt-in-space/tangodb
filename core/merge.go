package core

import (
	"fmt"
	"sort"
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

	if err := validateFields(collection, filter); err != nil {
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

	if err := validateFields(collection, payload); err != nil {
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

	var updated []Entity
	if wantRecords {
		updated = []Entity{}
	}

	for _, id := range ids {
		record := collection.records[id]
		if !recordMatchesFilter(record, filter) {
			continue
		}

		for field, value := range payload {
			if value == nil {
				delete(record, field)
				continue
			}
			record[field] = value
		}
		count++

		if wantRecords {
			updated = append(updated, record)
		}
	}

	result := MergeResult{Count: count}

	if wantRecords {
		result.Projection = projection
		result.Records = updated
	}

	return result, nil
}

func (r MergeResult) String() string {
	return renderCountOrTable(r.Count, r.Projection, r.Records)
}
