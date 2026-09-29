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
// in sorted order and the first problem is returned, so the reported error is
// deterministic.
func validateFields(collection *Collection, fields map[string]any) error {
	for _, name := range sortedKeys(fields) {
		if err := fieldProblem(collection, name, fields[name]); err != nil {
			return err
		}
	}

	return nil
}

// fieldProblem checks one field against the schema: it must be declared, a
// null is only allowed on an optional field, and any other value must match
// the declared type exactly.
func fieldProblem(collection *Collection, name string, value any) error {
	dataType, ok := collection.data[name]
	if !ok {
		return fmt.Errorf("field %q not found in schema for collection %q", name, collection.name)
	}

	if value == nil {
		if !collection.optional[name] {
			return fmt.Errorf("field %q is required and cannot be null", name)
		}
		return nil
	}

	if err := validateValue(dataType, value); err != nil {
		return fmt.Errorf("field %q: %w", name, err)
	}

	return nil
}

// recordProblems checks a record being inserted and returns every problem
// found, rather than stopping at the first. Checks run in a fixed order, and
// once a field has a problem, later checks skip it, so one mistake is reported
// once. It also returns the set of fields that had a problem.
func recordProblems(collection *Collection, record Entity) ([]string, map[string]bool) {
	var problems []string
	bad := map[string]bool{}

	report := func(field string, err error) {
		problems = append(problems, err.Error())
		bad[field] = true
	}

	for _, field := range collection.sortedAutoFields() {
		if _, supplied := record[field]; supplied {
			report(field, fmt.Errorf("field %q is auto-increment and must not be supplied for collection %q", field, collection.name))
		}
	}

	key := collection.primaryKey
	if key != "" && !collection.isAuto(key) && !bad[key] && record[key] == nil {
		report(key, fmt.Errorf("record missing primary key %q for collection %q", key, collection.name))
	}

	for _, name := range sortedKeys(record) {
		if bad[name] {
			continue
		}
		if err := fieldProblem(collection, name, record[name]); err != nil {
			report(name, err)
		}
	}

	for _, name := range sortedKeys(collection.data) {
		if bad[name] || collection.optional[name] || collection.isAuto(name) {
			continue
		}
		if record[name] == nil {
			report(name, fmt.Errorf("field %q is required for collection %q", name, collection.name))
		}
	}

	return problems, bad
}

// sortedKeys returns a map's keys in sorted order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
