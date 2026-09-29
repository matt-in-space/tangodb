package core

import (
	"fmt"
	"sort"
	"strings"
)

type DataType int

const (
	TypeInt DataType = iota
	TypeFloat
	TypeText
	TypeBool
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
	default:
		return "unknown"
	}
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
}

func (c Collection) String() string {
	fields := make([]string, 0, len(c.data))
	for field := range c.data {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	var b strings.Builder
	fmt.Fprintf(&b, "%s {\n", c.name)

	for _, field := range fields {
		fmt.Fprintf(&b, "  %s: %s", field, c.data[field])
		if field == c.primaryKey {
			b.WriteString(" @id")
		}
		if c.isAuto(field) {
			b.WriteString(" @auto")
		}
		if c.optional[field] {
			b.WriteString(" @optional")
		}
		b.WriteString("\n")
	}

	b.WriteString("}")

	return b.String()
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

type Entity map[string]any
