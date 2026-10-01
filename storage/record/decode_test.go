package record

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestDecode_Golden(t *testing.T) {
	for _, g := range goldenRecords {
		got, err := Decode(mustHex(t, g.hex))
		if err != nil {
			t.Fatalf("%s: Decode failed: %v", g.name, err)
		}
		if !reflect.DeepEqual(got, g.record) {
			t.Fatalf("%s: got %#v, want %#v", g.name, got, g.record)
		}
	}
}

func TestRoundTrip_EveryValueType(t *testing.T) {
	record := Entity{
		"id":      int64(-9000),
		"big":     int64(math.MaxInt64),
		"small":   int64(math.MinInt64),
		"price":   9.99,
		"zero":    math.Copysign(0, -1),
		"name":    "Smith, Matt ☕",
		"empty":   "",
		"active":  true,
		"deleted": false,
		"address": Entity{
			"city": "MSP",
			"geo":  Entity{"lat": 44.98, "lng": -93.26},
			"tags": Entity{},
		},
	}

	encoded, err := Encode(record)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !reflect.DeepEqual(decoded, record) {
		t.Fatalf("round trip changed the record:\n got %#v\nwant %#v", decoded, record)
	}
	if !math.Signbit(decoded["zero"].(float64)) {
		t.Fatal("expected -0.0 to keep its sign")
	}

	reencoded, err := Encode(decoded)
	if err != nil || !bytes.Equal(reencoded, encoded) {
		t.Fatalf("expected re-encoding to give the same bytes, got % X (%v)", reencoded, err)
	}
}

func TestRoundTrip_GoldenBytesReencodeIdentically(t *testing.T) {
	for _, g := range goldenRecords {
		b := mustHex(t, g.hex)
		decoded, err := Decode(b)
		if err != nil {
			t.Fatalf("%s: Decode failed: %v", g.name, err)
		}
		if reencoded, _ := Encode(decoded); !bytes.Equal(reencoded, b) {
			t.Fatalf("%s: re-encoded to % X", g.name, reencoded)
		}
	}
}

// nestedBytes builds a record whose objects nest `levels` deep, by hand, since
// Encode refuses to produce one deeper than MaxDepth.
func nestedBytes(levels int) []byte {
	body := []byte{0x01, 0x01, 'v', tagTrue} // the innermost object: {v: true}
	for range levels {
		wrapped := []byte{0x01, 0x01, 'x', tagObject}
		wrapped = appendUvarint(wrapped, uint64(len(body)))
		body = append(wrapped, body...)
	}
	return append([]byte{formatVersion, kindRecord}, body...)
}

func TestDecode_RejectsMalformed(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty input", "", "record: unexpected end of input at byte 0"},
		{"unknown version", "02 00 00", "record: unknown format version 0x02 at byte 0"},
		{"reserved kind", "01 01 00", "record: unknown record kind 0x01 at byte 1"},
		{"unknown tag", "01 00 01 01 78 09", "record: unknown value tag 0x09 at byte 5"},
		{"reserved list tag", "01 00 01 01 78 07", "record: unknown value tag 0x07 at byte 5"},
		{"truncated", "01 00 02 02 69 64 01 54 04 6E", "record: unexpected end of input: need 4 bytes, 1 remain at byte 9"},
		{"non-minimal varint", "01 00 01 02 69 64 01 D4 00", "record: varint is not minimally encoded at byte 7"},
		{"length past the end", "01 00 01 01 74 03 05 41", "record: length 5 is more than the 1 bytes remaining at byte 6"},
		{"empty field name", "01 00 01 00 05 00", "record: empty field name at byte 3"},
		{"name not UTF-8", "01 00 01 01 FF 05", "record: field name is not valid UTF-8 at byte 4"},
		{"text not UTF-8", "01 00 01 01 74 03 01 FF", `record: text for field "t" is not valid UTF-8 at byte 7`},
		{"NaN", "01 00 01 01 70 02 00 00 00 00 00 00 F8 7F", `record: float for field "p" is NaN or infinite at byte 6`},
		{"infinity", "01 00 01 01 70 02 00 00 00 00 00 00 F0 7F", `record: float for field "p" is NaN or infinite at byte 6`},
		{"out of order", "01 00 02 01 62 05 01 61 05", `record: field "a" is out of order (after "b") at byte 6`},
		{"duplicate", "01 00 02 01 61 05 01 61 05", `record: duplicate field "a" at byte 6`},
		{"nested out of order", "01 00 01 01 6F 06 07 02 01 62 05 01 61 05", `record: field "o.a" is out of order (after "o.b") at byte 11`},
		{"object longer than its fields", "01 00 01 01 61 06 05 01 01 62 05 00", `record: object for field "a" is 5 bytes, but its fields used 4 at byte 11`},
		{"object shorter than its fields", "01 00 01 01 61 06 03 01 01 62 05", "record: field count 1 is more than the remaining bytes could hold at byte 7"},
		{"huge field count", "01 00 FF FF FF FF 0F", "record: field count 4294967295 is more than the remaining bytes could hold at byte 2"},
		{"trailing bytes", "01 00 02 02 69 64 01 54 04 6E 61 6D 65 03 04 4D 61 74 74 00", "record: 1 trailing bytes after the record at byte 19"},
	}

	for _, c := range cases {
		_, err := Decode(mustHex(t, c.input))
		if err == nil || err.Error() != c.want {
			t.Fatalf("%s: expected %q, got %v", c.name, c.want, err)
		}

		var decodeErr *DecodeError
		if !errors.As(err, &decodeErr) {
			t.Fatalf("%s: expected a *DecodeError, got %T", c.name, err)
		}
	}
}

func TestDecode_NestingLimit(t *testing.T) {
	if _, err := Decode(nestedBytes(MaxDepth)); err != nil {
		t.Fatalf("expected %d levels to decode, got %v", MaxDepth, err)
	}

	_, err := Decode(nestedBytes(MaxDepth + 1))
	if err == nil || !strings.Contains(err.Error(), "is nested more than 32 levels deep") {
		t.Fatalf("expected the nesting error, got %v", err)
	}
}

func TestDecode_EveryTruncationFailsCleanly(t *testing.T) {
	for _, g := range goldenRecords {
		full := mustHex(t, g.hex)
		for n := range len(full) {
			if _, err := Decode(full[:n]); err == nil {
				t.Fatalf("%s: expected the first %d bytes to fail", g.name, n)
			}
		}
	}
}
