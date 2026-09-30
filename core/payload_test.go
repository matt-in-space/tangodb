package core

import (
	"reflect"
	"testing"
)

func TestParseMerge_DottedKeysBecomeNested(t *testing.T) {
	o, err := ParseMerge(`~> user(id: 1) {name: "M" address.city: "STP" address: {zip: null geo.lat: 1.0}};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	want := Entity{
		"name": "M",
		"address": Entity{
			"city": "STP",
			"zip":  nil,
			"geo":  Entity{"lat": 1.0},
		},
	}
	if !reflect.DeepEqual(o.Payload, want) {
		t.Fatalf("expected %#v, got %#v", want, o.Payload)
	}
}

func TestParseMerge_PayloadConflicts(t *testing.T) {
	cases := map[string]string{
		`~> user(id: 1) {address.city: "A" address: {city: "B"}};`:       `field "address.city" is given more than once`,
		`~> user(id: 1) {address: null address.city: "X"};`:              `field "address.city" conflicts with "address"`,
		`~> user(id: 1) {address.geo: null address: {geo: {lat: 1.0}}};`: `field "address.geo.lat" conflicts with "address.geo"`,
		`~> user(id: 1) {address.city: "A" address.city: "B"};`:          `field "address.city" is given more than once`,
	}

	for input, want := range cases {
		_, err := Parse(input)
		if err == nil || err.Error() != want {
			t.Fatalf("for %q expected error %q, got %v", input, want, err)
		}
	}
}

func TestParseMerge_DifferentFieldsOfAnObjectCombine(t *testing.T) {
	o, err := ParseMerge(`~> user(id: 1) {address.city: "A" address: {street: "B"}};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	want := Entity{"address": Entity{"city": "A", "street": "B"}}
	if !reflect.DeepEqual(o.Payload, want) {
		t.Fatalf("expected %#v, got %#v", want, o.Payload)
	}
}

func TestParse_DottedKeysStillRejectedInInserts(t *testing.T) {
	_, err := Parse(`>> user {id: 1 address.city: "MSP"};`)
	if err == nil || err.Error() != `field name "address.city" cannot contain "."` {
		t.Fatalf("expected the dotted-name error, got %v", err)
	}

	_, err = Parse(`>> user {id: 1 address: {geo.lat: 1.0}};`)
	if err == nil || err.Error() != `field name "geo.lat" cannot contain "."` {
		t.Fatalf("expected the dotted-name error in a nested insert record, got %v", err)
	}
}

const fullAddress = `user { id: int @id name: text address: { street: text city: text zip: text @optional } @optional };`

func setupMergeUsers(t *testing.T) *Database {
	t.Helper()

	d := NewDatabase("test")
	runStatements(t, d, fullAddress,
		`>> user {id: 1 name: "A" address: {street: "1 Main" city: "MSP" zip: "55401"}} & {id: 2 name: "B"};`)
	return d
}

func storedAddress(t *testing.T, d *Database, id int64) Entity {
	t.Helper()

	for _, record := range d.collections["user"].records {
		if record["id"] == id {
			address, _ := record["address"].(Entity)
			return address
		}
	}
	t.Fatalf("no record with id %d", id)
	return nil
}

func TestMerge_UpdatesOneFieldOfAnObject(t *testing.T) {
	d := setupMergeUsers(t)
	runStatements(t, d, `~> user(id: 1) {address: {city: "STP"}};`)

	want := Entity{"street": "1 Main", "city": "STP", "zip": "55401"}
	if got := storedAddress(t, d, 1); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestMerge_DottedDeepUpdate(t *testing.T) {
	d := setupMergeUsers(t)
	runStatements(t, d, `~> user(id: 1) {address.city: "STP"};`)

	want := Entity{"street": "1 Main", "city": "STP", "zip": "55401"}
	if got := storedAddress(t, d, 1); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestMerge_ClearsAFieldInsideAnObject(t *testing.T) {
	d := setupMergeUsers(t)
	runStatements(t, d, `~> user(id: 1) {address: {zip: null}};`)

	want := Entity{"street": "1 Main", "city": "MSP"}
	if got := storedAddress(t, d, 1); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestMerge_ClearsAnOptionalObject(t *testing.T) {
	d := setupMergeUsers(t)
	runStatements(t, d, `~> user(id: 1) {address: null};`)

	expectCount(t, d, `<< user(address: null);`, "2")
}

func TestMerge_CreatesAMissingObject(t *testing.T) {
	d := setupMergeUsers(t)
	runStatements(t, d, `~> user(id: 2) {address: {street: "2 Oak" city: "STP"}};`)

	expectCount(t, d, `<< user(address.city: "STP");`, "1")
}

func TestMerge_EmptyPayloadObject(t *testing.T) {
	d := setupMergeUsers(t)

	got := runExpectingError(t, d, `~> user(id: 1) {address: {}};`)
	if got != `empty object in merge payload for field "address"` {
		t.Fatalf("expected the empty object error, got %q", got)
	}
}

func TestMerge_PayloadValuesAreValidatedAtTheirPath(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } };`, `>> user {id: 1 address: {city: "MSP"}};`)

	cases := map[string]string{
		`~> user() {address.city: 5};`:         `field "address.city": expected text, got int`,
		`~> user() {address: {zip: "55401"}};`: `field "address.zip" not found in schema for collection "user"`,
		`~> user() {address.zip: "55401"};`:    `field "address.zip" not found in schema for collection "user"`,
		`~> user() {address: null};`:           `field "address" is required and cannot be null`,
		`~> user() {address: {city: null}};`:   `field "address.city" is required and cannot be null`,
		`~> user() {address: "MSP"};`:          `field "address": expected object, got text`,
		`~> user() {id.x: 1};`:                 `payload must not set primary key "id" for collection "user"`,
	}

	for statement, want := range cases {
		if got := runExpectingError(t, d, statement); got != want {
			t.Fatalf("for %q expected error %q, got %q", statement, want, got)
		}
	}

	expectCount(t, d, `<< user(address.city: "MSP");`, "1")
}

func TestMerge_CreatingAnIncompleteObjectChangesNothing(t *testing.T) {
	d := setupMergeUsers(t)

	got := runExpectingError(t, d, `~> user() {address: {city: "STP"}};`)
	if got != `record with id 2: field "address.street" is required for collection "user"` {
		t.Fatalf("unexpected error %q", got)
	}

	if got := storedAddress(t, d, 1)["city"]; got != "MSP" {
		t.Fatalf("expected record 1 to be unchanged, got city %v", got)
	}
	if storedAddress(t, d, 2) != nil {
		t.Fatal("expected record 2 to still have no address")
	}
}

func TestMerge_ReportsEveryInvalidResult(t *testing.T) {
	d := setupMergeUsers(t)
	runStatements(t, d, `>> user {id: 3 name: "C"};`)

	got := runExpectingError(t, d, `~> user(address: null) {address: {city: "STP"}};`)

	want := "2 problems, nothing changed:\n" +
		"  record with id 2: field \"address.street\" is required for collection \"user\"\n" +
		"  record with id 3: field \"address.street\" is required for collection \"user\""
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestMerge_OneInvalidResultChangesNothingElse(t *testing.T) {
	d := setupMergeUsers(t)

	runExpectingError(t, d, `~> user() {name: "Z" address.city: "STP"};`)

	expectCount(t, d, `<< user(name: "Z");`, "0")
	if got := storedAddress(t, d, 1)["city"]; got != "MSP" {
		t.Fatalf("expected record 1's address to be unchanged, got city %v", got)
	}
}

func TestMerge_CollectionWithoutPrimaryKeyLabelsByPosition(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `log { message: text meta: { source: text level: text } @optional };`, `>> log {message: "a"};`)

	got := runExpectingError(t, d, `~> log() {meta.level: "warn"};`)
	if got != `matched record 1: field "meta.source" is required for collection "log"` {
		t.Fatalf("unexpected error %q", got)
	}
}

func TestMerge_TextPrimaryKeyIsQuotedInLabels(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `ticket { code: text @id meta: { a: text b: text } @optional };`, `>> ticket {code: "A"};`)

	got := runExpectingError(t, d, `~> ticket() {meta.a: "x"};`)
	if got != `record with code "A": field "meta.b" is required for collection "ticket"` {
		t.Fatalf("unexpected error %q", got)
	}
}

func TestMerge_RecordsNeverShareNestedObjects(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } @optional };`,
		`>> user {id: 1} & {id: 2};`,
		`~> user() {address: {city: "MSP"}};`,
		`~> user(id: 1) {address.city: "STP"};`,
	)

	if got := storedAddress(t, d, 2)["city"]; got != "MSP" {
		t.Fatalf("expected record 2's address to be its own copy, got city %v", got)
	}
}

func TestMerge_ProjectionReturnsMergedState(t *testing.T) {
	d := setupMergeUsers(t)

	got := runStatements(t, d, `~> user(id: 1) {address.city: "STP"} => {id address.city address.street};`).(MergeResult).String()

	want := "id  address.city  address.street\n1   STP           1 Main"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
