package core

import (
	"fmt"
	"strings"
)

// normalizePayload rewrites a merge payload so every dotted key becomes
// nested: {address.city: "STP"} becomes {address: {city: "STP"}}, merged with
// any sibling object for the same field. After this, the rest of merge only
// sees plain nested objects.
//
// It rejects a payload that sets the same field twice, or that sets a value
// (such as null) for a field while also setting something inside it: the
// result would depend on which one ran first.
func normalizePayload(payload Entity) (Entity, error) {
	n := payloadNormalizer{
		result:     Entity{},
		containers: map[string]bool{},
		leaves:     map[string]bool{},
	}

	for _, key := range sortedKeys(payload) {
		if err := n.add(key, payload[key]); err != nil {
			return nil, err
		}
	}

	return n.result, nil
}

type payloadNormalizer struct {
	result Entity

	// containers are the paths of objects built to hold deeper values;
	// leaves are the paths given a value directly.
	containers map[string]bool
	leaves     map[string]bool
	leafOrder  []string
}

// add places one payload entry. A non-empty object is spread into its fields,
// so it can combine with dotted keys into the same object; anything else,
// including null and an empty object (which validation rejects), is a value
// set at its path.
func (n *payloadNormalizer) add(path string, value any) error {
	if object, ok := value.(Entity); ok && len(object) > 0 {
		for _, field := range sortedKeys(object) {
			if err := n.add(path+"."+field, object[field]); err != nil {
				return err
			}
		}
		return nil
	}

	return n.set(path, value)
}

func (n *payloadNormalizer) set(path string, value any) error {
	segments := strings.Split(path, ".")
	node := n.result

	for i, segment := range segments[:len(segments)-1] {
		prefix := strings.Join(segments[:i+1], ".")

		if n.leaves[prefix] {
			return fmt.Errorf("field %q conflicts with %q", path, prefix)
		}

		if !n.containers[prefix] {
			n.containers[prefix] = true
			node[segment] = Entity{}
		}
		node = node[segment].(Entity)
	}

	if n.leaves[path] {
		return fmt.Errorf("field %q is given more than once", path)
	}

	if n.containers[path] {
		return fmt.Errorf("field %q conflicts with %q", n.firstLeafUnder(path), path)
	}

	n.leaves[path] = true
	n.leafOrder = append(n.leafOrder, path)
	node[segments[len(segments)-1]] = value

	return nil
}

// firstLeafUnder returns the first value set beneath a container path, to
// name in a conflict error.
func (n *payloadNormalizer) firstLeafUnder(path string) string {
	for _, leaf := range n.leafOrder {
		if strings.HasPrefix(leaf, path+".") {
			return leaf
		}
	}
	return path
}

// validatePayload checks a normalized merge payload against the schema at
// every path, before any record is looked at: every field must be declared,
// values must match their types and shapes, and null is only allowed on
// optional fields. Fields the payload leaves out aren't required, since a
// merge is a partial update. The first problem is returned.
func validatePayload(collection *Collection, payload Entity) error {
	return payloadObjectProblem(collection.name, collection.rootSchema(), "", payload)
}

func payloadObjectProblem(collectionName string, schema *Schema, path string, object Entity) error {
	for _, name := range sortedKeys(object) {
		if err := payloadValueProblem(collectionName, schema, path, name, object[name]); err != nil {
			return err
		}
	}
	return nil
}

func payloadValueProblem(collectionName string, schema *Schema, path, name string, value any) error {
	fullName := path + name

	dataType, declared := schema.Data[name]
	if !declared {
		return fmt.Errorf("field %q not found in schema for collection %q", fullName, collectionName)
	}

	switch v := value.(type) {
	case nil:
		if !schema.Optional[name] {
			return fmt.Errorf("field %q is required and cannot be null", fullName)
		}
		return nil

	case Entity:
		if dataType != TypeObject {
			return fmt.Errorf("field %q: expected %s, got object", fullName, dataType)
		}
		if len(v) == 0 {
			return fmt.Errorf("empty object in merge payload for field %q", fullName)
		}
		return payloadObjectProblem(collectionName, schema.Objects[name], fullName+".", v)
	}

	if dataType == TypeObject {
		return fmt.Errorf("field %q: expected object, got %s", fullName, valueTypeName(value))
	}

	if err := validateValue(dataType, value); err != nil {
		return fmt.Errorf("field %q: %w", fullName, err)
	}

	return nil
}
