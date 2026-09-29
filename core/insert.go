package core

import "fmt"

type InsertOperation struct {
	Collection string
	Record     Entity
	Projection []string
}

type InsertResult struct {
	Count      int
	Projection []string
	Records    []Entity
}

func (db *Database) insert(collectionName string, record Entity, projection []string) (OperationResult, error) {
	collection, ok := db.collections[collectionName]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", collectionName)
	}

	autoFields := collection.sortedAutoFields()

	for _, field := range autoFields {
		if _, exists := record[field]; exists {
			return nil, fmt.Errorf("field %q is auto-increment and must not be supplied for collection %q", field, collectionName)
		}
	}

	manualKey := collection.primaryKey != "" && !collection.isAuto(collection.primaryKey)

	if manualKey && record[collection.primaryKey] == nil {
		return nil, fmt.Errorf("record missing primary key %q for collection %q", collection.primaryKey, collectionName)
	}

	if err := validateFields(collection, record); err != nil {
		return nil, err
	}

	if err := validateRequired(collection, record); err != nil {
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

	if manualKey {
		key := record[collection.primaryKey]
		if _, exists := collection.primaryIndex[key]; exists {
			return nil, fmt.Errorf("duplicate primary key %v for collection %q", key, collectionName)
		}
	}

	// A null value is stored as an absent field, so "no value" has one representation.
	for field, value := range record {
		if value == nil {
			delete(record, field)
		}
	}

	// Counters advance only once every check has passed, so a failed insert
	// never consumes a value. An auto primary key can't collide, since each
	// value is new.
	for _, field := range autoFields {
		record[field] = collection.autoCounters[field]
		collection.autoCounters[field]++
	}

	id := collection.nextID
	collection.nextID++
	collection.records[id] = record

	if collection.primaryKey != "" {
		collection.primaryIndex[record[collection.primaryKey]] = id
	}

	result := InsertResult{Count: 1}

	if wantRecords {
		result.Projection = projection
		result.Records = []Entity{record}
	}

	return result, nil
}

func (r InsertResult) String() string {
	return renderCountOrTable(r.Count, r.Projection, r.Records)
}
