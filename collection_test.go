package main

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

func TestCollection_StringWithNoFields(t *testing.T) {
	c := Collection{name: "user", data: map[string]DataType{}}

	want := "user {\n}"

	if got := c.String(); got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
