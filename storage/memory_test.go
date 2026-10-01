package storage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/matt-in-space/tangodb/storage/record"
)

type Entity = record.Entity

func newUserStore(t *testing.T) *MemoryStore {
	t.Helper()
	s, err := NewMemoryStore(Config{Collection: "user", PrimaryKey: "id", KeyType: IntKey, AutoFields: []string{"id"}})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	return s
}

func mustInsert(t *testing.T, s Store, records ...Entity) []RecordRef {
	t.Helper()
	refs, err := s.Insert(records)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	return refs
}

// sized builds a record that encodes to exactly n bytes, and checks it does.
func sized(t *testing.T, id int64, n int) Entity {
	t.Helper()

	base := Entity{"id": id, "t": ""}
	empty, err := record.Encode(base)
	if err != nil {
		t.Fatal(err)
	}

	// Growing the text by k bytes adds k bytes, plus one more once its length
	// varint needs a second byte (at 128 bytes).
	k := n - len(empty)
	if k >= 128 {
		k--
	}
	rec := Entity{"id": id, "t": strings.Repeat("x", k)}

	encoded, err := record.Encode(rec)
	if err != nil || len(encoded) != n {
		t.Fatalf("expected a %d-byte record, got %d (%v)", n, len(encoded), err)
	}
	return rec
}

func scanAll(t *testing.T, s Store) map[RecordRef]Entity {
	t.Helper()

	cur, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	defer cur.Close()

	seen := map[RecordRef]Entity{}
	for cur.Next() {
		if _, dup := seen[cur.Ref()]; dup {
			t.Fatalf("scan visited %v twice", cur.Ref())
		}
		seen[cur.Ref()] = cur.Record()
	}
	if err := cur.Err(); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	return seen
}

func TestMemoryStore_ReturnsIndependentCopies(t *testing.T) {
	s := newUserStore(t)
	mustInsert(t, s, Entity{"id": int64(1), "name": "Matt", "address": Entity{"city": "MSP"}})

	_, rec, _, _ := s.Lookup(int64(1))
	rec["name"] = "Sam"
	rec["address"].(Entity)["city"] = "STP"

	_, again, _, _ := s.Lookup(int64(1))
	if again["name"] != "Matt" || again["address"].(Entity)["city"] != "MSP" {
		t.Fatalf("expected the stored record to be unchanged, got %v", again)
	}
}

func TestMemoryStore_InsertIsAllOrNothing(t *testing.T) {
	s := newUserStore(t)

	_, err := s.Insert([]Entity{
		{"id": int64(1)},
		{"id": int64(2)},
		sized(t, 3, MaxRecordSize+1),
	})
	if err == nil || err.Error() != "storage: record 3 is 4077 bytes; the maximum is 4076" {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := scanAll(t, s); len(got) != 0 {
		t.Fatalf("expected nothing stored, got %d records", len(got))
	}
	if next, _ := s.NextAuto("id"); next != 1 {
		t.Fatalf("expected the counter unchanged at 1, got %d", next)
	}
}

func TestMemoryStore_DuplicateKeys(t *testing.T) {
	s := newUserStore(t)

	_, err := s.Insert([]Entity{{"id": int64(1)}, {"id": int64(1)}})
	if err == nil || err.Error() != `storage: record 2: duplicate primary key 1 in collection "user"` {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := scanAll(t, s); len(got) != 0 {
		t.Fatalf("expected nothing stored, got %d records", len(got))
	}

	mustInsert(t, s, Entity{"id": int64(1)})
	_, err = s.Insert([]Entity{{"id": int64(1)}})
	if err == nil || err.Error() != `storage: duplicate primary key 1 in collection "user"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryStore_RefsComeBackInInputOrder(t *testing.T) {
	s := newUserStore(t)
	refs := mustInsert(t, s, Entity{"id": int64(10)}, Entity{"id": int64(20)}, Entity{"id": int64(30)})

	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}

	for i, want := range []int64{10, 20, 30} {
		ref, _, _, _ := s.Lookup(want)
		if ref != refs[i] {
			t.Fatalf("expected ref %d to refer to id %d", i, want)
		}
	}
}

func TestMemoryStore_SizeLimit(t *testing.T) {
	s := newUserStore(t)

	mustInsert(t, s, sized(t, 1, MaxRecordSize))

	_, err := s.Insert([]Entity{sized(t, 2, MaxRecordSize+1)})
	if err == nil || err.Error() != "storage: record is 4077 bytes; the maximum is 4076" {
		t.Fatalf("unexpected error: %v", err)
	}

	if MaxRecordSize != 4076 {
		t.Fatalf("expected the limit to be 4076, got %d", MaxRecordSize)
	}
}

func TestMemoryStore_Lookup(t *testing.T) {
	s := newUserStore(t)
	refs := mustInsert(t, s, Entity{"id": int64(42), "name": "Matt"})

	ref, rec, found, err := s.Lookup(int64(42))
	if err != nil || !found || ref != refs[0] || rec["name"] != "Matt" {
		t.Fatalf("expected to find id 42, got %v %v %v %v", ref, rec, found, err)
	}

	if _, _, found, err := s.Lookup(int64(7)); found || err != nil {
		t.Fatalf("expected not found with no error, got %v %v", found, err)
	}

	_, _, _, err = s.Lookup("42")
	if err == nil || err.Error() != `storage: primary key "id" of collection "user" is int, got text` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryStore_TextKey(t *testing.T) {
	s, err := NewMemoryStore(Config{Collection: "ticket", PrimaryKey: "code", KeyType: TextKey})
	if err != nil {
		t.Fatal(err)
	}

	mustInsert(t, s, Entity{"code": "A"}, Entity{"code": "B"})

	if _, _, found, _ := s.Lookup("B"); !found {
		t.Fatal("expected to find code B")
	}

	_, err = s.Insert([]Entity{{"code": "A"}})
	if err == nil || err.Error() != `storage: duplicate primary key "A" in collection "ticket"` {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = s.Insert([]Entity{{"code": int64(1)}})
	if err == nil || err.Error() != `storage: primary key "code" of collection "ticket" is text, got int` {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = s.Insert([]Entity{{"name": "no key"}})
	if err == nil || err.Error() != `storage: missing primary key "code" for collection "ticket"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryStore_ScanVisitsEveryRecordOnce(t *testing.T) {
	s := newUserStore(t)
	for i := range int64(5) {
		mustInsert(t, s, Entity{"id": i + 1})
	}

	if got := scanAll(t, s); len(got) != 5 {
		t.Fatalf("expected 5 records, got %d", len(got))
	}
}

func TestMemoryStore_ScanSkipsRecordsDeletedMidScan(t *testing.T) {
	s := newUserStore(t)
	refs := mustInsert(t, s, Entity{"id": int64(1)}, Entity{"id": int64(2)}, Entity{"id": int64(3)})

	cur, _ := s.Scan()
	defer cur.Close()

	if !cur.Next() {
		t.Fatal("expected a first record")
	}
	first := cur.Ref()

	// Delete one the cursor hasn't reached yet.
	var victim RecordRef
	for _, ref := range refs {
		if ref != first {
			victim = ref
			break
		}
	}
	if err := s.Delete(victim); err != nil {
		t.Fatal(err)
	}

	visited := 1
	for cur.Next() {
		if cur.Ref() == victim {
			t.Fatal("expected the deleted record to be skipped")
		}
		visited++
	}

	if visited != 2 {
		t.Fatalf("expected 2 records visited, got %d", visited)
	}
}

func TestMemoryStore_ScanDoesNotSeeRecordsInsertedMidScan(t *testing.T) {
	s := newUserStore(t)
	mustInsert(t, s, Entity{"id": int64(1)})

	cur, _ := s.Scan()
	defer cur.Close()

	mustInsert(t, s, Entity{"id": int64(2)})

	visited := 0
	for cur.Next() {
		visited++
	}
	if visited != 1 {
		t.Fatalf("expected only the record that existed at the start, got %d", visited)
	}
}

func TestMemoryStore_Update(t *testing.T) {
	s := newUserStore(t)
	refs := mustInsert(t, s, Entity{"id": int64(1), "name": "Matt"})

	if err := s.Update(refs[0], Entity{"id": int64(1), "name": "Sam"}); err != nil {
		t.Fatal(err)
	}
	if _, rec, _, _ := s.Lookup(int64(1)); rec["name"] != "Sam" {
		t.Fatalf("expected the update, got %v", rec)
	}

	failures := map[string]Entity{
		`storage: update cannot change primary key "id" from 1 to 2 in collection "user"`: {"id": int64(2), "name": "X"},
		`storage: missing primary key "id" for collection "user"`:                         {"name": "X"},
		"storage: record is 4077 bytes; the maximum is 4076":                              sized(t, 1, MaxRecordSize+1),
	}

	for want, rec := range failures {
		if err := s.Update(refs[0], rec); err == nil || err.Error() != want {
			t.Fatalf("expected %q, got %v", want, err)
		}
		if _, stored, _, _ := s.Lookup(int64(1)); stored["name"] != "Sam" {
			t.Fatalf("expected the record unchanged after a failed update, got %v", stored)
		}
	}

	if err := s.Update(RecordRef{}, Entity{"id": int64(1)}); err == nil || !strings.HasPrefix(err.Error(), "storage: ") {
		t.Fatalf("expected an unknown-ref error, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := newUserStore(t)
	refs := mustInsert(t, s, Entity{"id": int64(1)}, Entity{"id": int64(2)})

	if err := s.Delete(refs[0]); err != nil {
		t.Fatal(err)
	}

	if _, _, found, _ := s.Lookup(int64(1)); found {
		t.Fatal("expected id 1 to be gone")
	}
	if got := scanAll(t, s); len(got) != 1 {
		t.Fatalf("expected 1 record left, got %d", len(got))
	}

	// The key is free again after its record is deleted.
	mustInsert(t, s, Entity{"id": int64(1)})

	err := s.Delete(refs[0])
	if err == nil || err.Error() != `storage: no record for this ref in collection "user"` {
		t.Fatalf("expected an unknown-ref error, got %v", err)
	}
}

func TestMemoryStore_AutoCounters(t *testing.T) {
	s := newUserStore(t)

	if next, _ := s.NextAuto("id"); next != 1 {
		t.Fatalf("expected the counter to start at 1, got %d", next)
	}

	mustInsert(t, s, Entity{"id": int64(1)}, Entity{"id": int64(2)})
	if next, _ := s.NextAuto("id"); next != 3 {
		t.Fatalf("expected 3 after inserting 1 and 2, got %d", next)
	}

	if _, err := s.Insert([]Entity{{"id": int64(3)}, sized(t, 4, MaxRecordSize+1)}); err == nil {
		t.Fatal("expected the insert to fail")
	}
	if next, _ := s.NextAuto("id"); next != 3 {
		t.Fatalf("expected the counter still at 3 after a failed insert, got %d", next)
	}

	_, err := s.NextAuto("name")
	if err == nil || err.Error() != `storage: field "name" is not an auto-increment field in collection "user"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryStore_ErrorsArePrefixed(t *testing.T) {
	s := newUserStore(t)

	err := s.Delete(RecordRef{})
	if err == nil || !strings.HasPrefix(err.Error(), "storage: ") {
		t.Fatalf("expected a storage-prefixed error, got %v", err)
	}

	_, err = s.Insert([]Entity{{"id": int64(1), "bad": 1}})
	if err == nil || !strings.HasPrefix(err.Error(), "storage: cannot encode record: record: ") {
		t.Fatalf("expected a wrapped encode error, got %v", err)
	}
}

func TestMemoryStore_WithoutPrimaryKey(t *testing.T) {
	s, err := NewMemoryStore(Config{Collection: "log"})
	if err != nil {
		t.Fatal(err)
	}

	refs := mustInsert(t, s, Entity{"msg": "a"}, Entity{"msg": "a"})
	if got := scanAll(t, s); len(got) != 2 {
		t.Fatalf("expected 2 records (duplicates are fine without a key), got %d", len(got))
	}

	if _, _, _, err := s.Lookup("a"); err == nil || err.Error() != `storage: collection "log" has no primary key to look up by` {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := s.Update(refs[0], Entity{"msg": "b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(refs[1]); err != nil {
		t.Fatal(err)
	}
	if got := scanAll(t, s); len(got) != 1 || got[refs[0]]["msg"] != "b" {
		t.Fatalf("expected one updated record, got %v", got)
	}
}

func TestMemoryStore_EmptyInsert(t *testing.T) {
	s := newUserStore(t)

	refs, err := s.Insert(nil)
	if err != nil || len(refs) != 0 {
		t.Fatalf("expected no refs and no error, got %v %v", refs, err)
	}
}

func TestMemoryStore_ConfigValidation(t *testing.T) {
	cases := map[string]Config{
		`storage: primary key "id" of collection "user" must be int or text`: {Collection: "user", PrimaryKey: "id"},
		`storage: collection "user" has a key type but no primary key`:       {Collection: "user", KeyType: IntKey},
	}

	for want, cfg := range cases {
		if _, err := NewMemoryStore(cfg); err == nil || err.Error() != want {
			t.Fatalf("expected %q, got %v", want, err)
		}
	}
}

func TestMemoryStore_CursorAfterTheEndAndAfterClose(t *testing.T) {
	s := newUserStore(t)
	mustInsert(t, s, Entity{"id": int64(1)})

	cur, _ := s.Scan()
	if !cur.Next() || cur.Next() || cur.Next() {
		t.Fatal("expected exactly one record, then false every time")
	}
	if cur.Record() != nil || cur.Err() != nil {
		t.Fatal("expected no record and no error after the end")
	}

	cur, _ = s.Scan()
	cur.Close()
	if cur.Next() {
		t.Fatal("expected no records after Close")
	}
}

func TestMemoryStore_EveryValueTypeRoundTrips(t *testing.T) {
	s := newUserStore(t)
	rec := Entity{
		"id":     int64(1),
		"price":  9.99,
		"name":   "Smith, Matt",
		"active": true,
		"gone":   false,
		"address": Entity{
			"city": "MSP",
			"geo":  Entity{"lat": 44.98, "lng": -93.26},
		},
	}

	refs := mustInsert(t, s, rec)

	if _, got, _, _ := s.Lookup(int64(1)); !reflect.DeepEqual(got, rec) {
		t.Fatalf("lookup changed the record:\n got %#v\nwant %#v", got, rec)
	}
	if got := scanAll(t, s)[refs[0]]; !reflect.DeepEqual(got, rec) {
		t.Fatalf("scan changed the record:\n got %#v\nwant %#v", got, rec)
	}

	updated := Entity{"id": int64(1), "address": Entity{"city": "STP"}}
	if err := s.Update(refs[0], updated); err != nil {
		t.Fatal(err)
	}
	if _, got, _, _ := s.Lookup(int64(1)); !reflect.DeepEqual(got, updated) {
		t.Fatalf("update didn't round-trip:\n got %#v\nwant %#v", got, updated)
	}
}
