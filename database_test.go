package main

import "testing"

func TestDatabaseRun_DefinesACollection(t *testing.T) {
	o := DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
			"age":  TypeInt,
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

	if defineResult.Collection.name != o.Name {
		t.Fatalf("expected collection name %q, got %q", o.Name, defineResult.Collection.name)
	}

	for field, dataType := range o.Data {
		if defineResult.Collection.data[field] != dataType {
			t.Fatalf("expected field %q to have type %v, got %v", field, dataType, defineResult.Collection.data[field])
		}
	}
}
