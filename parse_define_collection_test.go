package main

import "testing"

func TestParseDefineCollection_ParsesFieldsAndPrimaryKey(t *testing.T) {
	input := `
	user {
	  id: int @id
	  name: text
	}
	`

	o, err := ParseDefineCollection(input)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Name != "user" {
		t.Fatalf("expected name %q, got %q", "user", o.Name)
	}

	if o.PrimaryKey != "id" {
		t.Fatalf("expected primary key %q, got %q", "id", o.PrimaryKey)
	}

	want := map[string]DataType{
		"id":   TypeInt,
		"name": TypeText,
	}

	for field, dataType := range want {
		if o.Data[field] != dataType {
			t.Fatalf("expected field %q to have type %v, got %v", field, dataType, o.Data[field])
		}
	}

	if len(o.Data) != len(want) {
		t.Fatalf("expected %d fields, got %d", len(want), len(o.Data))
	}
}

func TestParseDefineCollection_AllowsNoPrimaryKey(t *testing.T) {
	o, err := ParseDefineCollection(`user { name: text }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.PrimaryKey != "" {
		t.Fatalf("expected no primary key, got %q", o.PrimaryKey)
	}
}

func TestParseDefineCollection_AllowsEmptyCollection(t *testing.T) {
	o, err := ParseDefineCollection(`user {}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Data) != 0 {
		t.Fatalf("expected no fields, got %d", len(o.Data))
	}
}

func TestParseDefineCollection_IsCaseInsensitiveAboutTypes(t *testing.T) {
	o, err := ParseDefineCollection(`user { age: INT }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Data["age"] != TypeInt {
		t.Fatalf("expected age to be TypeInt, got %v", o.Data["age"])
	}
}

func TestParseDefineCollection_RejectsUnknownType(t *testing.T) {
	if _, err := ParseDefineCollection(`user { age: number }`); err == nil {
		t.Fatal("expected an error for an unknown type")
	}
}

func TestParseDefineCollection_RejectsUnknownAnnotation(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int @unique }`); err == nil {
		t.Fatal("expected an error for an unknown annotation")
	}
}

func TestParseDefineCollection_RejectsMultiplePrimaryKeys(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int @id name: text @id }`); err == nil {
		t.Fatal("expected an error for multiple @id fields")
	}
}

func TestParseDefineCollection_RejectsMissingClosingBrace(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int`); err == nil {
		t.Fatal("expected an error for a missing closing brace")
	}
}

func TestParseDefineCollection_RejectsTrailingInput(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int } extra`); err == nil {
		t.Fatal("expected an error for unexpected trailing input")
	}
}

func TestParseDefineCollection_EndToEndThroughDatabase(t *testing.T) {
	o, err := ParseDefineCollection(`
	user {
	  id: int @id
	  name: text
	}
	`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	d := NewDatabase("test")

	result, err := d.run(o)
	if err != nil {
		t.Fatalf("Failed to run parsed operation, err: %v", err)
	}

	defineResult, ok := result.(DefineCollectionResult)
	if !ok {
		t.Fatal("Collection was not returned")
	}

	if defineResult.Collection.name != "user" {
		t.Fatalf("expected collection name %q, got %q", "user", defineResult.Collection.name)
	}
}
