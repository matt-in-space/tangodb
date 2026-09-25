package main

import "testing"

func TestDatabaseRun_ReturnsAResult(t *testing.T) {
	db := &Database{}
	db.run(Operation{})
}
