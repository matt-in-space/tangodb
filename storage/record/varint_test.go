package record

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

var uvarintCases = []uint64{0, 1, 42, 127, 128, 255, 300, 16383, 16384, 1 << 32, math.MaxInt64, math.MaxUint64}

var varintCases = []int64{0, 1, -1, 2, -2, 42, -42, 63, -64, 64, -65, 300, -300, math.MaxInt64, math.MinInt64}

func TestUvarint_MatchesEncodingBinary(t *testing.T) {
	for _, v := range uvarintCases {
		want := binary.AppendUvarint(nil, v)
		got := appendUvarint(nil, v)
		if !bytes.Equal(got, want) {
			t.Fatalf("appendUvarint(%d) = % x, want % x", v, got, want)
		}

		decoded, n, reason := readUvarint(got)
		if reason != "" || decoded != v || n != len(got) {
			t.Fatalf("readUvarint(% x) = %d, %d, %q; want %d, %d", got, decoded, n, reason, v, len(got))
		}
	}
}

func TestVarint_MatchesEncodingBinary(t *testing.T) {
	for _, v := range varintCases {
		want := binary.AppendVarint(nil, v)
		got := appendVarint(nil, v)
		if !bytes.Equal(got, want) {
			t.Fatalf("appendVarint(%d) = % x, want % x", v, got, want)
		}

		decoded, n, reason := readVarint(got)
		if reason != "" || decoded != v || n != len(got) {
			t.Fatalf("readVarint(% x) = %d, %d, %q; want %d", got, decoded, n, reason, v)
		}
	}
}

func TestVarint_KnownBytes(t *testing.T) {
	cases := map[uint64][]byte{0: {0x00}, 42: {0x2A}, 300: {0xAC, 0x02}}
	for v, want := range cases {
		if got := appendUvarint(nil, v); !bytes.Equal(got, want) {
			t.Fatalf("appendUvarint(%d) = % x, want % x", v, got, want)
		}
	}

	if got := appendVarint(nil, 42); !bytes.Equal(got, []byte{0x54}) {
		t.Fatalf("appendVarint(42) = % x, want 54", got)
	}
	if got := appendVarint(nil, -1); !bytes.Equal(got, []byte{0x01}) {
		t.Fatalf("appendVarint(-1) = % x, want 01", got)
	}
}

func TestUvarint_RejectsMalformed(t *testing.T) {
	cases := map[string][]byte{
		"unexpected end of input in varint": {0x80},
		"varint is not minimally encoded":   {0xAA, 0x00},
		"varint overflows 64 bits":          {0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x02},
	}

	for want, input := range cases {
		if _, _, reason := readUvarint(input); reason != want {
			t.Fatalf("readUvarint(% x): expected %q, got %q", input, want, reason)
		}
	}

	if _, _, reason := readUvarint(nil); reason == "" {
		t.Fatal("expected an error for empty input")
	}
}

func TestUvarint_ReadsOnlyItsOwnBytes(t *testing.T) {
	v, n, reason := readUvarint([]byte{0xAC, 0x02, 0xFF})
	if reason != "" || v != 300 || n != 2 {
		t.Fatalf("expected 300 using 2 bytes, got %d using %d (%q)", v, n, reason)
	}
}

func TestUvarint_TenthByteCanOnlyHoldOneBit(t *testing.T) {
	maxBytes := appendUvarint(nil, math.MaxUint64)
	if len(maxBytes) != maxVarintLen || maxBytes[maxVarintLen-1] != 0x01 {
		t.Fatalf("expected the largest value to take 10 bytes ending in 01, got % x", maxBytes)
	}

	// Even a continuation bit on the 10th byte is an overflow, so a varint
	// can never run past 10 bytes.
	tooLong := append(bytes.Repeat([]byte{0x80}, 9), 0x80, 0x01)
	if _, _, reason := readUvarint(tooLong); reason != "varint overflows 64 bits" {
		t.Fatalf("expected an overflow, got %q", reason)
	}
}
