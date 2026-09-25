package main

type Database struct{}

type Operation struct{}
type OperationResult struct{}

func (db *Database) run(operation Operation) (OperationResult, error) {
	return OperationResult{}, nil
}
