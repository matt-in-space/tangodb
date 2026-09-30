package core

import "testing"

func TestCollection_String(t *testing.T) {
	c := Collection{
		name: "user",
		data: map[string]DataType{
			"id":   TypeInt,
			"name": TypeText,
		},
		primaryKey: "id",
	}

	want := "user {\n  id: int @id\n  name: text\n}"

	if got := c.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestCollection_StringWithNoPrimaryKey(t *testing.T) {
	c := Collection{
		name: "user",
		data: map[string]DataType{
			"name": TypeText,
		},
	}

	want := "user {\n  name: text\n}"

	if got := c.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestCollection_StringWithAutoIncrement(t *testing.T) {
	c := Collection{
		name:         "user",
		data:         map[string]DataType{"id": TypeInt},
		primaryKey:   "id",
		autoCounters: map[string]int64{"id": 1},
	}

	want := "user {\n  id: int @id @auto\n}"

	if got := c.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestCollection_StringWithNoFields(t *testing.T) {
	c := Collection{name: "user", data: map[string]DataType{}}

	want := "user {\n}"

	if got := c.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestCollection_StringWithAutoOnNonIDField(t *testing.T) {
	c := Collection{
		name: "ticket",
		data: map[string]DataType{
			"code":   TypeText,
			"number": TypeInt,
		},
		primaryKey:   "code",
		autoCounters: map[string]int64{"number": 1},
	}

	want := "ticket {\n  code: text @id\n  number: int @auto\n}"

	if got := c.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestCollection_StringWithOptional(t *testing.T) {
	c := Collection{
		name: "user",
		data: map[string]DataType{
			"id":       TypeInt,
			"nickname": TypeText,
		},
		primaryKey: "id",
		optional:   map[string]bool{"nickname": true},
	}

	want := "user {\n  id: int @id\n  nickname: text @optional\n}"

	if got := c.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestCollection_StringWithEmbeddedBlocks(t *testing.T) {
	d := NewDatabase("test")
	op, err := Parse(`user { id: int @id address: { street: text geo: { lat: float } @optional } @optional };`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	result, err := d.Run(op)
	if err != nil {
		t.Fatalf("Failed to define collection, err: %v", err)
	}

	want := "user {\n" +
		"  address: {\n" +
		"    geo: {\n" +
		"      lat: float\n" +
		"    } @optional\n" +
		"    street: text\n" +
		"  } @optional\n" +
		"  id: int @id\n" +
		"}"

	if got := result.(DefineCollectionResult).String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
