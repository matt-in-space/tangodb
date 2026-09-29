package core

import (
	"errors"
	"testing"
)

func TestParseRead_ParsesFilterAndProjection(t *testing.T) {
	o, err := ParseRead(`<< user(id: 1) => {id, name}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Collection != "user" {
		t.Fatalf("expected collection %q, got %q", "user", o.Collection)
	}

	if o.Filter["id"] != int64(1) {
		t.Fatalf("expected filter id 1, got %v", o.Filter["id"])
	}

	if len(o.Projection) != 2 || o.Projection[0] != "id" || o.Projection[1] != "name" {
		t.Fatalf("expected projection [id name], got %v", o.Projection)
	}
}

func TestParseRead_AllowsMultipleFilterConditions(t *testing.T) {
	o, err := ParseRead(`<< user(id: 1, name: "Matt") => {id}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Filter) != 2 {
		t.Fatalf("expected 2 filter conditions, got %d", len(o.Filter))
	}
}

func TestParseRead_AllowsEmptyFilter(t *testing.T) {
	o, err := ParseRead(`<< user() => {id}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Filter) != 0 {
		t.Fatalf("expected no filter conditions, got %d", len(o.Filter))
	}
}

func TestParseRead_RejectsMissingProjection(t *testing.T) {
	if _, err := ParseRead(`<< user(id: 1)`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a missing projection, got %v", err)
	}
}

func TestParseRead_AllowsOmittingFilterParensWhenArrowFollows(t *testing.T) {
	o, err := ParseRead(`<< user => {id, name}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Filter) != 0 {
		t.Fatalf("expected no filter conditions, got %d", len(o.Filter))
	}

	if len(o.Projection) != 2 {
		t.Fatalf("expected 2 projected fields, got %d", len(o.Projection))
	}
}

func TestParseRead_SemicolonTerminatesWithNoFilterOrProjection(t *testing.T) {
	o, err := ParseRead(`<< user;`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Collection != "user" {
		t.Fatalf("expected collection %q, got %q", "user", o.Collection)
	}

	if len(o.Filter) != 0 {
		t.Fatalf("expected no filter conditions, got %d", len(o.Filter))
	}

	if o.Projection != nil {
		t.Fatalf("expected a nil projection (meaning \"count only\"), got %v", o.Projection)
	}
}

func TestParseRead_SemicolonTerminatesAfterFilterWithNoProjection(t *testing.T) {
	o, err := ParseRead(`<< user(id: 1);`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Filter["id"] != int64(1) {
		t.Fatalf("expected filter id 1, got %v", o.Filter["id"])
	}

	if o.Projection != nil {
		t.Fatalf("expected a nil projection, got %v", o.Projection)
	}
}

func TestParseRead_SemicolonToleratedOnAFullyExplicitStatement(t *testing.T) {
	o, err := ParseRead(`<< user(id: 1) => {id};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Projection) != 1 || o.Projection[0] != "id" {
		t.Fatalf("expected projection [id], got %v", o.Projection)
	}
}

func TestParseRead_ParsesWildcardProjection(t *testing.T) {
	o, err := ParseRead(`<< user => {*};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Projection) != 1 || o.Projection[0] != "*" {
		t.Fatalf("expected projection [*], got %v", o.Projection)
	}
}

func TestParseRead_RejectsWildcardMixedWithNamedFieldsAfter(t *testing.T) {
	if _, err := ParseRead(`<< user => {*, id};`); err == nil {
		t.Fatal("expected an error for a wildcard combined with named fields")
	}
}

func TestParseRead_RejectsWildcardMixedWithNamedFieldsBefore(t *testing.T) {
	if _, err := ParseRead(`<< user => {id, *};`); err == nil {
		t.Fatal("expected an error for named fields combined with a wildcard")
	}
}

func TestParseRead_BareCollectionNameWithoutSemicolonIsIncomplete(t *testing.T) {
	if _, err := ParseRead(`<< user`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a bare collection name with no ';', got %v", err)
	}
}

func TestParse_DispatchesToRead(t *testing.T) {
	o, err := Parse(`<< user() => {id}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.(ReadOperation); !ok {
		t.Fatalf("expected a ReadOperation, got %T", o)
	}
}
