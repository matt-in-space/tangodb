package main

import "fmt"

type InsertOperation struct {
	Collection string
	Record     Entity
}

type InsertResult struct {
	Record Entity
}

func (db *Database) insert(collectionName string, record Entity) (OperationResult, error) {
	collection, ok := db.collections[collectionName]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", collectionName)
	}

	if collection.primaryKey != "" {
		key, ok := record[collection.primaryKey]
		if !ok {
			return nil, fmt.Errorf("record missing primary key %q for collection %q", collection.primaryKey, collectionName)
		}
		if _, exists := collection.primaryIndex[key]; exists {
			return nil, fmt.Errorf("duplicate primary key %v for collection %q", key, collectionName)
		}
	}

	id := collection.nextID
	collection.nextID++
	collection.records[id] = record

	if collection.primaryKey != "" {
		collection.primaryIndex[record[collection.primaryKey]] = id
	}

	return InsertResult{Record: record}, nil
}
