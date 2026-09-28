package main

import (
	"errors"
	"testing"
)

func TestParseInsert_ParsesFlatValues(t *testing.T) {
	o, err := ParseInsert(`>> user => {id: 1, name: "Matt", age: 39}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Collection != "user" {
		t.Fatalf("expected collection %q, got %q", "user", o.Collection)
	}

	if o.Record["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v (%T)", o.Record["id"], o.Record["id"])
	}

	if o.Record["name"] != "Matt" {
		t.Fatalf("expected name %q, got %v", "Matt", o.Record["name"])
	}

	if o.Record["age"] != int64(39) {
		t.Fatalf("expected age 39, got %v (%T)", o.Record["age"], o.Record["age"])
	}
}

func TestParseInsert_ParsesFloats(t *testing.T) {
	o, err := ParseInsert(`>> product => {price: 9.99}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Record["price"] != 9.99 {
		t.Fatalf("expected price 9.99, got %v (%T)", o.Record["price"], o.Record["price"])
	}
}

func TestParseInsert_AllowsTrailingComma(t *testing.T) {
	o, err := ParseInsert(`>> user => {id: 1,}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Record) != 1 {
		t.Fatalf("expected 1 field, got %d", len(o.Record))
	}
}

func TestParseInsert_AllowsEmptyRecord(t *testing.T) {
	o, err := ParseInsert(`>> user => {}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Record) != 0 {
		t.Fatalf("expected no fields, got %d", len(o.Record))
	}
}

func TestParseInsert_UnescapesStrings(t *testing.T) {
	o, err := ParseInsert(`>> user => {name: "say \"hi\" \\ bye"}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	want := `say "hi" \ bye`
	if o.Record["name"] != want {
		t.Fatalf("expected %q, got %q", want, o.Record["name"])
	}
}

func TestParseInsert_RejectsMissingComma(t *testing.T) {
	if _, err := ParseInsert(`>> user => {id: 1 name: "Matt"}`); err == nil {
		t.Fatal("expected an error for a missing comma between fields")
	}
}

func TestParseInsert_RejectsUnterminatedString(t *testing.T) {
	if _, err := ParseInsert(`>> user => {name: "Matt`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for an unterminated string, got %v", err)
	}
}

func TestParseInsert_RejectsIncompleteInsertOperator(t *testing.T) {
	if _, err := ParseInsert(`>`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a lone '>', got %v", err)
	}
}

func TestParseInsert_RejectsIncompleteArrow(t *testing.T) {
	if _, err := ParseInsert(`>> user =`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a lone '=', got %v", err)
	}
}

func TestParse_DispatchesToInsert(t *testing.T) {
	o, err := Parse(`>> user => {id: 1}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.(InsertOperation); !ok {
		t.Fatalf("expected an InsertOperation, got %T", o)
	}
}
