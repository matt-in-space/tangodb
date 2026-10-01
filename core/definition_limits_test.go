package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/matt-in-space/tangodb/storage/record"
)

// nestedDefinition builds `c { l1: { l2: { ... { leaf: int } ... } } };` with
// the given number of nested blocks.
func nestedDefinition(levels int) string {
	var b strings.Builder
	b.WriteString("c { ")
	for range levels {
		b.WriteString("l: { ")
	}
	b.WriteString("leaf: int ")
	for range levels {
		b.WriteString("} ")
	}
	b.WriteString("};")
	return b.String()
}

func TestDefinitionLimits_FieldNameLength(t *testing.T) {
	d := NewDatabase("test")

	longest := strings.Repeat("n", 255)
	runStatements(t, d, `ok { `+longest+`: int };`)

	tooLong := strings.Repeat("n", 256)
	_, err := Parse(`bad { ` + tooLong + `: int };`)

	want := `field name "` + tooLong + `" is longer than 255 bytes`
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}

	if _, err := Parse(`bad { a: { ` + tooLong + `: int } };`); err == nil || err.Error() != want {
		t.Fatalf("expected the limit inside a block too, got %v", err)
	}
}

func TestDefinitionLimits_NestingDepth(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, nestedDefinition(32))

	_, err := Parse(nestedDefinition(33))

	path := strings.TrimSuffix(strings.Repeat("l.", 33), ".")
	want := `field "` + path + `" is nested more than 32 levels deep`
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestDefinitionLimits_CheckedForOperationsBuiltInGo(t *testing.T) {
	d := NewDatabase("test")

	_, err := d.Run(DefineCollectionOperation{
		Name: "bad",
		Data: map[string]DataType{strings.Repeat("n", 256): TypeInt},
	})
	if err == nil || !strings.Contains(err.Error(), "is longer than 255 bytes") {
		t.Fatalf("expected the name length error, got %v", err)
	}

	// 33 nested blocks, built directly.
	schema := &Schema{Data: map[string]DataType{"leaf": TypeInt}}
	for range 32 {
		schema = &Schema{Data: map[string]DataType{"l": TypeObject}, Objects: map[string]*Schema{"l": schema}}
	}

	_, err = d.Run(DefineCollectionOperation{
		Name:    "deep",
		Data:    map[string]DataType{"l": TypeObject},
		Objects: map[string]*Schema{"l": schema},
	})
	if err == nil || !strings.Contains(err.Error(), "is nested more than 32 levels deep") {
		t.Fatalf("expected the nesting error, got %v", err)
	}

	if _, exists := d.collections["deep"]; exists {
		t.Fatal("expected no collection to be defined")
	}
}

func TestDefinitionLimits_DeepestValidRecordIsStorable(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, nestedDefinition(32))

	var b strings.Builder
	b.WriteString(">> c {")
	for range 32 {
		b.WriteString("l: {")
	}
	b.WriteString("leaf: 1")
	for range 32 {
		b.WriteString("}")
	}
	b.WriteString("};")

	runStatements(t, d, b.String())
	expectCount(t, d, `<< c;`, "1")

	// The deepest record a schema allows must also be one the record format
	// can store.
	stored := d.collections["c"].records[0]
	encoded, err := record.Encode(stored)
	if err != nil {
		t.Fatalf("expected the deepest valid record to encode, got %v", err)
	}
	decoded, err := record.Decode(encoded)
	if err != nil || !reflect.DeepEqual(decoded, stored) {
		t.Fatalf("expected the record to round-trip, got %v", err)
	}
}
