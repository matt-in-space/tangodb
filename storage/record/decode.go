package record

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"
)

// DecodeError reports bytes that aren't a valid record, and where the
// problem was found.
type DecodeError struct {
	Offset int
	Reason string
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("record: %s at byte %d", e.Reason, e.Offset)
}

// Decode turns stored bytes back into a record. It never trusts its input:
// every read is bounds-checked first, and anything that isn't exactly what
// Encode would produce is an error rather than a panic or a guess. So any
// bytes that decode successfully re-encode to the same bytes.
func Decode(b []byte) (Entity, error) {
	d := &decoder{b: b}

	version, err := d.byte()
	if err != nil {
		return nil, err
	}
	if version != formatVersion {
		return nil, d.failAt(0, "unknown format version 0x%02x", version)
	}

	kind, err := d.byte()
	if err != nil {
		return nil, err
	}
	if kind != kindRecord {
		return nil, d.failAt(1, "unknown record kind 0x%02x", kind)
	}

	record, err := d.object(0, "")
	if err != nil {
		return nil, err
	}

	if d.pos != len(d.b) {
		return nil, d.failAt(d.pos, "%d trailing bytes after the record", len(d.b)-d.pos)
	}

	return record, nil
}

// decoder reads through b. pos is always an absolute offset into the whole
// record, so errors point at the right byte even inside nested objects.
type decoder struct {
	b   []byte
	pos int
}

func (d *decoder) failAt(offset int, format string, args ...any) error {
	return &DecodeError{Offset: offset, Reason: fmt.Sprintf(format, args...)}
}

func (d *decoder) remaining() int { return len(d.b) - d.pos }

func (d *decoder) byte() (byte, error) {
	if d.remaining() < 1 {
		return 0, d.failAt(d.pos, "unexpected end of input")
	}
	c := d.b[d.pos]
	d.pos++
	return c, nil
}

// take returns the next n bytes, after checking they exist.
func (d *decoder) take(n int) ([]byte, error) {
	if n < 0 || n > d.remaining() {
		return nil, d.failAt(d.pos, "unexpected end of input: need %d bytes, %d remain", n, d.remaining())
	}
	bytes := d.b[d.pos : d.pos+n]
	d.pos += n
	return bytes, nil
}

func (d *decoder) uvarint() (uint64, error) {
	v, n, reason := readUvarint(d.b[d.pos:])
	if reason != "" {
		return 0, d.failAt(d.pos, "%s", reason)
	}
	d.pos += n
	return v, nil
}

// length reads a uvarint byte length, and checks that many bytes actually
// remain, so a corrupt length can't make the decoder allocate gigabytes.
func (d *decoder) length() (int, error) {
	at := d.pos
	v, err := d.uvarint()
	if err != nil {
		return 0, err
	}
	if v > uint64(d.remaining()) {
		return 0, d.failAt(at, "length %d is more than the %d bytes remaining", v, d.remaining())
	}
	return int(v), nil
}

// smallestField is the fewest bytes a field can take: a name length, a
// one-byte name, and a payload-free tag.
const smallestField = 3

func (d *decoder) object(depth int, path string) (Entity, error) {
	countAt := d.pos
	count, err := d.uvarint()
	if err != nil {
		return nil, err
	}
	if count > uint64(d.remaining()/smallestField) {
		return nil, d.failAt(countAt, "field count %d is more than the remaining bytes could hold", count)
	}

	object := make(Entity, count)
	previous := ""

	for i := range count {
		nameAt := d.pos
		nameLength, err := d.byte()
		if err != nil {
			return nil, err
		}
		if nameLength == 0 {
			return nil, d.failAt(nameAt, "empty field name")
		}

		nameBytes, err := d.take(int(nameLength))
		if err != nil {
			return nil, err
		}
		if !utf8.Valid(nameBytes) {
			return nil, d.failAt(nameAt+1, "field name is not valid UTF-8")
		}
		name := string(nameBytes)

		if i > 0 && name == previous {
			return nil, d.failAt(nameAt, "duplicate field %q", path+name)
		}
		if i > 0 && name < previous {
			return nil, d.failAt(nameAt, "field %q is out of order (after %q)", path+name, path+previous)
		}
		previous = name

		value, err := d.value(depth, path+name)
		if err != nil {
			return nil, err
		}
		object[name] = value
	}

	return object, nil
}

func (d *decoder) value(depth int, path string) (any, error) {
	tagAt := d.pos
	tag, err := d.byte()
	if err != nil {
		return nil, err
	}

	switch tag {
	case tagInt:
		v, n, reason := readVarint(d.b[d.pos:])
		if reason != "" {
			return nil, d.failAt(d.pos, "%s", reason)
		}
		d.pos += n
		return v, nil

	case tagFloat:
		at := d.pos
		raw, err := d.take(8)
		if err != nil {
			return nil, err
		}
		f := math.Float64frombits(binary.LittleEndian.Uint64(raw))
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, d.failAt(at, "float for field %q is NaN or infinite", path)
		}
		return f, nil

	case tagText:
		n, err := d.length()
		if err != nil {
			return nil, err
		}
		at := d.pos
		text, _ := d.take(n)
		if !utf8.Valid(text) {
			return nil, d.failAt(at, "text for field %q is not valid UTF-8", path)
		}
		return string(text), nil

	case tagFalse:
		return false, nil

	case tagTrue:
		return true, nil

	case tagObject:
		if depth+1 > MaxDepth {
			return nil, d.failAt(tagAt, "field %q is nested more than %d levels deep", path, MaxDepth)
		}

		n, err := d.length()
		if err != nil {
			return nil, err
		}

		// Decode the object within exactly its stated length.
		end := d.pos + n
		inner := &decoder{b: d.b[:end], pos: d.pos}
		object, err := inner.object(depth+1, path+".")
		if err != nil {
			return nil, err
		}
		if inner.pos != end {
			return nil, d.failAt(inner.pos, "object for field %q is %d bytes, but its fields used %d", path, n, inner.pos-d.pos)
		}

		d.pos = end
		return object, nil

	default:
		return nil, d.failAt(tagAt, "unknown value tag 0x%02x", tag)
	}
}
