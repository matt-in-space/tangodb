package core

import "testing"

func TestDatabaseRun_InsertsARecordWithPrimaryKey(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
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
		Records:    []Entity{Entity{"name": "Matt", "age": int64(39)}},
		Projection: []string{"*"},
	}

	result, err := d.Run(o)
	if err != nil {
		t.Fatalf("Failed to insert record, err: %v", err)
	}

	insertResult, ok := result.(InsertResult)
	if !ok {
		t.Fatal("Record was not returned")
	}

	if insertResult.Records[0]["name"] != "Matt" {
		t.Fatalf("expected returned record name %q, got %q", "Matt", insertResult.Records[0]["name"])
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

	if stored["age"] != int64(39) {
		t.Fatalf("expected stored record age %v, got %v", 39, stored["age"])
	}
}

func TestDatabaseRun_InsertRejectsMissingPrimaryKeyField(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
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
		Records:    []Entity{Entity{"age": int64(39)}},
	}

	_, err = d.Run(o)
	if err == nil {
		t.Fatal("expected an error for a record missing the primary key field")
	}

	want := `record missing primary key "name" for collection "user"`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestDatabaseRun_InsertRejectsDuplicatePrimaryKey(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
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
		Records:    []Entity{Entity{"name": "Matt"}},
	}

	if _, err := d.Run(o); err != nil {
		t.Fatalf("Failed to insert first record, err: %v", err)
	}

	if _, err := d.Run(o); err == nil {
		t.Fatal("expected an error for a duplicate primary key")
	}
}

func TestDatabaseRun_InsertsARecordWithNoPrimaryKey(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
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
		Records:    []Entity{Entity{"name": "Matt"}},
		Projection: []string{"*"},
	}

	result, err := d.Run(o)
	if err != nil {
		t.Fatalf("Failed to insert record, err: %v", err)
	}

	insertResult, ok := result.(InsertResult)
	if !ok {
		t.Fatal("Record was not returned")
	}

	if insertResult.Records[0]["name"] != "Matt" {
		t.Fatalf("expected returned record name %q, got %q", "Matt", insertResult.Records[0]["name"])
	}

	collection := d.collections["user"]

	if len(collection.primaryIndex) != 0 {
		t.Fatalf("expected no primary index entries, got %d", len(collection.primaryIndex))
	}

	if len(collection.records) != 1 {
		t.Fatalf("expected 1 stored record, got %d", len(collection.records))
	}
}

func TestDatabaseRun_InsertAssignsSequentialAutoIncrementIDs(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
		Name:       "user",
		Data:       map[string]DataType{"id": TypeInt, "name": TypeText},
		PrimaryKey: "id",
		AutoFields: map[string]bool{"id": true},
	})
	if err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	first, err := d.Run(InsertOperation{Collection: "user", Records: []Entity{Entity{"name": "Matt"}}, Projection: []string{"id"}})
	if err != nil {
		t.Fatalf("Failed to insert first record, err: %v", err)
	}

	second, err := d.Run(InsertOperation{Collection: "user", Records: []Entity{Entity{"name": "Sam"}}, Projection: []string{"id"}})
	if err != nil {
		t.Fatalf("Failed to insert second record, err: %v", err)
	}

	firstID := first.(InsertResult).Records[0]["id"]
	secondID := second.(InsertResult).Records[0]["id"]

	if firstID != int64(1) {
		t.Fatalf("expected first id 1, got %v", firstID)
	}

	if secondID != int64(2) {
		t.Fatalf("expected second id 2, got %v", secondID)
	}
}

func TestDatabaseRun_InsertRejectsManualValueOnAutoIncrementField(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
		Name:       "user",
		Data:       map[string]DataType{"id": TypeInt},
		PrimaryKey: "id",
		AutoFields: map[string]bool{"id": true},
	})
	if err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	if _, err := d.Run(InsertOperation{Collection: "user", Records: []Entity{Entity{"id": int64(5)}}}); err == nil {
		t.Fatal("expected an error for manually supplying an auto-increment field")
	}
}

func TestDatabaseRun_InsertRejectsUnknownCollection(t *testing.T) {
	d := NewDatabase("test")

	o := InsertOperation{
		Collection: "user",
		Records:    []Entity{Entity{"name": "Matt"}},
	}

	if _, err := d.Run(o); err == nil {
		t.Fatal("expected an error for inserting into a collection that doesn't exist")
	}
}

func TestInsert_WithoutProjectionReturnsCount(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`)

	result := runStatements(t, d, `>> user {id: 1 name: "Matt"};`).(InsertResult)

	if result.String() != "1" {
		t.Fatalf("expected %q, got %q", "1", result.String())
	}

	if result.Records != nil {
		t.Fatalf("expected no records without a projection, got %v", result.Records)
	}

	expectCount(t, d, `<< user;`, "1")
}

func TestInsert_ProjectionReturnsGeneratedID(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text };`)

	got := runStatements(t, d, `>> user {name: "Matt"} => {id};`).(InsertResult).String()

	want := "id\n1"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestInsert_WildcardProjectionShowsStoredRecord(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text nickname: text @optional };`)

	got := runStatements(t, d, `>> user {id: 1 name: "Matt"} => {*};`).(InsertResult).String()

	want := "id  name  nickname\n1   Matt  null"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestInsert_UnknownProjectionFieldStoresNothing(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id @auto name: text };`)

	got := runExpectingError(t, d, `>> user {name: "Matt"} => {nope};`)

	want := `field "nope" not found in schema for collection "user"`
	if got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	expectCount(t, d, `<< user;`, "0")

	// The failed insert must not have used up an auto-increment value either.
	got = runStatements(t, d, `>> user {name: "Matt"} => {id};`).(InsertResult).String()
	if got != "id\n1" {
		t.Fatalf("expected id 1, got:\n%s", got)
	}
}
