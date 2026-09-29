package core

import (
	"fmt"
	"sort"
)

// validateValue reports whether value's runtime type matches dataType exactly.
// There are no conversions: an int64 is never accepted for a float field.
func validateValue(dataType DataType, value any) error {
	var ok bool

	switch dataType {
	case TypeInt:
		_, ok = value.(int64)
	case TypeFloat:
		_, ok = value.(float64)
	case TypeText:
		_, ok = value.(string)
	case TypeBool:
		_, ok = value.(bool)
	}

	if !ok {
		return fmt.Errorf("expected %s, got %s", dataType, valueTypeName(value))
	}

	return nil
}

// validateFields checks every field against the collection's schema: it must
// be declared, and its value must match the declared type. Fields are checked
// in sorted order so the reported error is deterministic.
func validateFields(collection *Collection, fields map[string]any) error {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		dataType, ok := collection.data[name]
		if !ok {
			return fmt.Errorf("field %q not found in schema for collection %q", name, collection.name)
		}

		if err := validateValue(dataType, fields[name]); err != nil {
			return fmt.Errorf("field %q: %w", name, err)
		}
	}

	return nil
}

// valueTypeName names a value's type in the query language's terms.
func valueTypeName(value any) string {
	switch value.(type) {
	case int64:
		return TypeInt.String()
	case float64:
		return TypeFloat.String()
	case string:
		return TypeText.String()
	case bool:
		return TypeBool.String()
	default:
		return fmt.Sprintf("unsupported Go type %T", value)
	}
}
