package core

import "testing"

func TestDatabaseRun_DefinesACollection(t *testing.T) {
	o := DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
			"age":  TypeInt,
		},
		PrimaryKey: "name",
	}

	d := NewDatabase("test")

	result, err := d.Run(o)

	if err != nil {
		t.Fatalf("Failed to create table, err: %v", err)
	}

	defineResult, ok := result.(DefineCollectionResult)

	if !ok {
		t.Fatal("Collection was not returned")
	}

	if defineResult.Collection.name != o.Name {
		t.Fatalf("expected collection name %q, got %q", o.Name, defineResult.Collection.name)
	}

	if defineResult.Collection.primaryKey != o.PrimaryKey {
		t.Fatalf("expected primary key %q, got %q", o.PrimaryKey, defineResult.Collection.primaryKey)
	}

	for field, dataType := range o.Data {
		if defineResult.Collection.data[field] != dataType {
			t.Fatalf("expected field %q to have type %v, got %v", field, dataType, defineResult.Collection.data[field])
		}
	}
}

func TestDatabaseRun_DefineCollectionRejectsUnknownPrimaryKey(t *testing.T) {
	o := DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
		},
		PrimaryKey: "id",
	}

	d := NewDatabase("test")

	_, err := d.Run(o)

	if err == nil {
		t.Fatal("expected an error for a primary key not present in the schema")
	}
}

func TestDatabaseRun_DefineCollectionAllowsAutoWithoutPrimaryKey(t *testing.T) {
	o := DefineCollectionOperation{
		Name:       "ticket",
		Data:       map[string]DataType{"number": TypeInt},
		AutoFields: map[string]bool{"number": true},
	}

	d := NewDatabase("test")

	if _, err := d.Run(o); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDatabaseRun_DefineCollectionRejectsOptionalAutoField(t *testing.T) {
	o := DefineCollectionOperation{
		Name:       "ticket",
		Data:       map[string]DataType{"number": TypeInt},
		AutoFields: map[string]bool{"number": true},
		Optional:   map[string]bool{"number": true},
	}

	d := NewDatabase("test")

	_, err := d.Run(o)
	if err == nil {
		t.Fatal("expected an error for an optional auto field")
	}

	want := `@auto field "number" cannot be @optional`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestDatabaseRun_DefineCollectionRejectsUnknownAutoField(t *testing.T) {
	o := DefineCollectionOperation{
		Name:       "ticket",
		Data:       map[string]DataType{"number": TypeInt},
		AutoFields: map[string]bool{"nope": true},
	}

	d := NewDatabase("test")

	if _, err := d.Run(o); err == nil {
		t.Fatal("expected an error for an auto field missing from the schema")
	}
}

func TestDatabaseRun_DefineCollectionRejectsAutoOnNonIntField(t *testing.T) {
	o := DefineCollectionOperation{
		Name:       "user",
		Data:       map[string]DataType{"id": TypeText},
		PrimaryKey: "id",
		AutoFields: map[string]bool{"id": true},
	}

	d := NewDatabase("test")

	_, err := d.Run(o)
	if err == nil {
		t.Fatal("expected an error for @auto on a non-int field")
	}

	want := `@auto requires an int field (field "id")`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestDatabaseRun_DefineCollectionAllowsNoPrimaryKey(t *testing.T) {
	o := DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
		},
	}

	d := NewDatabase("test")

	result, err := d.Run(o)

	if err != nil {
		t.Fatalf("Failed to create table, err: %v", err)
	}

	defineResult, ok := result.(DefineCollectionResult)

	if !ok {
		t.Fatal("Collection was not returned")
	}

	if defineResult.Collection.primaryKey != "" {
		t.Fatalf("expected no primary key, got %q", defineResult.Collection.primaryKey)
	}
}

func TestDatabaseRun_DefineCollectionRejectsOptionalPrimaryKey(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
		Name:       "user",
		Data:       map[string]DataType{"id": TypeInt},
		PrimaryKey: "id",
		Optional:   map[string]bool{"id": true},
	})
	if err == nil {
		t.Fatal("expected an error for an optional primary key")
	}

	if _, exists := d.collections["user"]; exists {
		t.Fatal("expected no collection to be defined")
	}
}

func TestDatabaseRun_DefineCollectionChecksEmbeddedSchemas(t *testing.T) {
	cases := []DefineCollectionOperation{
		{Name: "user", Data: map[string]DataType{"address": TypeObject}},
		{Name: "user", Data: map[string]DataType{"name": TypeText}, Objects: map[string]*Schema{"name": {Data: map[string]DataType{}}}},
		{Name: "user", Data: map[string]DataType{"address": TypeObject}, Objects: map[string]*Schema{
			"address": {Data: map[string]DataType{"city": TypeText}, Optional: map[string]bool{"zip": true}},
		}},
		{Name: "user", Data: map[string]DataType{"address": TypeObject}, PrimaryKey: "address", Objects: map[string]*Schema{
			"address": {Data: map[string]DataType{"city": TypeText}},
		}},
	}

	for i, op := range cases {
		d := NewDatabase("test")
		if _, err := d.Run(op); err == nil {
			t.Fatalf("case %d: expected an error", i)
		}
	}
}
