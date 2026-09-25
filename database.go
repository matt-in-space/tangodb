package main

import "errors"

type Operation interface{}

type DataType int

const (
	TypeInt DataType = iota
	TypeFloat
	TypeText
	TypeBool
)

type DefineCollectionOperation struct {
	Name string
	Data map[string]DataType
}

type Database struct {
	path        string
	collections map[string]Collection
}

func NewDatabase(path string) *Database {
	return &Database{
		path:        path,
		collections: make(map[string]Collection),
	}
}

type OperationResult interface{}

type DefineCollectionResult struct {
	Collection Collection
}

type Collection struct {
	name    string
	data    map[string]DataType
	records map[any]Entity
}

type Entity struct{}

func (db *Database) run(o Operation) (OperationResult, error) {
	switch op := o.(type) {
	case DefineCollectionOperation:
		return db.defineCollection(op.Name, op.Data)

	default:
		return nil, errors.New("invalid operation")
	}
}

func (db *Database) defineCollection(name string, fields map[string]DataType) (OperationResult, error) {
	collection := Collection{
		name:    name,
		data:    fields,
		records: make(map[any]Entity),
	}

	db.collections[name] = collection

	return DefineCollectionResult{Collection: collection}, nil
}
