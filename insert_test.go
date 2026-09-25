package main

import "testing"

func TestDatabaseRun_InsertsARecordWithPrimaryKey(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.run(DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
			"age":  TypeInt,
		},
		PrimaryKey: "name",
	})
	if err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	o := InsertOperation{
		Collection: "user",
		Record:     Entity{"name": "Matt", "age": 39},
	}

	result, err := d.run(o)
	if err != nil {
		t.Fatalf("Failed to insert record, err: %v", err)
	}

	insertResult, ok := result.(InsertResult)
	if !ok {
		t.Fatal("Record was not returned")
	}

	if insertResult.Record["name"] != "Matt" {
		t.Fatalf("expected returned record name %q, got %q", "Matt", insertResult.Record["name"])
	}

	collection := d.collections["user"]

	id, ok := collection.primaryIndex["Matt"]
	if !ok {
		t.Fatal("expected primary key \"Matt\" to be indexed")
	}

	stored, ok := collection.records[id]
	if !ok {
		t.Fatalf("expected a record stored at id %d", id)
	}

	if stored["age"] != 39 {
		t.Fatalf("expected stored record age %v, got %v", 39, stored["age"])
	}
}

func TestDatabaseRun_InsertRejectsMissingPrimaryKeyField(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.run(DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
			"age":  TypeInt,
		},
		PrimaryKey: "name",
	})
	if err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	o := InsertOperation{
		Collection: "user",
		Record:     Entity{"age": 39},
	}

	_, err = d.run(o)
	if err == nil {
		t.Fatal("expected an error for a record missing the primary key field")
	}
}

func TestDatabaseRun_InsertRejectsDuplicatePrimaryKey(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.run(DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
		},
		PrimaryKey: "name",
	})
	if err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	o := InsertOperation{
		Collection: "user",
		Record:     Entity{"name": "Matt"},
	}

	if _, err := d.run(o); err != nil {
		t.Fatalf("Failed to insert first record, err: %v", err)
	}

	if _, err := d.run(o); err == nil {
		t.Fatal("expected an error for a duplicate primary key")
	}
}

func TestDatabaseRun_InsertsARecordWithNoPrimaryKey(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.run(DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"name": TypeText,
		},
	})
	if err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	o := InsertOperation{
		Collection: "user",
		Record:     Entity{"name": "Matt"},
	}

	result, err := d.run(o)
	if err != nil {
		t.Fatalf("Failed to insert record, err: %v", err)
	}

	insertResult, ok := result.(InsertResult)
	if !ok {
		t.Fatal("Record was not returned")
	}

	if insertResult.Record["name"] != "Matt" {
		t.Fatalf("expected returned record name %q, got %q", "Matt", insertResult.Record["name"])
	}

	collection := d.collections["user"]

	if len(collection.primaryIndex) != 0 {
		t.Fatalf("expected no primary index entries, got %d", len(collection.primaryIndex))
	}

	if len(collection.records) != 1 {
		t.Fatalf("expected 1 stored record, got %d", len(collection.records))
	}
}

func TestDatabaseRun_InsertRejectsUnknownCollection(t *testing.T) {
	d := NewDatabase("test")

	o := InsertOperation{
		Collection: "user",
		Record:     Entity{"name": "Matt"},
	}

	if _, err := d.run(o); err == nil {
		t.Fatal("expected an error for inserting into a collection that doesn't exist")
	}
}
