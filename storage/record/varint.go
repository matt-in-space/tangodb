package record

// Varints store an integer in as few bytes as it needs: 7 bits per byte,
// least significant group first, with the top bit set on every byte except
// the last. 42 is 2A; 300 is AC 02.

// maxVarintLen is the most bytes a 64-bit varint can take: ceil(64 / 7).
const maxVarintLen = 10

// appendUvarint appends v as an unsigned varint.
func appendUvarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

// appendVarint appends v as a zigzag varint. Zigzag maps signed numbers to
// unsigned ones so small negatives stay small: 0->0, -1->1, 1->2, -2->3.
func appendVarint(b []byte, v int64) []byte {
	return appendUvarint(b, uint64(v<<1)^uint64(v>>63))
}

// readUvarint reads an unsigned varint from the start of b, returning the
// value and how many bytes it used. Every value has exactly one encoding, so
// a non-minimal varint (one that ends in a 00 byte after the first, such as
// 42 written as AA 00) is rejected, as is one too big for 64 bits. A varint
// can't run past 10 bytes: by the 10th, only 00 or 01 fit (anything else
// overflows), and both end it. On failure, reason says why.
func readUvarint(b []byte) (v uint64, n int, reason string) {
	var shift uint

	for i, c := range b {
		// The 10th byte holds only the 64th bit.
		if i == maxVarintLen-1 && c > 1 {
			return 0, 0, "varint overflows 64 bits"
		}

		v |= uint64(c&0x7f) << shift

		if c < 0x80 {
			if i > 0 && c == 0 {
				return 0, 0, "varint is not minimally encoded"
			}
			return v, i + 1, ""
		}

		shift += 7
	}

	return 0, 0, "unexpected end of input in varint"
}

// readVarint reads a zigzag varint, as written by appendVarint.
func readVarint(b []byte) (v int64, n int, reason string) {
	u, n, reason := readUvarint(b)
	if reason != "" {
		return 0, 0, reason
	}
	return int64(u>>1) ^ -int64(u&1), n, ""
}
