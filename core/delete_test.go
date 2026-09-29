package core

import "testing"

func setupUserCollectionForDelete(t *testing.T) *Database {
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

	if _, err := d.Run(InsertOperation{Collection: "user", Records: []Entity{Entity{"id": int64(1), "name": "Matt"}}}); err != nil {
		t.Fatalf("Failed to insert, err: %v", err)
	}

	if _, err := d.Run(InsertOperation{Collection: "user", Records: []Entity{Entity{"id": int64(2), "name": "Sam"}}}); err != nil {
		t.Fatalf("Failed to insert, err: %v", err)
	}

	return d
}

func TestDatabaseRun_DeleteRemovesMatchingRecordsOnly(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	result, err := d.Run(DeleteOperation{Collection: "user", Filter: map[string]any{"name": "Matt"}})
	if err != nil {
		t.Fatalf("Failed to delete, err: %v", err)
	}

	deleteResult := result.(DeleteResult)

	if deleteResult.Count != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleteResult.Count)
	}

	collection := d.collections["user"]

	if len(collection.records) != 1 {
		t.Fatalf("expected 1 record remaining, got %d", len(collection.records))
	}

	if _, exists := collection.primaryIndex[int64(1)]; exists {
		t.Fatal("expected the deleted record's primary index entry to be removed")
	}

	if _, exists := collection.primaryIndex[int64(2)]; !exists {
		t.Fatal("expected the non-matching record's primary index entry to remain")
	}
}

func TestDatabaseRun_DeleteWithEmptyFilterRemovesEverything(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	result, err := d.Run(DeleteOperation{Collection: "user", Filter: map[string]any{}})
	if err != nil {
		t.Fatalf("Failed to delete, err: %v", err)
	}

	if result.(DeleteResult).Count != 2 {
		t.Fatalf("expected 2 deleted, got %d", result.(DeleteResult).Count)
	}

	collection := d.collections["user"]

	if len(collection.records) != 0 {
		t.Fatalf("expected 0 records remaining, got %d", len(collection.records))
	}

	if len(collection.primaryIndex) != 0 {
		t.Fatalf("expected 0 primary index entries remaining, got %d", len(collection.primaryIndex))
	}
}

func TestDatabaseRun_DeleteMatchingNothingIsANoOp(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	result, err := d.Run(DeleteOperation{Collection: "user", Filter: map[string]any{"id": int64(999)}})
	if err != nil {
		t.Fatalf("expected no error for a delete matching nothing, got: %v", err)
	}

	if result.(DeleteResult).Count != 0 {
		t.Fatalf("expected 0 deleted, got %d", result.(DeleteResult).Count)
	}

	collection := d.collections["user"]

	if len(collection.records) != 2 {
		t.Fatalf("expected both records to remain, got %d", len(collection.records))
	}
}

func TestDatabaseRun_DeleteWithoutProjectionReturnsOnlyCount(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	result, err := d.Run(DeleteOperation{Collection: "user", Filter: map[string]any{"id": int64(1)}})
	if err != nil {
		t.Fatalf("Failed to delete, err: %v", err)
	}

	deleteResult := result.(DeleteResult)

	if deleteResult.Count != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleteResult.Count)
	}

	if deleteResult.Records != nil {
		t.Fatalf("expected no records to be returned without a projection, got %v", deleteResult.Records)
	}

	if deleteResult.Projection != nil {
		t.Fatalf("expected no projection to be set, got %v", deleteResult.Projection)
	}
}

func TestDatabaseRun_DeleteWithProjectionReturnsProjectedRecords(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	result, err := d.Run(DeleteOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(1)},
		Projection: []string{"name"},
	})
	if err != nil {
		t.Fatalf("Failed to delete, err: %v", err)
	}

	deleteResult := result.(DeleteResult)

	if deleteResult.Count != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleteResult.Count)
	}

	if len(deleteResult.Records) != 1 {
		t.Fatalf("expected 1 returned record, got %d", len(deleteResult.Records))
	}

	if deleteResult.Records[0]["name"] != "Matt" {
		t.Fatalf("expected returned record name %q, got %v", "Matt", deleteResult.Records[0]["name"])
	}

	// Projection narrows what String() displays (matching ReadResult's
	// existing behavior), not the shape of the stored record itself —
	// Records always holds the full entity.
	if got := result.(DeleteResult).String(); got != "name\nMatt" {
		t.Fatalf("expected the rendered table to show only the projected field, got: %q", got)
	}
}

func TestDatabaseRun_DeleteDoesNotReuseIDs(t *testing.T) {
	d := NewDatabase("test")

	if _, err := d.Run(DefineCollectionOperation{
		Name:       "user",
		Data:       map[string]DataType{"id": TypeInt},
		PrimaryKey: "id",
		AutoFields: map[string]bool{"id": true},
	}); err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := d.Run(InsertOperation{Collection: "user", Records: []Entity{Entity{}}, Projection: []string{"id"}}); err != nil {
			t.Fatalf("Failed to insert, err: %v", err)
		}
	}

	if _, err := d.Run(DeleteOperation{Collection: "user", Filter: map[string]any{"id": int64(2)}}); err != nil {
		t.Fatalf("Failed to delete, err: %v", err)
	}

	result, err := d.Run(InsertOperation{Collection: "user", Records: []Entity{Entity{}}, Projection: []string{"id"}})
	if err != nil {
		t.Fatalf("Failed to insert after delete, err: %v", err)
	}

	newID := result.(InsertResult).Records[0]["id"]

	if newID != int64(4) {
		t.Fatalf("expected the next auto-increment id to be 4 (never reusing deleted id 2), got %v", newID)
	}
}

func TestDatabaseRun_DeleteRejectsUnknownCollection(t *testing.T) {
	d := NewDatabase("test")

	if _, err := d.Run(DeleteOperation{Collection: "ghost", Filter: map[string]any{}}); err == nil {
		t.Fatal("expected an error for deleting from a collection that doesn't exist")
	}
}

func TestDatabaseRun_DeleteRejectsUnknownFilterField(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	if _, err := d.Run(DeleteOperation{Collection: "user", Filter: map[string]any{"nope": int64(1)}}); err == nil {
		t.Fatal("expected an error for an unknown filter field")
	}
}

func TestDatabaseRun_DeleteRejectsUnknownProjectionField(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	if _, err := d.Run(DeleteOperation{Collection: "user", Filter: map[string]any{}, Projection: []string{"nope"}}); err == nil {
		t.Fatal("expected an error for an unknown projection field")
	}
}

func TestDeleteResult_StringWithCountOnly(t *testing.T) {
	r := DeleteResult{Count: 3}

	if got := r.String(); got != "3" {
		t.Fatalf("expected %q, got %q", "3", got)
	}
}

func TestDatabaseRun_DeleteWithWildcardProjectionReturnsAllFields(t *testing.T) {
	d := setupUserCollectionForDelete(t)

	result, err := d.Run(DeleteOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(1)},
		Projection: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Failed to delete, err: %v", err)
	}

	deleteResult := result.(DeleteResult)

	if len(deleteResult.Records) != 1 {
		t.Fatalf("expected 1 returned record, got %d", len(deleteResult.Records))
	}

	want := map[string]bool{"id": true, "name": true}

	if len(deleteResult.Projection) != len(want) {
		t.Fatalf("expected %d projected fields, got %v", len(want), deleteResult.Projection)
	}

	for _, field := range deleteResult.Projection {
		if !want[field] {
			t.Fatalf("unexpected field %q in wildcard projection %v", field, deleteResult.Projection)
		}
	}
}

func TestDeleteResult_StringWithProjection(t *testing.T) {
	r := DeleteResult{
		Count:      1,
		Projection: []string{"id", "name"},
		Records:    []Entity{{"id": int64(1), "name": "Matt"}},
	}

	want := "id  name\n1   Matt"

	if got := r.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestDeleteResult_StringWithProjectionButNoMatches(t *testing.T) {
	r := DeleteResult{Count: 0, Projection: []string{"id"}, Records: []Entity{}}

	if got := r.String(); got != "no records found" {
		t.Fatalf("expected %q, got %q", "no records found", got)
	}
}
