package core

import "fmt"

type DefineCollectionOperation struct {
	Name       string
	Data       map[string]DataType
	PrimaryKey string
	AutoFields map[string]bool
	Optional   map[string]bool
}

type DefineCollectionResult struct {
	Collection Collection
}

func (r DefineCollectionResult) String() string {
	return r.Collection.String()
}

func (db *Database) defineCollection(name string, fields map[string]DataType, primaryKey string, autoFields map[string]bool, optional map[string]bool) (OperationResult, error) {
	for field := range optional {
		if _, ok := fields[field]; !ok {
			return nil, fmt.Errorf("optional field %q not found in schema for collection %q", field, name)
		}
		if field == primaryKey {
			return nil, fmt.Errorf("@id field %q cannot be @optional", field)
		}
	}

	if optional == nil {
		optional = map[string]bool{}
	}

	if primaryKey != "" {
		if _, ok := fields[primaryKey]; !ok {
			return nil, fmt.Errorf("primary key %q not found in schema for collection %q", primaryKey, name)
		}
	}

	autoCounters := map[string]int64{}

	for field := range autoFields {
		dataType, ok := fields[field]
		if !ok {
			return nil, fmt.Errorf("auto-increment field %q not found in schema for collection %q", field, name)
		}
		if dataType != TypeInt {
			return nil, fmt.Errorf("@auto requires an int field (field %q)", field)
		}
		if optional[field] {
			return nil, fmt.Errorf("@auto field %q cannot be @optional", field)
		}
		autoCounters[field] = 1
	}

	collection := &Collection{
		name:         name,
		data:         fields,
		records:      make(map[uint64]Entity),
		primaryKey:   primaryKey,
		primaryIndex: make(map[any]uint64),
		autoCounters: autoCounters,
		optional:     optional,
	}

	db.collections[name] = collection

	return DefineCollectionResult{Collection: *collection}, nil
}
