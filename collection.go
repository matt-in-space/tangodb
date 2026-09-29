package main

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
	name          string
	data          map[string]DataType
	records       map[uint64]Entity
	nextID        uint64
	primaryKey    string
	primaryIndex  map[any]uint64
	autoIncrement bool
	nextAutoValue int64
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
			if c.autoIncrement {
				b.WriteString(" @auto")
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("}")

	return b.String()
}

type Entity map[string]any
