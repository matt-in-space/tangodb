package core

import "testing"

func TestAuto_SequentialValuesOnNonIDField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `ticket { code: text @id number: int @auto };`)

	first := runStatements(t, d, `>> ticket {code: "A"} => {*};`).(InsertResult)
	second := runStatements(t, d, `>> ticket {code: "B"} => {*};`).(InsertResult)

	if first.Records[0]["number"] != int64(1) || second.Records[0]["number"] != int64(2) {
		t.Fatalf("expected numbers 1 and 2, got %v and %v", first.Records[0]["number"], second.Records[0]["number"])
	}
}

func TestAuto_IndependentCounters(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `event { id: int @id @auto seq: int @auto name: text };`)

	first := runStatements(t, d, `>> event {name: "a"} => {*};`).(InsertResult)
	second := runStatements(t, d, `>> event {name: "b"} => {*};`).(InsertResult)

	for _, field := range []string{"id", "seq"} {
		if first.Records[0][field] != int64(1) || second.Records[0][field] != int64(2) {
			t.Fatalf("expected %s to count 1, 2, got %v, %v", field, first.Records[0][field], second.Records[0][field])
		}
	}
}

func TestAuto_CollectionWithNoPrimaryKey(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `log { seq: int @auto message: text };`)

	result := runStatements(t, d, `>> log {message: "hi"} => {*};`).(InsertResult)
	if result.Records[0]["seq"] != int64(1) {
		t.Fatalf("expected seq 1, got %v", result.Records[0]["seq"])
	}
}

func TestAuto_InsertRejectsSuppliedNonIDAutoField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `ticket { code: text @id number: int @auto };`)

	for _, statement := range []string{`>> ticket {code: "A" number: 7};`, `>> ticket {code: "A" number: null};`} {
		got := runExpectingError(t, d, statement)

		want := `field "number" is auto-increment and must not be supplied for collection "ticket"`
		if got != want {
			t.Fatalf("for %q expected error %q, got %q", statement, want, got)
		}
	}

	expectCount(t, d, `<< ticket;`, "0")
}

func TestAuto_FailedInsertDoesNotConsumeValue(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `ticket { code: text @id number: int @auto };`, `>> ticket {code: "A"};`)

	got := runExpectingError(t, d, `>> ticket {code: "A"};`)
	if got != `duplicate primary key A for collection "ticket"` {
		t.Fatalf("expected a duplicate key error, got %q", got)
	}

	result := runStatements(t, d, `>> ticket {code: "B"} => {*};`).(InsertResult)
	if result.Records[0]["number"] != int64(2) {
		t.Fatalf("expected number 2, got %v", result.Records[0]["number"])
	}
}

func TestAuto_MergeRejectsPayloadSettingNonIDAutoField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `ticket { code: text @id number: int @auto };`, `>> ticket {code: "A"};`)

	got := runExpectingError(t, d, `~> ticket() {number: 5};`)

	want := `payload must not set auto-increment field "number" for collection "ticket"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< ticket(number: 1);`, "1")
}

func TestAuto_MergeReportsPrimaryKeyErrorForAutoPrimaryKey(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text };`, `>> user {name: "Matt"};`)

	got := runExpectingError(t, d, `~> user() {id: 5};`)

	want := `payload must not set primary key "id" for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestAuto_MergeCanFilterOnAutoField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`ticket { code: text @id number: int @auto title: text @optional };`,
		`>> ticket {code: "A"};`,
		`~> ticket(number: 1) {title: "hi"};`,
	)

	expectCount(t, d, `<< ticket(title: "hi");`, "1")
}
