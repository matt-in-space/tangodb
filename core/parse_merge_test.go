package core

import (
	"errors"
	"testing"
)

func TestParseMerge_ParsesFilterAndPayload(t *testing.T) {
	o, err := ParseMerge(`~> user(id: 1) {name: "Matt"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Collection != "user" {
		t.Fatalf("expected collection %q, got %q", "user", o.Collection)
	}

	if o.Filter["id"] != int64(1) {
		t.Fatalf("expected filter id 1, got %v", o.Filter["id"])
	}

	if o.Payload["name"] != "Matt" {
		t.Fatalf("expected payload name %q, got %v", "Matt", o.Payload["name"])
	}

	if o.Projection != nil {
		t.Fatalf("expected a nil projection (no '=>' given), got %v", o.Projection)
	}
}

func TestParseMerge_AllowsExplicitEmptyFilter(t *testing.T) {
	o, err := ParseMerge(`~> user() {name: "Matt"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Filter) != 0 {
		t.Fatalf("expected no filter conditions, got %d", len(o.Filter))
	}
}

func TestParseMerge_ParsesFilterPayloadAndProjection(t *testing.T) {
	o, err := ParseMerge(`~> user(id: 1) {name: "Matt"} => {id, name};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Projection) != 2 || o.Projection[0] != "id" || o.Projection[1] != "name" {
		t.Fatalf("expected projection [id name], got %v", o.Projection)
	}
}

func TestParseMerge_RejectsMissingFilterParens(t *testing.T) {
	if _, err := ParseMerge(`~> user {name: "Matt"};`); err == nil {
		t.Fatal("expected an error for a merge with no filter parens at all")
	}
}

func TestParseMerge_MissingPayloadIsIncomplete(t *testing.T) {
	if _, err := ParseMerge(`~> user(id: 1)`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a filter with no payload yet, got %v", err)
	}
}

func TestParseMerge_RejectsArrowInPlaceOfPayload(t *testing.T) {
	if _, err := ParseMerge(`~> user(id: 1) => {id}`); err == nil {
		t.Fatal("expected an error for skipping straight to '=>' without a payload")
	}
}

func TestParseMerge_BareCollectionNameIsIncomplete(t *testing.T) {
	if _, err := ParseMerge(`~> user`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a collection name with nothing after it yet, got %v", err)
	}
}

func TestParseMerge_RejectsIncompleteMergeOperator(t *testing.T) {
	if _, err := ParseMerge(`~`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a lone '~', got %v", err)
	}
}

func TestParse_DispatchesToMerge(t *testing.T) {
	o, err := Parse(`~> user(id: 1) {name: "Matt"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.(MergeOperation); !ok {
		t.Fatalf("expected a MergeOperation, got %T", o)
	}
}
