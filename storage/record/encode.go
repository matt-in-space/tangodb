package record

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"unicode/utf8"
)

// Encode turns a record into its stored bytes. It only accepts what a
// validated record can contain, and it reports anything else as an error
// rather than writing bytes that wouldn't decode back to the same record.
func Encode(e Entity) ([]byte, error) {
	b := []byte{formatVersion, kindRecord}
	return appendObject(b, e, 0, "")
}

func appendObject(b []byte, e Entity, depth int, path string) ([]byte, error) {
	names := make([]string, 0, len(e))
	for name := range e {
		names = append(names, name)
	}
	sort.Strings(names)

	b = appendUvarint(b, uint64(len(names)))

	for _, name := range names {
		fullName := path + name

		if len(name) == 0 {
			return nil, fmt.Errorf("record: field %q has an empty name", fullName)
		}
		if len(name) > MaxNameLength {
			return nil, fmt.Errorf("record: field name %q is longer than %d bytes", name, MaxNameLength)
		}
		if !utf8.ValidString(name) {
			return nil, fmt.Errorf("record: field name %q is not valid UTF-8", fullName)
		}

		b = append(b, byte(len(name)))
		b = append(b, name...)

		var err error
		b, err = appendValue(b, e[name], depth, fullName)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func appendValue(b []byte, value any, depth int, path string) ([]byte, error) {
	switch v := value.(type) {
	case int64:
		b = append(b, tagInt)
		return appendVarint(b, v), nil

	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("record: field %q is NaN or infinite", path)
		}
		b = append(b, tagFloat)
		return binary.LittleEndian.AppendUint64(b, math.Float64bits(v)), nil

	case string:
		if !utf8.ValidString(v) {
			return nil, fmt.Errorf("record: field %q is not valid UTF-8", path)
		}
		b = append(b, tagText)
		b = appendUvarint(b, uint64(len(v)))
		return append(b, v...), nil

	case bool:
		if v {
			return append(b, tagTrue), nil
		}
		return append(b, tagFalse), nil

	case Entity:
		if depth+1 > MaxDepth {
			return nil, fmt.Errorf("record: field %q is nested more than %d levels deep", path, MaxDepth)
		}

		// The object is length-prefixed, so it's encoded on its own first to
		// learn its length.
		body, err := appendObject(nil, v, depth+1, path+".")
		if err != nil {
			return nil, err
		}

		b = append(b, tagObject)
		b = appendUvarint(b, uint64(len(body)))
		return append(b, body...), nil

	case nil:
		// "No value" is an absent field, never a stored nil.
		return nil, fmt.Errorf("record: field %q is nil; a field with no value should be absent", path)

	default:
		return nil, fmt.Errorf("record: field %q has unsupported type %T", path, value)
	}
}
