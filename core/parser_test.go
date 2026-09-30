package core

import (
	"errors"
	"testing"
)

func TestParse_DispatchesToDefineCollection(t *testing.T) {
	o, err := Parse(`user { id: int @id };`)
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
	if _, err := Parse(`: nonsense;`); err == nil || errors.Is(err, ErrIncompleteInput) {
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

	if o.(InsertOperation).Records[0]["name"] != "{" {
		t.Fatalf("expected name %q, got %v", "{", o.(InsertOperation).Records[0]["name"])
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

func TestLex_DottedPathIsOneIdentifier(t *testing.T) {
	tokens, err := lex(`address.geo.lat 9.99`)
	if err != nil {
		t.Fatalf("Failed to lex, err: %v", err)
	}

	if tokens[0].kind != tokenIdent || tokens[0].value != "address.geo.lat" {
		t.Fatalf("expected the identifier address.geo.lat, got %+v", tokens[0])
	}

	if tokens[1].kind != tokenNumber || tokens[1].value != "9.99" {
		t.Fatalf("expected the number 9.99, got %+v", tokens[1])
	}
}

func TestLex_TrailingDotIsNotPartOfAName(t *testing.T) {
	if _, err := lex(`address.`); err == nil {
		t.Fatal("expected an error for a trailing dot")
	}
}

func TestParse_RejectsDottedNamesWhereNamesAreDeclaredOrWritten(t *testing.T) {
	cases := map[string]string{
		`user { address.city: text };`:   `field name "address.city" cannot contain "."`,
		`user.x { id: int };`:            `collection name "user.x" cannot contain "."`,
		`>> user {address.city: "MSP"};`: `field name "address.city" cannot contain "."`,
		`<< user.address;`:               `collection name "user.address" cannot contain "."`,
	}

	for input, want := range cases {
		_, err := Parse(input)
		if err == nil || err.Error() != want {
			t.Fatalf("for %q expected error %q, got %v", input, want, err)
		}
	}
}

func TestParse_EveryStatementIsIncompleteWithoutSemicolon(t *testing.T) {
	for _, input := range []string{
		`user { id: int @id }`,
		`>> user {id: 1}`,
		`>> user {id: 1} => {id}`,
		`<< user => {*}`,
		`<< user(id: 1) => {id}`,
		`!> user(id: 1) => {id}`,
		`~> user(id: 1) {name: "Matt"} => {id}`,
		">> user {id: 1}\n\n",
	} {
		if _, err := Parse(input); !errors.Is(err, ErrIncompleteInput) {
			t.Fatalf("expected %q to be incomplete, got %v", input, err)
		}
	}
}

func TestParse_SemicolonCompletesEveryStatement(t *testing.T) {
	for _, input := range []string{
		`user { id: int @id };`,
		`>> user {id: 1};`,
		`<< user => {*};`,
		`!> user(id: 1);`,
		`~> user(id: 1) {name: "Matt"};`,
		">> user {id: 1}\n\n;",
	} {
		if _, err := Parse(input); err != nil {
			t.Fatalf("expected %q to parse, got %v", input, err)
		}
	}
}

func TestParse_SemicolonInsideStringOrBracketsDoesNotCount(t *testing.T) {
	for _, input := range []string{`>> user {name: "a;b"}`, `>> user {id: 1;`, `<< user(id: 1; name: "x"`} {
		if _, err := Parse(input); !errors.Is(err, ErrIncompleteInput) {
			t.Fatalf("expected %q to be incomplete, got %v", input, err)
		}
	}
}

func TestParse_MistakeWaitsForTheSemicolon(t *testing.T) {
	if _, err := Parse(`>> user {id: 1} nonsense`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected incomplete input before the ';', got %v", err)
	}

	_, err := Parse(`>> user {id: 1} nonsense;`)
	if err == nil || errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected a parse error once the ';' arrives, got %v", err)
	}
}

func TestParse_OneStatementAtATime(t *testing.T) {
	_, err := Parse(`>> user {id: 1}; << user;`)

	want := `unexpected input after statement: "<<"`
	if err == nil || err.Error() != want {
		t.Fatalf("expected error %q, got %v", want, err)
	}
}

func TestParse_DirectParsersRequireSemicolonToo(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int }`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ParseDefineCollection to require ';', got %v", err)
	}

	if _, err := ParseDelete(`!> user(id: 1)`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ParseDelete to require ';', got %v", err)
	}
}
