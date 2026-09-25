package main

import "testing"

func TestParse_DispatchesToDefineCollection(t *testing.T) {
	o, err := Parse(`user { id: int @id }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	defineOp, ok := o.(DefineCollectionOperation)
	if !ok {
		t.Fatalf("expected a DefineCollectionOperation, got %T", o)
	}

	if defineOp.Name != "user" {
		t.Fatalf("expected name %q, got %q", "user", defineOp.Name)
	}
}

func TestParse_RejectsUnrecognizedStatement(t *testing.T) {
	if _, err := Parse(`: nonsense`); err == nil {
		t.Fatal("expected an error for an unrecognized statement")
	}
}

func TestParse_RejectsEmptyInput(t *testing.T) {
	if _, err := Parse(``); err == nil {
		t.Fatal("expected an error for empty input")
	}
}
