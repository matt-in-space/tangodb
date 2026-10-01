package record

// The byte format of a stored record:
//
//	record   version (1 byte) | kind (1 byte) | object
//	object   field count (uvarint) | fields, sorted by name
//	field    name length (1 byte, 1-255) | name (UTF-8) | tag (1 byte) | payload
//
// Value tags and their payloads:
//
//	0x01 int     zigzag varint
//	0x02 float   8 bytes: IEEE 754 bits, little-endian
//	0x03 text    uvarint byte length | UTF-8 bytes
//	0x04 false   (no payload)
//	0x05 true    (no payload)
//	0x06 object  uvarint byte length | object
//
// There's no tag for null: a field with no value isn't written. Sorted field
// names and minimal varints make the encoding canonical: a record has exactly
// one encoding.

const formatVersion = 0x01

// Record kinds. Only kindRecord is produced so far; the others are reserved
// for records that moved to another page and records too big for one.
const (
	kindRecord   = 0x00
	kindForward  = 0x01 // reserved
	kindOverflow = 0x02 // reserved
)

// Value tags.
const (
	tagInt    = 0x01
	tagFloat  = 0x02
	tagText   = 0x03
	tagFalse  = 0x04
	tagTrue   = 0x05
	tagObject = 0x06
	tagList   = 0x07 // reserved
	tagRef    = 0x08 // reserved
)

// Limits that keep every record encodable and stop a corrupt one from making
// the decoder recurse without bound.
const (
	MaxNameLength = 255 // a name's length is stored in one byte
	MaxDepth      = 32  // how deeply objects may nest inside a record
)
