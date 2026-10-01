package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/matt-in-space/tangodb/storage/record"
)

type DataType int

const (
	TypeInt DataType = iota
	TypeFloat
	TypeText
	TypeBool
	TypeObject // an embedded block; its fields are described by a Schema
)

func (d DataType) String() string {
	switch d {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeText:
		return "text"
	case TypeBool:
		return "bool"
	case TypeObject:
		return "object"
	default:
		return "unknown"
	}
}

// Schema describes the fields of an embedded block: each field's type, which
// fields are optional, and, for fields that are themselves blocks, their shape.
type Schema struct {
	Data     map[string]DataType
	Optional map[string]bool
	Objects  map[string]*Schema
}

type Collection struct {
	name         string
	data         map[string]DataType
	records      map[uint64]Entity
	nextID       uint64
	primaryKey   string
	primaryIndex map[any]uint64
	autoCounters map[string]int64
	optional     map[string]bool
	objects      map[string]*Schema
}

// rootSchema returns the collection's top-level fields as a Schema, so code
// that walks embedded blocks can treat the top level like any other block.
func (c Collection) rootSchema() *Schema {
	return &Schema{Data: c.data, Optional: c.optional, Objects: c.objects}
}

func (c Collection) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s {\n", c.name)
	c.writeFields(&b, c.rootSchema(), "  ", true)
	b.WriteString("}")

	return b.String()
}

// writeFields writes a block's fields in declaration syntax, one per line at
// the given indent, recursing into embedded blocks. Only the top level can
// carry @id and @auto.
func (c Collection) writeFields(b *strings.Builder, schema *Schema, indent string, top bool) {
	fields := make([]string, 0, len(schema.Data))
	for field := range schema.Data {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	for _, field := range fields {
		if schema.Data[field] == TypeObject {
			fmt.Fprintf(b, "%s%s: {\n", indent, field)
			c.writeFields(b, schema.Objects[field], indent+"  ", false)
			fmt.Fprintf(b, "%s}", indent)
		} else {
			fmt.Fprintf(b, "%s%s: %s", indent, field, schema.Data[field])
		}
		if top && field == c.primaryKey {
			b.WriteString(" @id")
		}
		if top && c.isAuto(field) {
			b.WriteString(" @auto")
		}
		if schema.Optional[field] {
			b.WriteString(" @optional")
		}
		b.WriteString("\n")
	}
}

// isAuto reports whether field is auto-increment, with a counter the database assigns from.
func (c Collection) isAuto(field string) bool {
	_, ok := c.autoCounters[field]
	return ok
}

// sortedAutoFields returns the auto-increment field names in sorted order,
// so checks and counter assignment happen deterministically.
func (c Collection) sortedAutoFields() []string {
	fields := make([]string, 0, len(c.autoCounters))
	for field := range c.autoCounters {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

// Entity is a record. It's an alias of record.Entity, so a nested object is the
// same type in both packages: a type switch on Entity in either one matches it.
type Entity = record.Entity
