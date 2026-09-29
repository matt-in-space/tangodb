package core

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseInsert_ParsesFlatValues(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1, name: "Matt", age: 39}`)
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
	o, err := ParseInsert(`>> product {price: 9.99}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Record["price"] != 9.99 {
		t.Fatalf("expected price 9.99, got %v (%T)", o.Record["price"], o.Record["price"])
	}
}

func TestParseInsert_AllowsTrailingComma(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1,}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Record) != 1 {
		t.Fatalf("expected 1 field, got %d", len(o.Record))
	}
}

func TestParseInsert_AllowsEmptyRecord(t *testing.T) {
	o, err := ParseInsert(`>> user {}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Record) != 0 {
		t.Fatalf("expected no fields, got %d", len(o.Record))
	}
}

func TestParseInsert_UnescapesStrings(t *testing.T) {
	o, err := ParseInsert(`>> user {name: "say \"hi\" \\ bye"}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	want := `say "hi" \ bye`
	if o.Record["name"] != want {
		t.Fatalf("expected %q, got %q", want, o.Record["name"])
	}
}

func TestParseInsert_ToleratesTrailingSemicolon(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Record["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v", o.Record["id"])
	}
}

func TestParseInsert_CommasAreOptional(t *testing.T) {
	withoutCommas, err := ParseInsert(`>> user {id: 1 name: "Matt"}`)
	if err != nil {
		t.Fatalf("Failed to parse without commas, err: %v", err)
	}

	withCommas, err := ParseInsert(`>> user {id: 1, name: "Matt"}`)
	if err != nil {
		t.Fatalf("Failed to parse with commas, err: %v", err)
	}

	if !reflect.DeepEqual(withoutCommas, withCommas) {
		t.Fatalf("expected identical operations, got %v and %v", withoutCommas, withCommas)
	}
}

func TestParseInsert_AllowsMixedAndTrailingCommas(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1, name: "Matt" age: 39,}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Record) != 3 || o.Record["age"] != int64(39) {
		t.Fatalf("expected all three fields, got %v", o.Record)
	}
}

func TestParseInsert_CommaInsideStringIsPartOfTheValue(t *testing.T) {
	o, err := ParseInsert(`>> user {name: "Smith, Matt"}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Record["name"] != "Smith, Matt" {
		t.Fatalf("expected %q, got %q", "Smith, Matt", o.Record["name"])
	}
}

func TestParseInsert_RejectsUnterminatedString(t *testing.T) {
	if _, err := ParseInsert(`>> user {name: "Matt`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for an unterminated string, got %v", err)
	}
}

func TestParseInsert_RejectsIncompleteInsertOperator(t *testing.T) {
	if _, err := ParseInsert(`>`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a lone '>', got %v", err)
	}
}

func TestParseInsert_RejectsIncompleteWithNoRecordLiteral(t *testing.T) {
	if _, err := ParseInsert(`>> user`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a collection name with no record literal yet, got %v", err)
	}
}

func TestParse_DispatchesToInsert(t *testing.T) {
	o, err := Parse(`>> user {id: 1}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.(InsertOperation); !ok {
		t.Fatalf("expected an InsertOperation, got %T", o)
	}
}
