package core

import (
	"errors"
	"testing"
)

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

func TestParse_UnclosedBraceIsIncomplete(t *testing.T) {
	// "Matt" is unquoted, which would be a parse error, but the record literal
	// isn't closed yet, so the statement isn't parsed at all.
	if _, err := Parse(">> user {\n  id: 1\n  name: Matt\n"); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput, got %v", err)
	}
}

func TestParse_UnclosedParenIsIncomplete(t *testing.T) {
	if _, err := Parse(`<< user(name: Matt`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput, got %v", err)
	}
}

func TestParse_BraceInsideStringDoesNotCount(t *testing.T) {
	o, err := Parse(`>> user {name: "{"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.(InsertOperation).Record["name"] != "{" {
		t.Fatalf("expected name %q, got %v", "{", o.(InsertOperation).Record["name"])
	}
}

func TestParse_ExtraClosingBracketErrorsImmediately(t *testing.T) {
	for _, input := range []string{`};`, `>> user {id: 1}} {`} {
		_, err := Parse(input)
		if err == nil || errors.Is(err, ErrIncompleteInput) {
			t.Fatalf("expected a parse error for %q, got %v", input, err)
		}
	}
}

func TestParse_BalancedMistakeIsReported(t *testing.T) {
	_, err := Parse(">> user {\n  id: 1\n  name: Matt\n};")
	if err == nil || err.Error() != `expected a value, got "Matt"` {
		t.Fatalf("expected the value error, got %v", err)
	}
}
