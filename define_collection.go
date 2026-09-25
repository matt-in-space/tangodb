package main

import "fmt"

type DefineCollectionOperation struct {
	Name       string
	Data       map[string]DataType
	PrimaryKey string
}

type DefineCollectionResult struct {
	Collection Collection
}

func (db *Database) defineCollection(name string, fields map[string]DataType, primaryKey string) (OperationResult, error) {
	if primaryKey != "" {
		if _, ok := fields[primaryKey]; !ok {
			return nil, fmt.Errorf("primary key %q not found in schema for collection %q", primaryKey, name)
		}
	}

	collection := Collection{
		name:       name,
		data:       fields,
		records:    make(map[any]Entity),
		primaryKey: primaryKey,
	}

	db.collections[name] = collection

	return DefineCollectionResult{Collection: collection}, nil
}
