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

	got := runExpectingError(t, d, `>> user {id: 1 age: "old"};`)

	want := `field "age": expected int, got text`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_IntegerIntoFloatFieldIsRejected(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `item { id: int @id price: float }`)

	got := runExpectingError(t, d, `>> item {id: 1 price: 10};`)

	want := `field "price": expected float, got int`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_DecimalIntoFloatFieldIsAccepted(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `item { id: int @id price: float }`, `>> item {id: 1 price: 10.0};`)

	expectCount(t, d, `<< item(price: 10.0);`, "1")
}

func TestSchema_FilterRejectsWrongType(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `item { id: int @id price: float }`, `>> item {id: 1 price: 10.0};`)

	got := runExpectingError(t, d, `<< item(price: 10) => {id}`)

	want := `field "price": expected float, got int`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_MergePayloadRejectsWrongType(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id active: bool }`, `>> user {id: 1 active: false};`)

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
		`>> user {id: 1 active: true};`,
		`>> user {id: 2 active: false};`,
	)

	result := runStatements(t, d, `<< user(active: true) => {id}`).(ReadResult)

	if len(result.Records) != 1 || result.Records[0]["id"] != int64(1) {
		t.Fatalf("expected only record 1, got %v", result.Records)
	}
}

func TestSchema_InsertRejectsUndeclaredField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`)

	got := runExpectingError(t, d, `>> user {id: 1 nmae: "Matt"};`)

	// The typo is reported alongside the required field it left missing.
	want := "2 problems, nothing inserted:\n" +
		"  field \"nmae\" not found in schema for collection \"user\"\n" +
		"  field \"name\" is required for collection \"user\""
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_MergePayloadRejectsUndeclaredField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`, `>> user {id: 1 name: "Matt"};`)

	got := runExpectingError(t, d, `~> user() {nickname: "M"};`)

	want := `field "nickname" not found in schema for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_OneBadFieldFailsTheWholeInsert(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text age: int }`)

	runExpectingError(t, d, `>> user {id: 1 name: "Matt" age: "x"};`)

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_BadInsertDoesNotConsumeAutoIncrementValue(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text }`)

	runExpectingError(t, d, `>> user {name: 5};`)

	result := runStatements(t, d, `>> user {name: "Matt"} => {*};`).(InsertResult)
	if result.Records[0]["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v", result.Records[0]["id"])
	}
}

func TestSchema_BulkMergeNeverHalfApplies(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text age: int }`,
		`>> user {id: 1 name: "Matt" age: 39};`,
		`>> user {id: 2 name: "Pat" age: 40};`,
	)

	runExpectingError(t, d, `~> user() {name: "Sam" age: "x"};`)

	expectCount(t, d, `<< user(name: "Sam");`, "0")
	expectCount(t, d, `<< user(name: "Matt");`, "1")
	expectCount(t, d, `<< user(name: "Pat");`, "1")
}

func TestSchema_InvalidDeleteFilterDeletesNothing(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id }`, `>> user {id: 1};`)

	got := runExpectingError(t, d, `!> user(id: "1");`)

	if !strings.Contains(got, "expected int, got text") {
		t.Fatalf("expected a type error, got %q", got)
	}

	expectCount(t, d, `<< user;`, "1")
}

func TestSchema_NullIsNotATypeMismatchOnOptionalField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id nickname: text @optional }`, `>> user {id: 1 nickname: null};`)

	record := d.collections["user"].records[0]
	if _, present := record["nickname"]; present {
		t.Fatalf("expected nickname to be stored as absent, got %v", record)
	}
}

func TestSchema_InsertRejectsMissingRequiredField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`)

	got := runExpectingError(t, d, `>> user {id: 1};`)

	want := `field "name" is required for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_InsertRejectsNullOnRequiredField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`)

	got := runExpectingError(t, d, `>> user {id: 1 name: null};`)

	want := `field "name" is required and cannot be null`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestSchema_InsertRejectsNullPrimaryKey(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id }`)

	got := runExpectingError(t, d, `>> user {id: null};`)

	want := `record missing primary key "id" for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_InsertRejectsNullOnAutoPrimaryKey(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto }`)

	got := runExpectingError(t, d, `>> user {id: null};`)

	want := `field "id" is auto-increment and must not be supplied for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_InsertAllowsOmittedOptionalField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text nickname: text @optional }`,
		`>> user {id: 1 name: "Matt"};`,
	)

	expectCount(t, d, `<< user;`, "1")
}

func TestSchema_AutoPrimaryKeyIsNotRequiredOnInsert(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text }`)

	result := runStatements(t, d, `>> user {name: "Matt"} => {*};`).(InsertResult)
	if result.Records[0]["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v", result.Records[0]["id"])
	}
}

func TestSchema_MissingRequiredFieldDoesNotConsumeAutoIncrementValue(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text }`)

	runExpectingError(t, d, `>> user {};`)

	result := runStatements(t, d, `>> user {name: "Matt"} => {*};`).(InsertResult)
	if result.Records[0]["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v", result.Records[0]["id"])
	}
}

func TestSchema_MergeDoesNotRequireUnnamedFields(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text age: int }`,
		`>> user {id: 1 name: "Matt" age: 39};`,
		`~> user(id: 1) {age: 40};`,
	)

	expectCount(t, d, `<< user(name: "Matt" age: 40);`, "1")
}

func TestSchema_MergeRejectsNullOnRequiredField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`, `>> user {id: 1 name: "Matt"};`)

	got := runExpectingError(t, d, `~> user() {name: null};`)

	want := `field "name" is required and cannot be null`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user(name: "Matt");`, "1")
}

func TestSchema_MergeRejectsNullPrimaryKeyWithPrimaryKeyError(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`, `>> user {id: 1 name: "Matt"};`)

	got := runExpectingError(t, d, `~> user() {id: null};`)

	want := `payload must not set primary key "id" for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_MergeNullClearsOptionalField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id nickname: text @optional }`,
		`>> user {id: 1 nickname: "M"};`,
		`~> user(id: 1) {nickname: null};`,
	)

	expectCount(t, d, `<< user(nickname: null);`, "1")

	record := d.collections["user"].records[0]
	if _, present := record["nickname"]; present {
		t.Fatalf("expected nickname to be removed from the record, got %v", record)
	}
}

func TestSchema_OmittedAndNullAreTheSame(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id nickname: text @optional }`,
		`>> user {id: 1};`,
		`>> user {id: 2 nickname: null};`,
	)

	expectCount(t, d, `<< user(nickname: null);`, "2")
}

func TestSchema_NullFilterMatchesRecordsWithoutAValue(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id nickname: text @optional }`,
		`>> user {id: 1};`,
		`>> user {id: 2 nickname: "M"};`,
	)

	result := runStatements(t, d, `<< user(nickname: null) => {id}`).(ReadResult)
	if len(result.Records) != 1 || result.Records[0]["id"] != int64(1) {
		t.Fatalf("expected only record 1, got %v", result.Records)
	}
}

func TestSchema_NullFilterOnRequiredFieldIsRejected(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text }`)

	got := runExpectingError(t, d, `<< user(name: null);`)

	want := `field "name" is required and cannot be null`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestSchema_NullFilterWorksForDelete(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id nickname: text @optional }`,
		`>> user {id: 1};`,
		`>> user {id: 2 nickname: "M"};`,
		`!> user(nickname: null);`,
	)

	expectCount(t, d, `<< user;`, "1")
	expectCount(t, d, `<< user(id: 2);`, "1")
}

func TestSchema_ProjectionShowsNullForMissingValue(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id nickname: text @optional }`, `>> user {id: 1};`)

	got := runStatements(t, d, `<< user => {id, nickname};`).(ReadResult).String()

	want := "id  nickname\n1   null"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
