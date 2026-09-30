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
	root := collection.rootSchema()

	for _, name := range sortedKeys(fields) {
		if problems := valueProblems(collection.name, root, "", name, fields[name]); len(problems) > 0 {
			return problems[0]
		}
	}

	return nil
}

// valueProblems checks the value of one field of a block against the block's
// schema, and returns every problem found. path is the block's dotted prefix
// ("" at the top level, "address." inside), so each problem names the field's
// full path. An object value is checked field by field against its block.
func valueProblems(collectionName string, schema *Schema, path, name string, value any) []error {
	fullName := path + name

	dataType, ok := schema.Data[name]
	if !ok {
		return []error{fmt.Errorf("field %q not found in schema for collection %q", fullName, collectionName)}
	}

	if value == nil {
		if !schema.Optional[name] {
			return []error{fmt.Errorf("field %q is required and cannot be null", fullName)}
		}
		return nil
	}

	if dataType == TypeObject {
		object, ok := value.(Entity)
		if !ok {
			return []error{fmt.Errorf("field %q: expected object, got %s", fullName, valueTypeName(value))}
		}
		var problems []error
		for _, problem := range objectProblems(collectionName, schema.Objects[name], fullName+".", object, nil) {
			problems = append(problems, problem.err)
		}
		return problems
	}

	if err := validateValue(dataType, value); err != nil {
		return []error{fmt.Errorf("field %q: %w", fullName, err)}
	}

	return nil
}

// fieldError is a problem with one field of a block, by the field's name
// within that block.
type fieldError struct {
	field string
	err   error
}

// objectProblems checks a record or an embedded object against its block's
// schema: every field it supplies, then every required field it's missing,
// each in sorted order. Fields in skip are left out of both checks, because
// the caller has already reported them or the database fills them in.
func objectProblems(collectionName string, schema *Schema, path string, object Entity, skip map[string]bool) []fieldError {
	var problems []fieldError
	reported := map[string]bool{}

	for _, name := range sortedKeys(object) {
		if skip[name] {
			continue
		}
		for _, err := range valueProblems(collectionName, schema, path, name, object[name]) {
			problems = append(problems, fieldError{name, err})
			reported[name] = true
		}
	}

	for _, name := range sortedKeys(schema.Data) {
		if skip[name] || reported[name] || schema.Optional[name] {
			continue
		}
		if object[name] == nil {
			problems = append(problems, fieldError{name, fmt.Errorf("field %q is required for collection %q", path+name, collectionName)})
		}
	}

	return problems
}

// recordProblems checks a record being inserted and returns every problem
// found, rather than stopping at the first. The top-level-only checks
// (auto-increment and primary key fields) run first; once a field has a
// problem, later checks skip it, so one mistake is reported once. It also
// returns the set of top-level fields that had a problem.
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

	// Auto fields are filled in by the database, and fields already reported
	// shouldn't be reported again.
	skip := map[string]bool{}
	for field := range bad {
		skip[field] = true
	}
	for field := range collection.autoCounters {
		skip[field] = true
	}

	for _, problem := range objectProblems(collection.name, collection.rootSchema(), "", record, skip) {
		report(problem.field, problem.err)
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
	case Entity:
		return TypeObject.String()
	default:
		return fmt.Sprintf("unsupported Go type %T", value)
	}
}
