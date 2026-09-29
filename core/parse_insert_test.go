package core

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseInsert_ParsesFlatValues(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1, name: "Matt", age: 39};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Collection != "user" {
		t.Fatalf("expected collection %q, got %q", "user", o.Collection)
	}

	if o.Records[0]["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v (%T)", o.Records[0]["id"], o.Records[0]["id"])
	}

	if o.Records[0]["name"] != "Matt" {
		t.Fatalf("expected name %q, got %v", "Matt", o.Records[0]["name"])
	}

	if o.Records[0]["age"] != int64(39) {
		t.Fatalf("expected age 39, got %v (%T)", o.Records[0]["age"], o.Records[0]["age"])
	}
}

func TestParseInsert_ParsesFloats(t *testing.T) {
	o, err := ParseInsert(`>> product {price: 9.99};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Records[0]["price"] != 9.99 {
		t.Fatalf("expected price 9.99, got %v (%T)", o.Records[0]["price"], o.Records[0]["price"])
	}
}

func TestParseInsert_AllowsTrailingComma(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1,};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Records[0]) != 1 {
		t.Fatalf("expected 1 field, got %d", len(o.Records[0]))
	}
}

func TestParseInsert_AllowsEmptyRecord(t *testing.T) {
	o, err := ParseInsert(`>> user {};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Records[0]) != 0 {
		t.Fatalf("expected no fields, got %d", len(o.Records[0]))
	}
}

func TestParseInsert_UnescapesStrings(t *testing.T) {
	o, err := ParseInsert(`>> user {name: "say \"hi\" \\ bye"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	want := `say "hi" \ bye`
	if o.Records[0]["name"] != want {
		t.Fatalf("expected %q, got %q", want, o.Records[0]["name"])
	}
}

func TestParseInsert_ToleratesTrailingSemicolon(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Records[0]["id"] != int64(1) {
		t.Fatalf("expected id 1, got %v", o.Records[0]["id"])
	}
}

func TestParseInsert_CommasAreOptional(t *testing.T) {
	withoutCommas, err := ParseInsert(`>> user {id: 1 name: "Matt"};`)
	if err != nil {
		t.Fatalf("Failed to parse without commas, err: %v", err)
	}

	withCommas, err := ParseInsert(`>> user {id: 1, name: "Matt"};`)
	if err != nil {
		t.Fatalf("Failed to parse with commas, err: %v", err)
	}

	if !reflect.DeepEqual(withoutCommas, withCommas) {
		t.Fatalf("expected identical operations, got %v and %v", withoutCommas, withCommas)
	}
}

func TestParseInsert_AllowsMixedAndTrailingCommas(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1, name: "Matt" age: 39,};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Records[0]) != 3 || o.Records[0]["age"] != int64(39) {
		t.Fatalf("expected all three fields, got %v", o.Records[0])
	}
}

func TestParseInsert_CommaInsideStringIsPartOfTheValue(t *testing.T) {
	o, err := ParseInsert(`>> user {name: "Smith, Matt"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Records[0]["name"] != "Smith, Matt" {
		t.Fatalf("expected %q, got %q", "Smith, Matt", o.Records[0]["name"])
	}
}

func TestParseInsert_RejectsUnterminatedString(t *testing.T) {
	if _, err := ParseInsert(`>> user {name: "Matt`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for an unterminated string, got %v", err)
	}
}

func TestParseInsert_RejectsIncompleteInsertOperator(t *testing.T) {
	if _, err := ParseInsert(`>`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a lone '>', got %v", err)
	}
}

func TestParseInsert_RejectsIncompleteWithNoRecordLiteral(t *testing.T) {
	if _, err := ParseInsert(`>> user`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput for a collection name with no record literal yet, got %v", err)
	}
}

func TestParse_DispatchesToInsert(t *testing.T) {
	o, err := Parse(`>> user {id: 1};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if _, ok := o.(InsertOperation); !ok {
		t.Fatalf("expected an InsertOperation, got %T", o)
	}
}

func TestParseInsert_ParsesBooleans(t *testing.T) {
	o, err := ParseInsert(`>> user {active: true archived: false};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Records[0]["active"] != true {
		t.Fatalf("expected active true, got %v (%T)", o.Records[0]["active"], o.Records[0]["active"])
	}

	if o.Records[0]["archived"] != false {
		t.Fatalf("expected archived false, got %v (%T)", o.Records[0]["archived"], o.Records[0]["archived"])
	}
}

func TestParseInsert_QuotedTrueIsAString(t *testing.T) {
	o, err := ParseInsert(`>> user {active: "true"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Records[0]["active"] != "true" {
		t.Fatalf("expected the string %q, got %v (%T)", "true", o.Records[0]["active"], o.Records[0]["active"])
	}
}

func TestParseInsert_RejectsOtherBareWordsAsValues(t *testing.T) {
	_, err := ParseInsert(`>> user {active: yes};`)
	if err == nil {
		t.Fatal("expected an error for a bare word value")
	}

	want := `expected a value, got "yes"`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestParseInsert_ParsesNull(t *testing.T) {
	o, err := ParseInsert(`>> user {nickname: null};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	value, ok := o.Records[0]["nickname"]
	if !ok {
		t.Fatal("expected nickname to be present in the parsed record")
	}

	if value != nil {
		t.Fatalf("expected nickname nil, got %v (%T)", value, value)
	}
}

func TestParseInsert_QuotedNullIsAString(t *testing.T) {
	o, err := ParseInsert(`>> user {nickname: "null"};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Records[0]["nickname"] != "null" {
		t.Fatalf("expected the string %q, got %v (%T)", "null", o.Records[0]["nickname"], o.Records[0]["nickname"])
	}
}

func TestParseInsert_NoProjectionIsNil(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Projection != nil {
		t.Fatalf("expected nil projection, got %v", o.Projection)
	}
}

func TestParseInsert_ParsesProjection(t *testing.T) {
	o, err := ParseInsert(`>> user {name: "Matt"} => {id, name}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !reflect.DeepEqual(o.Projection, []string{"id", "name"}) {
		t.Fatalf("expected projection [id name], got %v", o.Projection)
	}
}

func TestParseInsert_ParsesWildcardProjection(t *testing.T) {
	o, err := ParseInsert(`>> user {name: "Matt"} => {*};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !reflect.DeepEqual(o.Projection, []string{"*"}) {
		t.Fatalf("expected projection [*], got %v", o.Projection)
	}
}

func TestParseInsert_EmptyProjectionIsNotNil(t *testing.T) {
	o, err := ParseInsert(`>> user {name: "Matt"} => {}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Projection == nil {
		t.Fatal("expected a non-nil empty projection for `=> {}`")
	}
}

func TestParseInsert_UnterminatedRecordIsIncomplete(t *testing.T) {
	if _, err := ParseInsert(`>> user {name: "Matt"}`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput, got %v", err)
	}
}

func TestParseInsert_RejectsWildcardCombinedWithFields(t *testing.T) {
	for _, input := range []string{`>> user {name: "Matt"} => {*, id}`, `>> user {name: "Matt"} => {id, *}`} {
		if _, err := ParseInsert(input); err == nil {
			t.Fatalf("expected an error for %q", input)
		}
	}
}

func TestParseInsert_ParsesBatch(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1} & {id: 2} & {id: 3};`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(o.Records))
	}

	for i, record := range o.Records {
		if record["id"] != int64(i+1) {
			t.Fatalf("expected record %d to have id %d, got %v", i+1, i+1, record["id"])
		}
	}
}

func TestParseInsert_TrailingAmpersandIsIncomplete(t *testing.T) {
	if _, err := ParseInsert(`>> user {id: 1} &`); !errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected ErrIncompleteInput, got %v", err)
	}
}

func TestParseInsert_DanglingAmpersandIsAnError(t *testing.T) {
	_, err := ParseInsert(`>> user {id: 1} & ;`)
	if err == nil || errors.Is(err, ErrIncompleteInput) {
		t.Fatalf("expected a parse error, got %v", err)
	}
}

func TestParseInsert_BatchProjectionFollowsLastRecord(t *testing.T) {
	o, err := ParseInsert(`>> user {id: 1} & {id: 2} => {id}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Records) != 2 || !reflect.DeepEqual(o.Projection, []string{"id"}) {
		t.Fatalf("expected 2 records and projection [id], got %d and %v", len(o.Records), o.Projection)
	}
}

func TestParseInsert_ProjectionBetweenRecordsIsAnError(t *testing.T) {
	if _, err := ParseInsert(`>> user {id: 1} => {id} & {id: 2}`); err == nil {
		t.Fatal("expected an error for a projection between records")
	}
}
