package core

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

func (db *Database) Run(o Operation) (OperationResult, error) {
	switch op := o.(type) {
	case DefineCollectionOperation:
		return db.defineCollection(op)

	case InsertOperation:
		return db.insert(op.Collection, op.Records, op.Projection)

	case ReadOperation:
		return db.read(op.Collection, op.Filter, op.Projection)

	case DeleteOperation:
		return db.delete(op.Collection, op.Filter, op.Projection)

	case MergeOperation:
		return db.merge(op.Collection, op.Filter, op.Payload, op.Projection)

	default:
		return nil, errors.New("invalid operation")
	}
}
