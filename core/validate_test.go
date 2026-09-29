package core

import (
	"reflect"
	"testing"
)

func TestValidateValue_AcceptsMatchingTypes(t *testing.T) {
	cases := []struct {
		dataType DataType
		value    any
	}{
		{TypeInt, int64(1)},
		{TypeFloat, 1.5},
		{TypeText, "Matt"},
		{TypeBool, true},
	}

	for _, c := range cases {
		if err := validateValue(c.dataType, c.value); err != nil {
			t.Errorf("expected %v to be valid for %s, got %v", c.value, c.dataType, err)
		}
	}
}

func TestValidateValue_RejectsMismatchedTypes(t *testing.T) {
	cases := []struct {
		dataType DataType
		value    any
		want     string
	}{
		{TypeInt, "1", "expected int, got text"},
		{TypeInt, 1.0, "expected int, got float"},
		{TypeFloat, int64(10), "expected float, got int"},
		{TypeText, true, "expected text, got bool"},
		{TypeBool, "true", "expected bool, got text"},
		{TypeInt, 1, "expected int, got unsupported Go type int"},
	}

	for _, c := range cases {
		err := validateValue(c.dataType, c.value)
		if err == nil {
			t.Errorf("expected %v (%T) to be rejected for %s", c.value, c.value, c.dataType)
			continue
		}

		if err.Error() != c.want {
			t.Errorf("expected error %q, got %q", c.want, err.Error())
		}
	}
}

func TestValidateFields_RejectsUndeclaredField(t *testing.T) {
	collection := &Collection{name: "user", data: map[string]DataType{"name": TypeText}}

	err := validateFields(collection, map[string]any{"nmae": "Matt"})
	if err == nil {
		t.Fatal("expected an error for an undeclared field")
	}

	want := `field "nmae" not found in schema for collection "user"`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestValidateFields_NamesTheFieldOnTypeMismatch(t *testing.T) {
	collection := &Collection{name: "user", data: map[string]DataType{"age": TypeInt}}

	err := validateFields(collection, map[string]any{"age": "old"})
	if err == nil {
		t.Fatal("expected an error for a type mismatch")
	}

	want := `field "age": expected int, got text`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestValidateFields_ReportsFirstBadFieldInSortedOrder(t *testing.T) {
	collection := &Collection{name: "user", data: map[string]DataType{"age": TypeInt, "name": TypeText}}

	fields := map[string]any{"name": int64(1), "age": "old", "zzz": 1}

	for range 20 {
		err := validateFields(collection, fields)
		if err == nil {
			t.Fatal("expected an error")
		}

		want := `field "age": expected int, got text`
		if err.Error() != want {
			t.Fatalf("expected error %q, got %q", want, err.Error())
		}
	}
}

func TestValidateFields_AcceptsEmptyFields(t *testing.T) {
	collection := &Collection{name: "user", data: map[string]DataType{"age": TypeInt}}

	if err := validateFields(collection, map[string]any{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := validateFields(collection, nil); err != nil {
		t.Fatalf("expected no error for nil fields, got %v", err)
	}
}

func TestValidateFields_AcceptsNullOnOptionalField(t *testing.T) {
	collection := &Collection{
		name:     "user",
		data:     map[string]DataType{"nickname": TypeText},
		optional: map[string]bool{"nickname": true},
	}

	if err := validateFields(collection, map[string]any{"nickname": nil}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateFields_RejectsNullOnRequiredField(t *testing.T) {
	collection := &Collection{name: "user", data: map[string]DataType{"name": TypeText}}

	err := validateFields(collection, map[string]any{"name": nil})
	if err == nil {
		t.Fatal("expected an error for null on a required field")
	}

	want := `field "name" is required and cannot be null`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestRecordProblems_ReportsMissingRequiredField(t *testing.T) {
	collection := &Collection{
		name: "user",
		data: map[string]DataType{"age": TypeInt, "name": TypeText},
	}

	problems, _ := recordProblems(collection, Entity{"name": "Matt"})

	want := []string{`field "age" is required for collection "user"`}
	if !reflect.DeepEqual(problems, want) {
		t.Fatalf("expected %v, got %v", want, problems)
	}
}

func TestRecordProblems_ReportsEveryProblemInOrder(t *testing.T) {
	collection := &Collection{
		name:       "user",
		data:       map[string]DataType{"id": TypeInt, "age": TypeInt, "name": TypeText, "city": TypeText},
		primaryKey: "id",
	}

	for range 20 {
		problems, bad := recordProblems(collection, Entity{"name": int64(1), "age": "old", "nope": true})

		want := []string{
			`record missing primary key "id" for collection "user"`,
			`field "age": expected int, got text`,
			`field "name": expected text, got int`,
			`field "nope" not found in schema for collection "user"`,
			`field "city" is required for collection "user"`,
		}
		if !reflect.DeepEqual(problems, want) {
			t.Fatalf("expected:\n%v\ngot:\n%v", want, problems)
		}

		for _, field := range []string{"id", "age", "name", "nope", "city"} {
			if !bad[field] {
				t.Fatalf("expected %q to be flagged, got %v", field, bad)
			}
		}
	}
}

func TestRecordProblems_ReportsOneProblemPerField(t *testing.T) {
	collection := &Collection{
		name:         "user",
		data:         map[string]DataType{"id": TypeInt, "name": TypeText},
		primaryKey:   "id",
		autoCounters: map[string]int64{"id": 1},
	}

	// A null auto id is "supplied", and would also be "required and cannot be
	// null"; only the first check reports it.
	problems, _ := recordProblems(collection, Entity{"id": nil, "name": nil})

	want := []string{
		`field "id" is auto-increment and must not be supplied for collection "user"`,
		`field "name" is required and cannot be null`,
	}
	if !reflect.DeepEqual(problems, want) {
		t.Fatalf("expected %v, got %v", want, problems)
	}
}

func TestRecordProblems_NullManualPrimaryKeyReportedOnce(t *testing.T) {
	collection := &Collection{
		name:       "user",
		data:       map[string]DataType{"id": TypeInt},
		primaryKey: "id",
	}

	problems, _ := recordProblems(collection, Entity{"id": nil})

	want := []string{`record missing primary key "id" for collection "user"`}
	if !reflect.DeepEqual(problems, want) {
		t.Fatalf("expected %v, got %v", want, problems)
	}
}

func TestRecordProblems_NoProblemsSkipsOptionalAndAuto(t *testing.T) {
	collection := &Collection{
		name:         "user",
		data:         map[string]DataType{"id": TypeInt, "nickname": TypeText},
		primaryKey:   "id",
		autoCounters: map[string]int64{"id": 1},
		optional:     map[string]bool{"nickname": true},
	}

	if problems, _ := recordProblems(collection, Entity{}); len(problems) != 0 {
		t.Fatalf("expected no problems, got %v", problems)
	}
}
