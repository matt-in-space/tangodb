package record

import (
	"bytes"
	"testing"
)

// FuzzDecode throws arbitrary bytes at Decode. It must never panic, and
// anything it accepts must re-encode to exactly the same bytes, since the
// format has one encoding per record. Run it with:
//
//	go test -fuzz=FuzzDecode ./storage/record
func FuzzDecode(f *testing.F) {
	for _, g := range goldenRecords {
		f.Add(mustHex(f, g.hex))
	}
	f.Add(nestedBytes(MaxDepth))

	f.Fuzz(func(t *testing.T, b []byte) {
		record, err := Decode(b)
		if err != nil {
			return
		}

		reencoded, err := Encode(record)
		if err != nil {
			t.Fatalf("decoded % X into %#v, which then failed to encode: %v", b, record, err)
		}
		if !bytes.Equal(reencoded, b) {
			t.Fatalf("decoded % X, but it re-encoded to % X", b, reencoded)
		}
	})
}
