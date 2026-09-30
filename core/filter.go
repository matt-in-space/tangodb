package core

import (
	"fmt"
	"sort"
	"strings"
)

// wildcardObject is the value of `{*}` in a filter: the embedded object is
// present, whatever it contains.
type wildcardObject struct{}

// parseWildcardObject parses `{*}`, which is only allowed in a filter, and
// only on its own: `{* city: "MSP"}` is an error, as `{*, id}` is in a
// projection.
func (p *parser) parseWildcardObject() (any, error) {
	if !p.inFilter {
		return nil, fmt.Errorf("{*} is only allowed in a filter")
	}

	p.next() // {
	p.next() // *

	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}

	return wildcardObject{}, nil
}

// checkConstrainedOnce rejects a filter that constrains the same field more
// than once. A dotted key and a subset filter can reach the same field, as in
// (address.city: "A" address: {city: "B"}), so each condition is expanded to
// the paths it constrains before comparing.
func checkConstrainedOnce(filter map[string]any) error {
	seen := map[string]bool{}

	for _, key := range sortedKeys(filter) {
		for _, path := range constrainedPaths(key, filter[key]) {
			if seen[path] {
				return fmt.Errorf("field %q is given more than once", path)
			}
			seen[path] = true
		}
	}

	return nil
}

// constrainedPaths lists the field paths a filter condition constrains: a
// subset object constrains each field inside it (recursively), and anything
// else constrains the path itself.
func constrainedPaths(path string, value any) []string {
	object, ok := value.(Entity)
	if !ok || len(object) == 0 {
		return []string{path}
	}

	var paths []string
	for _, field := range sortedKeys(object) {
		paths = append(paths, constrainedPaths(path+"."+field, object[field])...)
	}
	sort.Strings(paths)

	return paths
}

// validateFilter checks each condition of a read, delete, or merge filter
// against the schema at its path, in sorted order, and returns the first
// problem.
func validateFilter(collection *Collection, filter map[string]any) error {
	for _, key := range sortedKeys(filter) {
		block, name, optionalAlong, err := resolveFilterPath(collection, key)
		if err != nil {
			return err
		}

		if err := filterValueProblem(collection.name, block, name, key, filter[key], optionalAlong); err != nil {
			return err
		}
	}

	return nil
}

// resolveFilterPath finds the field a filter key names, which may be a dotted
// path into embedded objects. It returns the block the field belongs to, the
// field's name within that block, and whether the field or any block above it
// is optional (in which case the path can have no value).
func resolveFilterPath(collection *Collection, key string) (block *Schema, name string, optionalAlong bool, err error) {
	block = collection.rootSchema()
	segments := strings.Split(key, ".")

	for i, segment := range segments {
		dataType, declared := block.Data[segment]
		if !declared {
			break
		}

		if i == len(segments)-1 {
			return block, segment, optionalAlong || block.Optional[segment], nil
		}

		if dataType != TypeObject {
			break
		}

		optionalAlong = optionalAlong || block.Optional[segment]
		block = block.Objects[segment]
	}

	return nil, "", false, fmt.Errorf("field %q not found in schema for collection %q", key, collection.name)
}

// filterValueProblem checks one filter condition against the field it names.
// nullAllowed says whether the field can have no value. On a dotted path,
// that's true if the field or any block above it is optional; inside a subset
// filter, which asserts the object is present, only the field's own
// optionality counts.
func filterValueProblem(collectionName string, block *Schema, name, path string, value any, nullAllowed bool) error {
	dataType := block.Data[name]

	switch v := value.(type) {
	case nil:
		if !nullAllowed {
			return fmt.Errorf("field %q is required and cannot be null", path)
		}
		return nil

	case wildcardObject:
		if dataType != TypeObject {
			return fmt.Errorf("field %q: expected %s, got {*}", path, dataType)
		}
		return nil

	case Entity:
		if dataType != TypeObject {
			return fmt.Errorf("field %q: expected %s, got object", path, dataType)
		}

		if len(v) == 0 {
			return fmt.Errorf("empty object filter for field %q; use {*} to match any value", path)
		}

		inner := block.Objects[name]
		for _, field := range sortedKeys(v) {
			fieldPath := path + "." + field

			if _, declared := inner.Data[field]; !declared {
				return fmt.Errorf("field %q not found in schema for collection %q", fieldPath, collectionName)
			}

			if err := filterValueProblem(collectionName, inner, field, fieldPath, v[field], inner.Optional[field]); err != nil {
				return err
			}
		}
		return nil
	}

	if dataType == TypeObject {
		return fmt.Errorf("field %q: expected object, got %s", path, valueTypeName(value))
	}

	if err := validateValue(dataType, value); err != nil {
		return fmt.Errorf("field %q: %w", path, err)
	}

	return nil
}
