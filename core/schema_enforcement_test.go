package core

import (
	"strings"
	"testing"
)

// runStatements parses and runs each statement in order, failing the test on
// any error, and returns the last result.
func runStatements(t *testing.T, d *Database, statements ...string) OperationResult {
	t.Helper()

	var result OperationResult

	for _, statement := range statements {
		op, err := Parse(statement)
		if err != nil {
			t.Fatalf("Failed to parse %q, err: %v", statement, err)
		}

		result, err = d.Run(op)
		if err != nil {
			t.Fatalf("Failed to run %q, err: %v", statement, err)
		}
	}

	return result
}

// runExpectingError parses and runs a statement that should fail validation,
// and returns the error message.
func runExpectingError(t *testing.T, d *Database, statement string) string {
	t.Helper()

	op, err := Parse(statement)
	if err != nil {
		t.Fatalf("Failed to parse %q, err: %v", statement, err)
	}

	if _, err := d.Run(op); err != nil {
		return err.Error()
	}

	t.Fatalf("expected %q to fail, but it succeeded", statement)
	return ""
}

func expectCount(t *testing.T, d *Database, statement string, want string) {
	t.Helper()

	if got := runStatements(t, d, statement).(ReadResult).String(); got != want {
		t.Fatalf("expected %q to return %s, got %s", statement, want, got)
	}
}

func TestSchema_InsertRejectsWrongType(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id age: int }`)

	got := runExpectingError(t, d, `>> user {id: 1 age: "old"}`)

	want := `field "age": expected int, got text`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_IntegerIntoFloatFieldIsRejected(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `item { id: int @id price: float }`)

	got := runExpectingError(t, d, `>> item {id: 1 price: 10}`)

	want := `field "price": expected float, got int`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_DecimalIntoFloatFieldIsAccepted(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `item { id: int @id price: float }`, `>> item {id: 1 price: 10.0}`)

	expectCount(t, d, `<< item(price: 10.0);`, "1")
}

func TestSchema_FilterRejectsWrongType(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `item { id: int @id price: float }`, `>> item {id: 1 price: 10.0}`)

	got := runExpectingError(t, d, `<< item(price: 10) => {id}`)

	want := `field "price": expected float, got int`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_MergePayloadRejectsWrongType(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id active: bool }`, `>> user {id: 1 active: false}`)

	got := runExpectingError(t, d, `~> user(id: 1) {active: "yes"};`)

	want := `field "active": expected bool, got text`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user(active: false);`, "1")
}

func TestSchema_BoolFieldRoundTrip(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id active: bool }`,
		`>> user {id: 1 active: true}`,
		`>> user {id: 2 active: false}`,
	)

	result := runStatements(t, d, `<< user(active: true) => {id}`).(ReadResult)

	if len(result.Records) != 1 || result.Records[0]["id"] != int64(1) {
		t.Fatalf("expected only record 1, got %v", result.Records)
	}
}

func TestSchema_InsertRejectsUndeclaredField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`)

	got := runExpectingError(t, d, `>> user {id: 1 nmae: "Matt"}`)

	want := `field "nmae" not found in schema for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_MergePayloadRejectsUndeclaredField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`, `>> user {id: 1 name: "Matt"}`)

	got := runExpectingError(t, d, `~> user() {nickname: "M"};`)

	want := `field "nickname" not found in schema for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_OneBadFieldFailsTheWholeInsert(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text age: int }`)

	runExpectingError(t, d, `>> user {id: 1 name: "Matt" age: "x"}`)

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_BadInsertDoesNotConsumeAutoIncrementValue(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text }`)

	runExpectingError(t, d, `>> user {name: 5}`)

	result := runStatements(t, d, `>> user {name: "Matt"}`).(InsertResult)
	if result.Record["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v", result.Record["id"])
	}
}

func TestSchema_BulkMergeNeverHalfApplies(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text age: int }`,
		`>> user {id: 1 name: "Matt" age: 39}`,
		`>> user {id: 2 name: "Pat" age: 40}`,
	)

	runExpectingError(t, d, `~> user() {name: "Sam" age: "x"};`)

	expectCount(t, d, `<< user(name: "Sam");`, "0")
	expectCount(t, d, `<< user(name: "Matt");`, "1")
	expectCount(t, d, `<< user(name: "Pat");`, "1")
}

func TestSchema_InvalidDeleteFilterDeletesNothing(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id }`, `>> user {id: 1}`)

	got := runExpectingError(t, d, `!> user(id: "1");`)

	if !strings.Contains(got, "expected int, got text") {
		t.Fatalf("expected a type error, got %q", got)
	}

	expectCount(t, d, `<< user;`, "1")
}
