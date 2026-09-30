package core

import "testing"

const userWithAddress = `user { id: int @id address: { street: text city: text } };`

func TestEmbedded_InsertStoresNestedRecord(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, userWithAddress, `>> user {id: 1 address: {street: "1 Main" city: "MSP"}};`)

	address := d.collections["user"].records[0]["address"].(Entity)
	if address["street"] != "1 Main" || address["city"] != "MSP" {
		t.Fatalf("expected the nested address to be stored, got %v", address)
	}
}

func TestEmbedded_WrongTypeInsideBlock(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } };`)

	got := runExpectingError(t, d, `>> user {id: 1 address: {city: 5}};`)

	want := `field "address.city": expected text, got int`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestEmbedded_MissingRequiredFieldInsideBlock(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, userWithAddress)

	got := runExpectingError(t, d, `>> user {id: 1 address: {city: "MSP"}};`)

	want := `field "address.street" is required for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestEmbedded_UndeclaredFieldInsideBlock(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } };`)

	got := runExpectingError(t, d, `>> user {id: 1 address: {city: "MSP" zip: "55401"}};`)

	want := `field "address.zip" not found in schema for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestEmbedded_OptionalBlockOmitted(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } @optional };`, `>> user {id: 1};`, `>> user {id: 2 address: null};`)

	expectCount(t, d, `<< user;`, "2")
}

func TestEmbedded_RequiredBlockOmitted(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } };`)

	got := runExpectingError(t, d, `>> user {id: 1};`)

	want := `field "address" is required for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestEmbedded_ScalarOnBlockField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } };`)

	got := runExpectingError(t, d, `>> user {id: 1 address: 5};`)

	want := `field "address": expected object, got int`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestEmbedded_ObjectOnScalarField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	got := runExpectingError(t, d, `>> user {id: 1 name: {first: "Matt"}};`)

	want := `field "name": expected text, got object`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestEmbedded_NestedProblemsAreAllReported(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { street: text city: text geo: { lat: float } } };`)

	got := runExpectingError(t, d, `>> user {id: 1 address: {city: 5 geo: {lat: 1}}};`)

	want := "3 problems, nothing inserted:\n" +
		"  field \"address.city\": expected text, got int\n" +
		"  field \"address.geo.lat\": expected float, got int\n" +
		"  field \"address.street\" is required for collection \"user\""
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestEmbedded_OptionalFieldInsideBlockNullIsStoredAbsent(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id address: { city: text zip: text @optional } };`,
		`>> user {id: 1 address: {city: "MSP" zip: null}};`,
	)

	address := d.collections["user"].records[0]["address"].(Entity)
	if _, present := address["zip"]; present {
		t.Fatalf("expected zip to be stored as absent, got %v", address)
	}
}

func TestEmbedded_BatchWithNestedRecords(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, userWithAddress)

	got := runExpectingError(t, d, `>> user {id: 1 address: {street: "1 Main" city: "MSP"}} & {id: 2 address: {street: "2 Oak"}};`)

	want := `record 2: field "address.city" is required for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	runStatements(t, d, `>> user {id: 1 address: {street: "1 Main" city: "MSP"}} & {id: 2 address: {street: "2 Oak" city: "STP"}};`)
	expectCount(t, d, `<< user;`, "2")
}

func TestProjection_ResolvesToNonNilEvenWhenEmpty(t *testing.T) {
	for _, projection := range [][]string{{}, {"*"}} {
		columns, problems := resolveProjection(&Collection{name: "empty", data: map[string]DataType{}}, projection)
		if columns == nil || len(problems) != 0 {
			t.Fatalf("for %v expected empty non-nil columns and no problems, got %#v, %v", projection, columns, problems)
		}
	}
}

func setupUserWithAddress(t *testing.T) *Database {
	t.Helper()

	d := NewDatabase("test")
	runStatements(t, d, userWithAddress, `>> user {id: 1 address: {street: "1 Main" city: "MSP"}};`)
	return d
}

func readTable(t *testing.T, d *Database, statement string) string {
	t.Helper()
	return runStatements(t, d, statement).(ReadResult).String()
}

func TestProjection_WildcardFlattensObjects(t *testing.T) {
	d := setupUserWithAddress(t)

	got := readTable(t, d, `<< user => {*};`)

	want := "address.city  address.street  id\n" +
		"MSP           1 Main          1"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestProjection_NamingAnObjectFieldExpandsToItsLeaves(t *testing.T) {
	d := setupUserWithAddress(t)

	got := readTable(t, d, `<< user => {id address};`)

	want := "id  address.city  address.street\n" +
		"1   MSP           1 Main"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestProjection_DottedPath(t *testing.T) {
	d := setupUserWithAddress(t)

	got := readTable(t, d, `<< user => {address.city};`)

	want := "address.city\nMSP"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestProjection_RepeatedColumnsAreKeptOnce(t *testing.T) {
	d := setupUserWithAddress(t)

	got := readTable(t, d, `<< user => {address.city address id};`)

	want := "address.city  address.street  id\n" +
		"MSP           1 Main          1"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestProjection_AbsentOptionalObjectShowsNull(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } @optional };`, `>> user {id: 1};`)

	got := readTable(t, d, `<< user => {*};`)

	want := "address.city  id\nnull          1"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestProjection_UnknownPath(t *testing.T) {
	d := setupUserWithAddress(t)

	for _, statement := range []string{`<< user => {address.zip};`, `<< user => {id.x};`, `<< user => {nope.city};`} {
		op, err := Parse(statement)
		if err != nil {
			t.Fatalf("Failed to parse %q, err: %v", statement, err)
		}

		_, err = d.Run(op)
		if err == nil {
			t.Fatalf("expected an error for %q", statement)
		}
	}

	got := runExpectingError(t, d, `<< user => {address.zip};`)
	want := `field "address.zip" not found in schema for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestProjection_InsertWildcardShowsNestedRecord(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, userWithAddress)

	got := runStatements(t, d, `>> user {id: 1 address: {street: "1 Main" city: "MSP"}} => {*};`).(InsertResult).String()

	want := "address.city  address.street  id\n" +
		"MSP           1 Main          1"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestProjection_DeleteReturnsFlattenedColumns(t *testing.T) {
	d := setupUserWithAddress(t)

	got := runStatements(t, d, `!> user(id: 1) => {id address.city};`).(DeleteResult).String()

	want := "id  address.city\n1   MSP"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestEmbedded_FilteringOnObjectFieldsWorksInEveryStatement(t *testing.T) {
	d := setupUserWithAddress(t)

	expectCount(t, d, `<< user(address: {city: "MSP"});`, "1")
	expectCount(t, d, `<< user(address.city: "MSP");`, "1")
	expectCount(t, d, `<< user(address.city: "STP");`, "0")

	if got := runStatements(t, d, `!> user(address: {street: "1 Main"});`).(DeleteResult).String(); got != "1" {
		t.Fatalf("expected the delete to remove 1 record, got %s", got)
	}

	expectCount(t, d, `<< user;`, "0")
}

func TestEmbedded_FilteringOnUnknownDottedPathIsNotFound(t *testing.T) {
	d := setupUserWithAddress(t)

	got := runExpectingError(t, d, `<< user(id.x: 1);`)

	want := `field "id.x" not found in schema for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestEmbedded_MergePayloadCanSetObject(t *testing.T) {
	d := setupUserWithAddress(t)

	runStatements(t, d, `~> user(id: 1) {address: {street: "2 Oak" city: "STP"}};`)

	address := d.collections["user"].records[0]["address"].(Entity)
	if address["street"] != "2 Oak" || address["city"] != "STP" {
		t.Fatalf("expected the address to be updated, got %v", address)
	}
}

func TestEmbedded_MergeStillUpdatesScalarFieldsAlongsideObjects(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text address: { city: text } };`,
		`>> user {id: 1 name: "Matt" address: {city: "MSP"}};`,
		`~> user(id: 1) {name: "Sam"};`,
	)

	got := readTable(t, d, `<< user => {*};`)

	want := "address.city  id  name\nMSP           1   Sam"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestEmbedded_MergeFilterOnObjectField(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text address: { city: text } };`,
		`>> user {id: 1 name: "Matt" address: {city: "MSP"}} & {id: 2 name: "Sam" address: {city: "STP"}};`,
	)

	got := runStatements(t, d, `~> user(address.city: "MSP") {name: "Pat"};`).(MergeResult).String()
	if got != "1" {
		t.Fatalf("expected the merge to update 1 record, got %s", got)
	}

	expectCount(t, d, `<< user(name: "Pat" address: {city: "MSP"});`, "1")
	expectCount(t, d, `<< user(name: "Sam");`, "1")
}
