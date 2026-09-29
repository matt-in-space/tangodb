package main

import "fmt"

type DefineCollectionOperation struct {
	Name          string
	Data          map[string]DataType
	PrimaryKey    string
	AutoIncrement bool
}

type DefineCollectionResult struct {
	Collection Collection
}

func (db *Database) defineCollection(name string, fields map[string]DataType, primaryKey string, autoIncrement bool) (OperationResult, error) {
	if primaryKey != "" {
		if _, ok := fields[primaryKey]; !ok {
			return nil, fmt.Errorf("primary key %q not found in schema for collection %q", primaryKey, name)
		}
	}

	if autoIncrement {
		if primaryKey == "" {
			return nil, fmt.Errorf("@auto requires a primary key for collection %q", name)
		}
		if fields[primaryKey] != TypeInt {
			return nil, fmt.Errorf("@auto requires primary key %q to be int for collection %q", primaryKey, name)
		}
	}

	collection := &Collection{
		name:          name,
		data:          fields,
		records:       make(map[uint64]Entity),
		primaryKey:    primaryKey,
		primaryIndex:  make(map[any]uint64),
		autoIncrement: autoIncrement,
		nextAutoValue: 1,
	}

	db.collections[name] = collection

	return DefineCollectionResult{Collection: *collection}, nil
}
