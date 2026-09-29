package main

import "errors"

type Operation interface{}

type OperationResult interface{}

type Database struct {
	path        string
	collections map[string]*Collection
}

func NewDatabase(path string) *Database {
	return &Database{
		path:        path,
		collections: make(map[string]*Collection),
	}
}

func (db *Database) run(o Operation) (OperationResult, error) {
	switch op := o.(type) {
	case DefineCollectionOperation:
		return db.defineCollection(op.Name, op.Data, op.PrimaryKey, op.AutoIncrement)

	case InsertOperation:
		return db.insert(op.Collection, op.Record)

	case ReadOperation:
		return db.read(op.Collection, op.Filter, op.Projection)

	case DeleteOperation:
		return db.delete(op.Collection, op.Filter, op.Projection)

	default:
		return nil, errors.New("invalid operation")
	}
}
