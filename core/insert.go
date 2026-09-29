package core

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
		if collection.autoIncrement {
			if _, exists := record[collection.primaryKey]; exists {
				return nil, fmt.Errorf("field %q is auto-increment and must not be supplied for collection %q", collection.primaryKey, collectionName)
			}
			record[collection.primaryKey] = collection.nextAutoValue
			collection.nextAutoValue++
		} else if _, ok := record[collection.primaryKey]; !ok {
			return nil, fmt.Errorf("record missing primary key %q for collection %q", collection.primaryKey, collectionName)
		}

		key := record[collection.primaryKey]
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
