package main

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

	result, err := d.run(o)

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

	_, err := d.run(o)

	if err == nil {
		t.Fatal("expected an error for a primary key not present in the schema")
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

	result, err := d.run(o)

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
