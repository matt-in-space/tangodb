package core

import "testing"

func setupUserCollectionForMerge(t *testing.T) *Database {
	t.Helper()

	d := NewDatabase("test")

	if _, err := d.Run(DefineCollectionOperation{
		Name: "user",
		Data: map[string]DataType{
			"id":   TypeInt,
			"name": TypeText,
			"age":  TypeInt,
		},
		PrimaryKey: "id",
	}); err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	if _, err := d.Run(InsertOperation{Collection: "user", Record: Entity{"id": int64(1), "name": "Sam", "age": int64(40)}}); err != nil {
		t.Fatalf("Failed to insert, err: %v", err)
	}

	if _, err := d.Run(InsertOperation{Collection: "user", Record: Entity{"id": int64(2), "name": "Pat", "age": int64(40)}}); err != nil {
		t.Fatalf("Failed to insert, err: %v", err)
	}

	return d
}

func TestDatabaseRun_MergeUpdatesMatchingRecordsOnly(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	result, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(1)},
		Payload:    Entity{"name": "Matt"},
	})
	if err != nil {
		t.Fatalf("Failed to merge, err: %v", err)
	}

	if result.(MergeResult).Count != 1 {
		t.Fatalf("expected 1 updated, got %d", result.(MergeResult).Count)
	}

	collection := d.collections["user"]

	id := collection.primaryIndex[int64(1)]
	if collection.records[id]["name"] != "Matt" {
		t.Fatalf("expected record 1's name to be updated to %q, got %v", "Matt", collection.records[id]["name"])
	}

	otherID := collection.primaryIndex[int64(2)]
	if collection.records[otherID]["name"] != "Pat" {
		t.Fatalf("expected non-matching record 2 to be untouched, got name %v", collection.records[otherID]["name"])
	}
}

func TestDatabaseRun_MergeWithEmptyFilterUpdatesEverything(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	result, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{},
		Payload:    Entity{"age": int64(41)},
	})
	if err != nil {
		t.Fatalf("Failed to merge, err: %v", err)
	}

	if result.(MergeResult).Count != 2 {
		t.Fatalf("expected 2 updated, got %d", result.(MergeResult).Count)
	}

	collection := d.collections["user"]

	for _, record := range collection.records {
		if record["age"] != int64(41) {
			t.Fatalf("expected every record's age to be updated to 41, got %v", record["age"])
		}
	}
}

func TestDatabaseRun_MergeIsAPartialUpdate(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	if _, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(1)},
		Payload:    Entity{"name": "Matt"},
	}); err != nil {
		t.Fatalf("Failed to merge, err: %v", err)
	}

	collection := d.collections["user"]
	id := collection.primaryIndex[int64(1)]
	record := collection.records[id]

	if record["age"] != int64(40) {
		t.Fatalf("expected age to remain untouched at 40, got %v", record["age"])
	}

	if record["id"] != int64(1) {
		t.Fatalf("expected id to remain untouched at 1, got %v", record["id"])
	}
}

func TestDatabaseRun_MergeMatchingNothingIsANoOp(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	result, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(999)},
		Payload:    Entity{"name": "Matt"},
	})
	if err != nil {
		t.Fatalf("expected no error for a merge matching nothing, got: %v", err)
	}

	if result.(MergeResult).Count != 0 {
		t.Fatalf("expected 0 updated, got %d", result.(MergeResult).Count)
	}

	collection := d.collections["user"]

	if len(collection.records) != 2 {
		t.Fatalf("expected no records created, still expected 2, got %d", len(collection.records))
	}
}

func TestDatabaseRun_MergeWithoutProjectionReturnsOnlyCount(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	result, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(1)},
		Payload:    Entity{"name": "Matt"},
	})
	if err != nil {
		t.Fatalf("Failed to merge, err: %v", err)
	}

	mergeResult := result.(MergeResult)

	if mergeResult.Records != nil {
		t.Fatalf("expected no records to be returned without a projection, got %v", mergeResult.Records)
	}

	if mergeResult.Projection != nil {
		t.Fatalf("expected no projection to be set, got %v", mergeResult.Projection)
	}
}

func TestDatabaseRun_MergeWithProjectionReturnsPostMergeState(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	result, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(1)},
		Payload:    Entity{"name": "Matt"},
		Projection: []string{"id", "name"},
	})
	if err != nil {
		t.Fatalf("Failed to merge, err: %v", err)
	}

	mergeResult := result.(MergeResult)

	if mergeResult.Count != 1 {
		t.Fatalf("expected 1 updated, got %d", mergeResult.Count)
	}

	if len(mergeResult.Records) != 1 {
		t.Fatalf("expected 1 returned record, got %d", len(mergeResult.Records))
	}

	if mergeResult.Records[0]["name"] != "Matt" {
		t.Fatalf("expected the returned record to reflect the post-merge name %q, got %v", "Matt", mergeResult.Records[0]["name"])
	}
}

func TestDatabaseRun_MergeRejectsPrimaryKeyInPayload(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	if _, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{"name": "Sam"},
		Payload:    Entity{"id": int64(5)},
	}); err == nil {
		t.Fatal("expected an error for a payload that includes the primary key field")
	}

	collection := d.collections["user"]

	if _, exists := collection.primaryIndex[int64(5)]; exists {
		t.Fatal("expected no mutation to have happened after the rejected merge")
	}

	if _, exists := collection.primaryIndex[int64(1)]; !exists {
		t.Fatal("expected the original record to still exist unmodified")
	}
}

func TestDatabaseRun_MergeAllowsFilteringOnPrimaryKey(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	result, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{"id": int64(1)},
		Payload:    Entity{"name": "Matt"},
	})
	if err != nil {
		t.Fatalf("expected filtering on the primary key to be allowed, got err: %v", err)
	}

	if result.(MergeResult).Count != 1 {
		t.Fatalf("expected 1 updated, got %d", result.(MergeResult).Count)
	}
}

func TestDatabaseRun_MergeRejectsUnknownCollection(t *testing.T) {
	d := NewDatabase("test")

	if _, err := d.Run(MergeOperation{Collection: "ghost", Filter: map[string]any{}, Payload: Entity{"name": "Matt"}}); err == nil {
		t.Fatal("expected an error for merging into a collection that doesn't exist")
	}
}

func TestDatabaseRun_MergeRejectsUnknownFilterField(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	if _, err := d.Run(MergeOperation{Collection: "user", Filter: map[string]any{"nope": int64(1)}, Payload: Entity{"name": "Matt"}}); err == nil {
		t.Fatal("expected an error for an unknown filter field")
	}
}

func TestDatabaseRun_MergeRejectsUnknownProjectionField(t *testing.T) {
	d := setupUserCollectionForMerge(t)

	if _, err := d.Run(MergeOperation{
		Collection: "user",
		Filter:     map[string]any{},
		Payload:    Entity{"name": "Matt"},
		Projection: []string{"nope"},
	}); err == nil {
		t.Fatal("expected an error for an unknown projection field")
	}
}

func TestMergeResult_StringWithCountOnly(t *testing.T) {
	r := MergeResult{Count: 3}

	if got := r.String(); got != "3 updated" {
		t.Fatalf("expected %q, got %q", "3 updated", got)
	}
}

func TestMergeResult_StringWithProjection(t *testing.T) {
	r := MergeResult{
		Count:      1,
		Projection: []string{"id", "name"},
		Records:    []Entity{{"id": int64(1), "name": "Matt"}},
	}

	want := "id  name\n1   Matt"

	if got := r.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestMergeResult_StringWithProjectionButNoMatches(t *testing.T) {
	r := MergeResult{Count: 0, Projection: []string{"id"}, Records: []Entity{}}

	if got := r.String(); got != "no records found" {
		t.Fatalf("expected %q, got %q", "no records found", got)
	}
}
