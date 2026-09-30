package core

import (
	"strings"
	"testing"
)

func TestBatch_ReturnsCount(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	got := runStatements(t, d, `>> user {id: 1 name: "Matt"} & {id: 2 name: "Sam"};`).(InsertResult).String()
	if got != "2" {
		t.Fatalf("expected %q, got %q", "2", got)
	}

	expectCount(t, d, `<< user;`, "2")
}

func TestBatch_SpreadOverLines(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	if _, err := Parse(">> user {id: 1 name: \"Matt\"} &\n"); err != ErrIncompleteInput {
		t.Fatalf("expected the first line to be incomplete, got %v", err)
	}

	runStatements(t, d, ">> user {id: 1 name: \"Matt\"} &\n{id: 2 name: \"Sam\"};\n")
	expectCount(t, d, `<< user;`, "2")
}

func TestBatch_RecordsMayDifferInOptionalFields(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text nickname: text @optional };`,
		`>> user {id: 1 name: "Matt" nickname: "M"} & {id: 2 name: "Sam"};`,
	)

	expectCount(t, d, `<< user;`, "2")
}

func TestBatch_OneBadRecordStoresNothing(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	got := runExpectingError(t, d, `>> user {id: 1 name: "Matt"} & {id: 2};`)

	want := `record 2: field "name" is required for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestBatch_OneBadValueFailsTheWholeBatch(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	runExpectingError(t, d, `>> user {id: 1 name: "Matt"} & {id: 2 name: 7};`)

	expectCount(t, d, `<< user;`, "0")
}

func TestBatch_DuplicateKeyWithinBatch(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	got := runExpectingError(t, d, `>> user {id: 1 name: "Matt"} & {id: 1 name: "Sam"};`)

	want := `record 2: duplicate primary key 1 for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestBatch_DuplicateOfStoredKey(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`, `>> user {id: 1 name: "Matt"};`)

	got := runExpectingError(t, d, `>> user {id: 2 name: "Sam"} & {id: 1 name: "Pat"};`)

	want := `record 2: duplicate primary key 1 for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "1")
}

func TestBatch_AutoValuesFollowInputOrder(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text };`)

	got := runStatements(t, d, `>> user {name: "Matt"} & {name: "Sam"} => {id name};`).(InsertResult).String()

	want := "id  name\n1   Matt\n2   Sam"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestBatch_FailedBatchConsumesNoAutoValues(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text };`)

	runExpectingError(t, d, `>> user {name: "Matt"} & {name: 5};`)

	got := runStatements(t, d, `>> user {name: "Sam"} => {id};`).(InsertResult).String()
	if got != "id\n1" {
		t.Fatalf("expected id 1, got:\n%s", got)
	}
}

func TestBatch_ProjectionReturnsEveryRecordInOrder(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	got := runStatements(t, d, `>> user {id: 1 name: "Matt"} & {id: 2 name: "Sam"} => {id};`).(InsertResult).String()

	want := "id\n1\n2"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestBatch_ReportsEveryProblemAcrossRecords(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text age: int @optional };`)

	got := runExpectingError(t, d, `>> user {id: 1 name: "Matt"} & {id: 2} & {id: 1 name: "Sam" age: "x"};`)

	want := "3 problems, nothing inserted:\n" +
		"  record 2: field \"name\" is required for collection \"user\"\n" +
		"  record 3: field \"age\": expected int, got text\n" +
		"  record 3: duplicate primary key 1 for collection \"user\""
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestBatch_SingleInsertReportsEveryProblemWithoutPrefix(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text age: int };`)

	got := runExpectingError(t, d, `>> user {id: 1 name: 5 age: "x"};`)

	want := "2 problems, nothing inserted:\n" +
		"  field \"age\": expected int, got text\n" +
		"  field \"name\": expected text, got int"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}

	if strings.Contains(got, "record ") {
		t.Fatalf("expected no record prefix for a single insert, got %q", got)
	}
}

func TestBatch_OneProblemPerField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto };`)

	got := runExpectingError(t, d, `>> user {id: null};`)

	want := `field "id" is auto-increment and must not be supplied for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}
