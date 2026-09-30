package core

import "testing"

func TestParseFilter_WildcardObject(t *testing.T) {
	o, err := ParseRead(`<< user(address: {*} geo: {lat: 1.0});`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.Filter["address"].(wildcardObject); !ok {
		t.Fatalf("expected address to be the {*} wildcard, got %#v", o.Filter["address"])
	}
}

func TestParseFilter_WildcardNestedInSubset(t *testing.T) {
	o, err := ParseRead(`<< user(address: {geo: {*}});`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.Filter["address"].(Entity)["geo"].(wildcardObject); !ok {
		t.Fatalf("expected address.geo to be the {*} wildcard, got %#v", o.Filter["address"])
	}
}

func TestParse_WildcardObjectOnlyInFilters(t *testing.T) {
	for _, input := range []string{
		`>> user {id: 1 address: {*}};`,
		`~> user(id: 1) {address: {*}};`,
		`~> user(address: {*}) {address: {*}};`,
	} {
		_, err := Parse(input)
		if err == nil || err.Error() != "{*} is only allowed in a filter" {
			t.Fatalf("for %q expected the {*} error, got %v", input, err)
		}
	}
}

func TestParse_WildcardObjectMustBeAlone(t *testing.T) {
	for _, input := range []string{`<< user(address: {* city: "MSP"});`, `<< user(address: {*, city: "MSP"});`} {
		if _, err := Parse(input); err == nil {
			t.Fatalf("expected a parse error for %q", input)
		}
	}
}

func TestParse_FilterConstrainsAFieldOnlyOnce(t *testing.T) {
	cases := map[string]string{
		`<< user(address.city: "A" address: {city: "B"});`:          `field "address.city" is given more than once`,
		`<< user(address: {geo: {lat: 1.0}} address.geo.lat: 2.0);`: `field "address.geo.lat" is given more than once`,
	}

	for input, want := range cases {
		_, err := Parse(input)
		if err == nil || err.Error() != want {
			t.Fatalf("for %q expected error %q, got %v", input, want, err)
		}
	}

	if _, err := Parse(`<< user(address.city: "MSP" address: {street: "1 Main"});`); err != nil {
		t.Fatalf("expected conditions on different fields to parse, got %v", err)
	}
}

const optionalAddress = `user { id: int @id address: { street: text @optional city: text zip: text @optional } @optional };`

func setupFilterUsers(t *testing.T) *Database {
	t.Helper()

	d := NewDatabase("test")
	runStatements(t, d, optionalAddress, `>> user {id: 1} `+
		`& {id: 2 address: {city: "MSP"}} `+
		`& {id: 3 address: {city: "MSP" zip: "55401" street: "1 Main"}} `+
		`& {id: 4 address: {city: "STP"}};`)
	return d
}

func ids(t *testing.T, d *Database, filter string) string {
	t.Helper()
	return runStatements(t, d, `<< user`+filter+` => {id};`).(ReadResult).String()
}

func TestFilter_DottedPathEquality(t *testing.T) {
	d := setupFilterUsers(t)

	if got := ids(t, d, `(address.city: "MSP")`); got != "id\n2\n3" {
		t.Fatalf("expected ids 2 and 3, got:\n%s", got)
	}
}

func TestFilter_NullOnDottedPathMatchesMissingObject(t *testing.T) {
	d := setupFilterUsers(t)

	if got := ids(t, d, `(address.zip: null)`); got != "id\n1\n2\n4" {
		t.Fatalf("expected ids 1, 2, and 4, got:\n%s", got)
	}
}

func TestFilter_SubsetIgnoresUnmentionedFields(t *testing.T) {
	d := setupFilterUsers(t)

	if got := ids(t, d, `(address: {city: "MSP"})`); got != "id\n2\n3" {
		t.Fatalf("expected ids 2 and 3, got:\n%s", got)
	}

	if got := ids(t, d, `(address: {city: "MSP" zip: "55401"})`); got != "id\n3" {
		t.Fatalf("expected id 3, got:\n%s", got)
	}
}

func TestFilter_NullInsideSubsetRequiresTheObject(t *testing.T) {
	d := setupFilterUsers(t)

	if got := ids(t, d, `(address: {zip: null})`); got != "id\n2\n4" {
		t.Fatalf("expected ids 2 and 4 (not 1, which has no address), got:\n%s", got)
	}
}

func TestFilter_NullObject(t *testing.T) {
	d := setupFilterUsers(t)

	if got := ids(t, d, `(address: null)`); got != "id\n1" {
		t.Fatalf("expected id 1, got:\n%s", got)
	}
}

func TestFilter_WildcardObject(t *testing.T) {
	d := setupFilterUsers(t)

	if got := ids(t, d, `(address: {*})`); got != "id\n2\n3\n4" {
		t.Fatalf("expected ids 2, 3, and 4, got:\n%s", got)
	}
}

func TestFilter_WildcardMatchesAnEmptyStoredObject(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { zip: text @optional } @optional };`,
		`>> user {id: 1 address: {zip: null}} & {id: 2};`)

	if got := ids(t, d, `(address: {*})`); got != "id\n1" {
		t.Fatalf("expected id 1, whose address is stored empty, got:\n%s", got)
	}
}

func TestFilter_NestedSubset(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text geo: { lat: float } } };`,
		`>> user {id: 1 address: {city: "MSP" geo: {lat: 44.9}}} & {id: 2 address: {city: "MSP" geo: {lat: 45.0}}};`)

	expectCount(t, d, `<< user(address: {geo: {lat: 44.9}});`, "1")
	expectCount(t, d, `<< user(address.geo: {lat: 45.0});`, "1")
	expectCount(t, d, `<< user(address.geo.lat: 44.9);`, "1")
}

func TestFilter_DifferentFieldsOfTheSameObject(t *testing.T) {
	d := setupFilterUsers(t)

	if got := ids(t, d, `(address.city: "MSP" address: {street: "1 Main"})`); got != "id\n3" {
		t.Fatalf("expected id 3, got:\n%s", got)
	}
}

func TestFilter_DottedPathsInDeleteAndMerge(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text address: { city: text } @optional };`,
		`>> user {id: 1 name: "A" address: {city: "MSP"}} & {id: 2 name: "B" address: {city: "STP"}} & {id: 3 name: "C"};`)

	if got := runStatements(t, d, `~> user(address: {city: "STP"}) {name: "Z"};`).(MergeResult).String(); got != "1" {
		t.Fatalf("expected the merge to update 1 record, got %s", got)
	}
	expectCount(t, d, `<< user(name: "Z");`, "1")

	if got := runStatements(t, d, `!> user(address.city: "MSP");`).(DeleteResult).String(); got != "1" {
		t.Fatalf("expected the delete to remove 1 record, got %s", got)
	}

	if got := runStatements(t, d, `!> user(address: null);`).(DeleteResult).String(); got != "1" {
		t.Fatalf("expected the delete to remove the record with no address, got %s", got)
	}

	expectCount(t, d, `<< user;`, "1")
}

func TestFilter_ValuesAreValidatedAtTheirPath(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } };`)

	cases := map[string]string{
		`<< user(address.zip: "55401");`:  `field "address.zip" not found in schema for collection "user"`,
		`<< user(id.x: 1);`:               `field "id.x" not found in schema for collection "user"`,
		`<< user(address.city: 5);`:       `field "address.city": expected text, got int`,
		`<< user(address: {city: 5});`:    `field "address.city": expected text, got int`,
		`<< user(address: {zip: "x"});`:   `field "address.zip" not found in schema for collection "user"`,
		`<< user(address: "MSP");`:        `field "address": expected object, got text`,
		`<< user(address.city: {a: 1});`:  `field "address.city": expected text, got object`,
		`<< user(id: {*});`:               `field "id": expected int, got {*}`,
		`<< user(address.city: null);`:    `field "address.city" is required and cannot be null`,
		`<< user(address: null);`:         `field "address" is required and cannot be null`,
		`<< user(address: {city: null});`: `field "address.city" is required and cannot be null`,
		`<< user(address: {});`:           `empty object filter for field "address"; use {*} to match any value`,
	}

	for statement, want := range cases {
		if got := runExpectingError(t, d, statement); got != want {
			t.Fatalf("for %q expected error %q, got %q", statement, want, got)
		}
	}
}

func TestFilter_NullAllowedThroughAnOptionalBlock(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id address: { city: text } @optional };`, `>> user {id: 1};`)

	expectCount(t, d, `<< user(address.city: null);`, "1")

	// Inside a subset the object is asserted present, so only the field's own
	// optionality counts: city is required.
	got := runExpectingError(t, d, `<< user(address: {city: null});`)
	if got != `field "address.city" is required and cannot be null` {
		t.Fatalf("expected the required error, got %q", got)
	}
}
