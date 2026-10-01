package record

import (
	"bytes"
	"encoding/hex"
	"math"
	"strings"
	"testing"
)

// mustHex turns "01 00 02 ..." into bytes.
func mustHex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// goldenRecords pairs records with their exact encodings, from the
// record-encoding spec. They pin the format down: if one changes, every
// stored record would read differently.
var goldenRecords = []struct {
	name   string
	record Entity
	hex    string
}{
	{"flat", Entity{"id": int64(42), "name": "Matt"},
		"01 00 02 02 69 64 01 54 04 6E 61 6D 65 03 04 4D 61 74 74"},
	{"nested", Entity{"id": int64(1), "address": Entity{"street": "1 Main", "city": "MSP"}},
		"01 00 02 07 61 64 64 72 65 73 73 06 1A 02 04 63 69 74 79 03 03 4D 53 50 06 73 74 72 65 65 74 03 06 31 20 4D 61 69 6E 02 69 64 01 02"},
	{"empty object", Entity{"address": Entity{}},
		"01 00 01 07 61 64 64 72 65 73 73 06 01 00"},
	{"true", Entity{"ok": true}, "01 00 01 02 6F 6B 05"},
	{"false", Entity{"ok": false}, "01 00 01 02 6F 6B 04"},
	{"negative int", Entity{"n": int64(-1)}, "01 00 01 01 6E 01 01"},
	{"float", Entity{"p": 1.5}, "01 00 01 01 70 02 00 00 00 00 00 00 F8 3F"},
	{"empty record", Entity{}, "01 00 00"},
}

func TestEncode_Golden(t *testing.T) {
	for _, g := range goldenRecords {
		got, err := Encode(g.record)
		if err != nil {
			t.Fatalf("%s: Encode failed: %v", g.name, err)
		}

		if want := mustHex(t, g.hex); !bytes.Equal(got, want) {
			t.Fatalf("%s:\n got % X\nwant % X", g.name, got, want)
		}
	}
}

func TestEncode_FlatRecordIs19Bytes(t *testing.T) {
	got, _ := Encode(Entity{"id": int64(42), "name": "Matt"})
	if len(got) != 19 {
		t.Fatalf("expected 19 bytes, got %d", len(got))
	}
}

func TestEncode_FieldOrderDoesNotMatter(t *testing.T) {
	a := Entity{}
	a["name"] = "Matt"
	a["id"] = int64(42)
	a["address"] = Entity{"zip": "55401", "city": "MSP"}

	b := Entity{}
	b["address"] = Entity{"city": "MSP", "zip": "55401"}
	b["id"] = int64(42)
	b["name"] = "Matt"

	for range 20 {
		encodedA, _ := Encode(a)
		encodedB, _ := Encode(b)
		if !bytes.Equal(encodedA, encodedB) {
			t.Fatalf("expected identical bytes:\n% X\n% X", encodedA, encodedB)
		}
	}
}

func TestEncode_RejectsWhatCannotBeStored(t *testing.T) {
	// The outermost Entity is the record itself (depth 0), so 33 wrappers
	// nest 33 objects inside it: one more than allowed.
	deep := Entity{"v": int64(1)}
	for range MaxDepth + 1 {
		deep = Entity{"x": deep}
	}
	deepPath := strings.TrimSuffix(strings.Repeat("x.", MaxDepth+1), ".")

	cases := []struct {
		record Entity
		want   string
	}{
		{Entity{"a": nil}, `record: field "a" is nil; a field with no value should be absent`},
		{Entity{"a": Entity{"b": nil}}, `record: field "a.b" is nil; a field with no value should be absent`},
		{Entity{"a": 1}, `record: field "a" has unsupported type int`},
		{Entity{"a": math.NaN()}, `record: field "a" is NaN or infinite`},
		{Entity{"a": math.Inf(-1)}, `record: field "a" is NaN or infinite`},
		{Entity{"a": "\xff"}, `record: field "a" is not valid UTF-8`},
		{Entity{"": int64(1)}, `record: field "" has an empty name`},
		{Entity{strings.Repeat("n", 256): int64(1)}, `record: field name "` + strings.Repeat("n", 256) + `" is longer than 255 bytes`},
		{deep, `record: field "` + deepPath + `" is nested more than 32 levels deep`},
	}

	for _, c := range cases {
		_, err := Encode(c.record)
		if err == nil || err.Error() != c.want {
			t.Fatalf("expected %q, got %v", c.want, err)
		}
	}
}

func TestEncode_AllowsExactlyTheMaximums(t *testing.T) {
	deep := Entity{"v": int64(1)}
	for range MaxDepth - 1 {
		deep = Entity{"x": deep}
	}

	for _, record := range []Entity{
		{strings.Repeat("n", MaxNameLength): int64(1)},
		{"x": deep},
		{"z": math.Copysign(0, -1)},
	} {
		if _, err := Encode(record); err != nil {
			t.Fatalf("expected the record to encode, got %v", err)
		}
	}
}
