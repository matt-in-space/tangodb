package core

import "fmt"

type DefineCollectionOperation struct {
	Name       string
	Data       map[string]DataType
	PrimaryKey string
	AutoFields map[string]bool
	Optional   map[string]bool
	Objects    map[string]*Schema
}

type DefineCollectionResult struct {
	Collection Collection
}

func (r DefineCollectionResult) String() string {
	return r.Collection.String()
}

func (db *Database) defineCollection(op DefineCollectionOperation) (OperationResult, error) {
	name, fields, primaryKey, autoFields := op.Name, op.Data, op.PrimaryKey, op.AutoFields

	optional := op.Optional
	if optional == nil {
		optional = map[string]bool{}
	}

	objects := op.Objects
	if objects == nil {
		objects = map[string]*Schema{}
	}

	root := &Schema{Data: fields, Optional: optional, Objects: objects}
	if err := checkSchema(name, root, ""); err != nil {
		return nil, err
	}

	if primaryKey != "" {
		dataType, ok := fields[primaryKey]
		if !ok {
			return nil, fmt.Errorf("primary key %q not found in schema for collection %q", primaryKey, name)
		}
		if dataType == TypeObject {
			return nil, fmt.Errorf("only @optional is allowed on an embedded block (field %q)", primaryKey)
		}
		if optional[primaryKey] {
			return nil, fmt.Errorf("@id field %q cannot be @optional", primaryKey)
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
		objects:      objects,
	}

	db.collections[name] = collection

	return DefineCollectionResult{Collection: *collection}, nil
}

// checkSchema checks that a block's optional and object entries match its
// fields, recursing into embedded blocks, for operations built directly in Go
// rather than parsed. @id and @auto can't appear inside a block at all, since
// a Schema has nowhere to record them.
func checkSchema(collectionName string, schema *Schema, path string) error {
	for field := range schema.Optional {
		if _, ok := schema.Data[field]; !ok {
			return fmt.Errorf("optional field %q not found in schema for collection %q", path+field, collectionName)
		}
	}

	for field, dataType := range schema.Data {
		nested, hasShape := schema.Objects[field]

		if dataType == TypeObject && !hasShape {
			return fmt.Errorf("embedded block field %q has no fields declared for collection %q", path+field, collectionName)
		}
		if dataType != TypeObject && hasShape {
			return fmt.Errorf("field %q has a block shape but is not an object for collection %q", path+field, collectionName)
		}

		if hasShape {
			if nested.Optional == nil {
				nested.Optional = map[string]bool{}
			}
			if nested.Objects == nil {
				nested.Objects = map[string]*Schema{}
			}
			if err := checkSchema(collectionName, nested, path+field+"."); err != nil {
				return err
			}
		}
	}

	return nil
}
