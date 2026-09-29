package core

import (
	"errors"
	"testing"
)

func TestParseDelete_ParsesFilterOnly(t *testing.T) {
	o, err := ParseDelete(`!> user(id: 1);`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Collection != "user" {
		t.Fatalf("expected collection %q, got %q", "user", o.Collection)
	}

	if o.Filter["id"] != int64(1) {
		t.Fatalf("expected filter id 1, got %v", o.Filter["id"])
	}

	if o.Projection != nil {
		t.Fatalf("expected a nil projection (no '=>' given), got %v", o.Projection)
	}
}

func TestParseDelete_AllowsExplicitEmptyFilter(t *testing.T) {
	o, err := ParseDelete(`!> user();`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Filter) != 0 {
		t.Fatalf("expected no filter conditions, got %d", len(o.Filter))
	}
}

func TestParseDelete_ParsesFilterAndProjection(t *testing.T) {
	o, err := ParseDelete(`!> user(id: 1) => {id, name};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Projection) != 2 || o.Projection[0] != "id" || o.Projection[1] != "name" {
		t.Fatalf("expected projection [id name], got %v", o.Projection)
	}
}

func TestParseDelete_EmptyProjectionIsDistinctFromNoProjection(t *testing.T) {
	o, err := ParseDelete(`!> user() => {};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Projection == nil {
		t.Fatal("expected a non-nil (empty) projection since '=>' was used")
	}

	if len(o.Projection) != 0 {
		t.Fatalf("expected an empty projection, got %v", o.Projection)
	}
}

func TestParseDelete_ParsesWildcardProjection(t *testing.T) {
	o, err := ParseDelete(`!> user(id: 1) => {*};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Projection) != 1 || o.Projection[0] != "*" {
		t.Fatalf("expected projection [*], got %v", o.Projection)
	}
}

func TestParseDelete_RejectsWildcardMixedWithNamedFields(t *testing.T) {
	if _, err := ParseDelete(`!> user(id: 1) => {*, id};`); err == nil {
		t.Fatal("expected an error for a wildcard combined with named fields")
	}

	if _, err := ParseDelete(`!> user(id: 1) => {id, *};`); err == nil {
		t.Fatal("expected an error for named fields combined with a wildcard")
	}
}

func TestParseDelete_RejectsMissingFilterParens(t *testing.T) {
	if _, err := ParseDelete(`!> user;`); err == nil {
		t.Fatal("expected an error for a delete with no filter parens at all")
	}
}

func TestParseDelete_RejectsArrowInPlaceOfFilterParens(t *testing.T) {
	if _, err := ParseDelete(`!> user => {id};`); err == nil {
		t.Fatal("expected an error for skipping straight to '=>' without filter parens")
	}
}

func TestParseDelete_BareCollectionNameIsIncomplete(t *testing.T) {
	if _, err := ParseDelete(`!> user`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a collection name with nothing after it yet, got %v", err)
	}
}

func TestParseDelete_RejectsIncompleteDeleteOperator(t *testing.T) {
	if _, err := ParseDelete(`!`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a lone '!', got %v", err)
	}
}

func TestParse_DispatchesToDelete(t *testing.T) {
	o, err := Parse(`!> user(id: 1);`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.(DeleteOperation); !ok {
		t.Fatalf("expected a DeleteOperation, got %T", o)
	}
}
