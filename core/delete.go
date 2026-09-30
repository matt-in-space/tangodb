package core

import (
	"fmt"
	"sort"
)

type DeleteOperation struct {
	Collection string
	Filter     map[string]any
	Projection []string
}

type DeleteResult struct {
	Count      int
	Projection []string
	Records    []Entity
}

func (db *Database) delete(collectionName string, filter map[string]any, projection []string) (OperationResult, error) {
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

	var deleted []Entity
	if wantRecords {
		deleted = []Entity{}
	}

	for _, id := range ids {
		record := collection.records[id]
		if !recordMatchesFilter(record, filter) {
			continue
		}

		if collection.primaryKey != "" {
			delete(collection.primaryIndex, record[collection.primaryKey])
		}
		delete(collection.records, id)
		count++

		if wantRecords {
			deleted = append(deleted, record)
		}
	}

	result := DeleteResult{Count: count}

	if wantRecords {
		result.Projection = projection
		result.Records = deleted
	}

	return result, nil
}

func (r DeleteResult) String() string {
	return renderCountOrTable(r.Count, r.Projection, r.Records)
}
