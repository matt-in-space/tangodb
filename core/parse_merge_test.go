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

func TestParseMerge_ParsesWildcardProjection(t *testing.T) {
	o, err := ParseMerge(`~> user(id: 1) {name: "Matt"} => {*};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Projection) != 1 || o.Projection[0] != "*" {
		t.Fatalf("expected projection [*], got %v", o.Projection)
	}
}

func TestParseMerge_RejectsWildcardMixedWithNamedFields(t *testing.T) {
	if _, err := ParseMerge(`~> user(id: 1) {name: "Matt"} => {*, id};`); err == nil {
		t.Fatal("expected an error for a wildcard combined with named fields")
	}

	if _, err := ParseMerge(`~> user(id: 1) {name: "Matt"} => {id, *};`); err == nil {
		t.Fatal("expected an error for named fields combined with a wildcard")
	}
}

func TestParseMerge_RejectsMissingFilterParens(t *testing.T) {
	if _, err := ParseMerge(`~> user {name: "Matt"};`); err == nil {
		t.Fatal("expected an error for a merge with no filter parens at all")
	}
}

func TestParseMerge_MissingPayloadIsIncomplete(t *testing.T) {
	if _, err := ParseMerge(`~> user(id: 1)`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a merge with no payload or ';' yet, got %v", err)
	}
}

func TestParseMerge_RejectsArrowInPlaceOfPayload(t *testing.T) {
	if _, err := ParseMerge(`~> user(id: 1) => {id};`); err == nil {
		t.Fatal("expected an error for skipping straight to '=>' without a payload")
	}
}

func TestParseMerge_BareCollectionNameIsIncomplete(t *testing.T) {
	if _, err := ParseMerge(`~> user`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a collection name with no filter or ';' yet, got %v", err)
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

func TestParseMerge_ParsesBooleansInFilterAndPayload(t *testing.T) {
	o, err := ParseMerge(`~> user(active: true) {active: false};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Filter["active"] != true {
		t.Fatalf("expected filter active true, got %v (%T)", o.Filter["active"], o.Filter["active"])
	}

	if o.Payload["active"] != false {
		t.Fatalf("expected payload active false, got %v (%T)", o.Payload["active"], o.Payload["active"])
	}
}

func TestParseMerge_ParsesNullInFilterAndPayload(t *testing.T) {
	o, err := ParseMerge(`~> user(nickname: null) {nickname: null};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if value, ok := o.Filter["nickname"]; !ok || value != nil {
		t.Fatalf("expected filter nickname nil, got %v (present: %v)", value, ok)
	}

	if value, ok := o.Payload["nickname"]; !ok || value != nil {
		t.Fatalf("expected payload nickname nil, got %v (present: %v)", value, ok)
	}
}
