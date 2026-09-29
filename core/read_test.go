package core

import "testing"

func setupUserCollectionWithRecords(t *testing.T) *Database {
	t.Helper()

	d := NewDatabase("test")

	if _, err := d.Run(DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"id":   TypeInt,
			"name": TypeText,
		},
		PrimaryKey: "id",
	}); err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	if _, err := d.Run(InsertOperation{Collection: "user", Record: Entity{"id": int64(1), "name": "Matt"}}); err != nil {
		t.Fatalf("Failed to insert, err: %v", err)
	}

	if _, err := d.Run(InsertOperation{Collection: "user", Record: Entity{"id": int64(2), "name": "Sam"}}); err != nil {
		t.Fatalf("Failed to insert, err: %v", err)
	}

	return d
}

func TestDatabaseRun_ReadReturnsAllMatchesWithEmptyFilter(t *testing.T) {
	d := setupUserCollectionWithRecords(t)

	result, err := d.Run(ReadOperation{Collection: "user", Filter: map[string]any{}, Projection: []string{"id", "name"}})
	if err != nil {
		t.Fatalf("Failed to read, err: %v", err)
	}

	readResult, ok := result.(ReadResult)
	if !ok {
		t.Fatal("Records were not returned")
	}

	if len(readResult.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(readResult.Records))
	}
}

func TestDatabaseRun_ReadFiltersByField(t *testing.T) {
	d := setupUserCollectionWithRecords(t)

	result, err := d.Run(ReadOperation{Collection: "user", Filter: map[string]any{"name": "Sam"}, Projection: []string{"id", "name"}})
	if err != nil {
		t.Fatalf("Failed to read, err: %v", err)
	}

	readResult := result.(ReadResult)

	if len(readResult.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(readResult.Records))
	}

	if readResult.Records[0]["id"] != int64(2) {
		t.Fatalf("expected id 2, got %v", readResult.Records[0]["id"])
	}
}

func TestDatabaseRun_ReadReturnsNoRecordsWhenNothingMatches(t *testing.T) {
	d := setupUserCollectionWithRecords(t)

	result, err := d.Run(ReadOperation{Collection: "user", Filter: map[string]any{"id": int64(99)}, Projection: []string{"id"}})
	if err != nil {
		t.Fatalf("Failed to read, err: %v", err)
	}

	readResult := result.(ReadResult)

	if len(readResult.Records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(readResult.Records))
	}
}

func TestDatabaseRun_ReadWithNilProjectionReturnsCountOnly(t *testing.T) {
	d := setupUserCollectionWithRecords(t)

	result, err := d.Run(ReadOperation{Collection: "user", Filter: map[string]any{}, Projection: nil})
	if err != nil {
		t.Fatalf("Failed to read, err: %v", err)
	}

	readResult := result.(ReadResult)

	if readResult.Count != 2 {
		t.Fatalf("expected count 2, got %d", readResult.Count)
	}

	if readResult.Records != nil {
		t.Fatalf("expected no records to be returned without a projection, got %v", readResult.Records)
	}

	if readResult.Projection != nil {
		t.Fatalf("expected no projection to be set, got %v", readResult.Projection)
	}
}

func TestDatabaseRun_ReadWithWildcardProjectionReturnsAllSchemaFields(t *testing.T) {
	d := setupUserCollectionWithRecords(t)

	result, err := d.Run(ReadOperation{Collection: "user", Filter: map[string]any{}, Projection: []string{"*"}})
	if err != nil {
		t.Fatalf("Failed to read, err: %v", err)
	}

	readResult := result.(ReadResult)

	want := map[string]bool{"id": true, "name": true}

	if len(readResult.Projection) != len(want) {
		t.Fatalf("expected %d projected fields, got %v", len(want), readResult.Projection)
	}

	for _, field := range readResult.Projection {
		if !want[field] {
			t.Fatalf("unexpected field %q in wildcard projection %v", field, readResult.Projection)
		}
	}

	if len(readResult.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(readResult.Records))
	}
}

func TestDatabaseRun_ReadRejectsUnknownFilterField(t *testing.T) {
	d := setupUserCollectionWithRecords(t)

	if _, err := d.Run(ReadOperation{Collection: "user", Filter: map[string]any{"nope": int64(1)}, Projection: []string{"id"}}); err == nil {
		t.Fatal("expected an error for an unknown filter field")
	}
}

func TestDatabaseRun_ReadRejectsUnknownProjectionField(t *testing.T) {
	d := setupUserCollectionWithRecords(t)

	if _, err := d.Run(ReadOperation{Collection: "user", Filter: map[string]any{}, Projection: []string{"nope"}}); err == nil {
		t.Fatal("expected an error for an unknown projection field")
	}
}

func TestDatabaseRun_ReadRejectsUnknownCollection(t *testing.T) {
	d := NewDatabase("test")

	if _, err := d.Run(ReadOperation{Collection: "ghost", Filter: map[string]any{}, Projection: []string{"id"}}); err == nil {
		t.Fatal("expected an error for an unknown collection")
	}
}

func TestReadResult_StringRendersATable(t *testing.T) {
	r := ReadResult{
		Projection: []string{"id", "name"},
		Records: []Entity{
			{"id": int64(1), "name": "Matt"},
			{"id": int64(2), "name": "Sam"},
		},
	}

	want := "id  name\n1   Matt\n2   Sam"

	if got := r.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestReadResult_StringWithNoRecords(t *testing.T) {
	r := ReadResult{Projection: []string{"id"}, Records: nil}

	if got := r.String(); got != "no records found" {
		t.Fatalf("expected \"no records found\", got %q", got)
	}
}
