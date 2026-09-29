package core

import "testing"

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

func TestValidateRequired_RejectsMissingRequiredField(t *testing.T) {
	collection := &Collection{
		name: "user",
		data: map[string]DataType{"age": TypeInt, "name": TypeText},
	}

	err := validateRequired(collection, Entity{"name": "Matt"})
	if err == nil {
		t.Fatal("expected an error for a missing required field")
	}

	want := `field "age" is required for collection "user"`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestValidateRequired_ReportsFirstMissingFieldInSortedOrder(t *testing.T) {
	collection := &Collection{
		name: "user",
		data: map[string]DataType{"a": TypeInt, "b": TypeInt, "c": TypeInt},
	}

	for range 20 {
		err := validateRequired(collection, Entity{"b": int64(1)})
		if err == nil || err.Error() != `field "a" is required for collection "user"` {
			t.Fatalf("expected an error for field \"a\", got %v", err)
		}
	}
}

func TestValidateRequired_SkipsOptionalAndAutoPrimaryKey(t *testing.T) {
	collection := &Collection{
		name:         "user",
		data:         map[string]DataType{"id": TypeInt, "nickname": TypeText},
		primaryKey:   "id",
		autoCounters: map[string]int64{"id": 1},
		optional:     map[string]bool{"nickname": true},
	}

	if err := validateRequired(collection, Entity{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
