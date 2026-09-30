package core

import "testing"

func TestParseDefineCollection_ParsesFieldsAndPrimaryKey(t *testing.T) {
	input := `
	user {
	  id: int @id
	  name: text
	}
	`

	o, err := ParseDefineCollection(input)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Name != "user" {
		t.Fatalf("expected name %q, got %q", "user", o.Name)
	}

	if o.PrimaryKey != "id" {
		t.Fatalf("expected primary key %q, got %q", "id", o.PrimaryKey)
	}

	want := map[string]DataType{
		"id":   TypeInt,
		"name": TypeText,
	}

	for field, dataType := range want {
		if o.Data[field] != dataType {
			t.Fatalf("expected field %q to have type %v, got %v", field, dataType, o.Data[field])
		}
	}

	if len(o.Data) != len(want) {
		t.Fatalf("expected %d fields, got %d", len(want), len(o.Data))
	}
}

func TestParseDefineCollection_AllowsNoPrimaryKey(t *testing.T) {
	o, err := ParseDefineCollection(`user { name: text }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.PrimaryKey != "" {
		t.Fatalf("expected no primary key, got %q", o.PrimaryKey)
	}
}

func TestParseDefineCollection_AllowsEmptyCollection(t *testing.T) {
	o, err := ParseDefineCollection(`user {}`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Data) != 0 {
		t.Fatalf("expected no fields, got %d", len(o.Data))
	}
}

func TestParseDefineCollection_IsCaseInsensitiveAboutTypes(t *testing.T) {
	o, err := ParseDefineCollection(`user { age: INT }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Data["age"] != TypeInt {
		t.Fatalf("expected age to be TypeInt, got %v", o.Data["age"])
	}
}

func TestParseDefineCollection_RejectsUnknownType(t *testing.T) {
	if _, err := ParseDefineCollection(`user { age: number }`); err == nil {
		t.Fatal("expected an error for an unknown type")
	}
}

func TestParseDefineCollection_RejectsUnknownAnnotation(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int @unique }`); err == nil {
		t.Fatal("expected an error for an unknown annotation")
	}
}

func TestParseDefineCollection_RejectsMultiplePrimaryKeys(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int @id name: text @id }`); err == nil {
		t.Fatal("expected an error for multiple @id fields")
	}
}

func TestParseDefineCollection_ParsesAutoIncrement(t *testing.T) {
	o, err := ParseDefineCollection(`user { id: int @id @auto }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !o.AutoFields["id"] {
		t.Fatal("expected id to be an auto field")
	}

	if o.PrimaryKey != "id" {
		t.Fatalf("expected primary key %q, got %q", "id", o.PrimaryKey)
	}
}

func TestParseDefineCollection_AllowsAutoBeforeID(t *testing.T) {
	o, err := ParseDefineCollection(`user { id: int @auto @id }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !o.AutoFields["id"] {
		t.Fatal("expected id to be an auto field")
	}
}

func TestParseDefineCollection_AllowsAutoWithoutID(t *testing.T) {
	o, err := ParseDefineCollection(`ticket { code: text @id number: int @auto }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !o.AutoFields["number"] {
		t.Fatal("expected number to be an auto field")
	}

	if o.AutoFields["code"] {
		t.Fatal("expected code not to be an auto field")
	}
}

func TestParseDefineCollection_AllowsMultipleAutoFields(t *testing.T) {
	o, err := ParseDefineCollection(`event { id: int @id @auto seq: int @auto }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !o.AutoFields["id"] || !o.AutoFields["seq"] {
		t.Fatalf("expected id and seq to be auto fields, got %v", o.AutoFields)
	}
}

func TestParseDefineCollection_RejectsOptionalAuto(t *testing.T) {
	for _, input := range []string{
		`ticket { number: int @auto @optional }`,
		`ticket { number: int @optional @auto }`,
	} {
		_, err := ParseDefineCollection(input)
		if err == nil {
			t.Fatalf("expected an error for %q", input)
		}

		want := `@auto field "number" cannot be @optional`
		if err.Error() != want {
			t.Fatalf("for %q expected error %q, got %q", input, want, err.Error())
		}
	}
}

func TestParseDefineCollection_RejectsAutoOnNonIntField(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: text @id @auto }`); err == nil {
		t.Fatal("expected an error for @auto on a non-int field")
	}
}

func TestParseDefineCollection_RejectsDuplicateAuto(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int @id @auto @auto }`); err == nil {
		t.Fatal("expected an error for a duplicate @auto annotation")
	}
}

func TestParseDefineCollection_RejectsMissingClosingBrace(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int`); err == nil {
		t.Fatal("expected an error for a missing closing brace")
	}
}

func TestParseDefineCollection_AllowsCommasBetweenFields(t *testing.T) {
	o, err := ParseDefineCollection(`user { id: int @id, name: text }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if len(o.Data) != 2 || o.Data["id"] != TypeInt || o.Data["name"] != TypeText {
		t.Fatalf("expected fields id:int and name:text, got %v", o.Data)
	}

	if o.PrimaryKey != "id" {
		t.Fatalf("expected primary key %q, got %q", "id", o.PrimaryKey)
	}
}

func TestParseDefineCollection_ToleratesTrailingSemicolon(t *testing.T) {
	o, err := ParseDefineCollection(`user { id: int };`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Name != "user" {
		t.Fatalf("expected name %q, got %q", "user", o.Name)
	}
}

func TestParseDefineCollection_RejectsTrailingInput(t *testing.T) {
	if _, err := ParseDefineCollection(`user { id: int } extra`); err == nil {
		t.Fatal("expected an error for unexpected trailing input")
	}
}

func TestParseDefineCollection_EndToEndThroughDatabase(t *testing.T) {
	o, err := ParseDefineCollection(`
	user {
	  id: int @id
	  name: text
	}
	`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	d := NewDatabase("test")

	result, err := d.Run(o)
	if err != nil {
		t.Fatalf("Failed to run parsed operation, err: %v", err)
	}

	defineResult, ok := result.(DefineCollectionResult)
	if !ok {
		t.Fatal("Collection was not returned")
	}

	if defineResult.Collection.name != "user" {
		t.Fatalf("expected collection name %q, got %q", "user", defineResult.Collection.name)
	}
}

func TestParseDefineCollection_ParsesOptional(t *testing.T) {
	o, err := ParseDefineCollection(`user { id: int @id nickname: text @optional }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !o.Optional["nickname"] {
		t.Fatal("expected nickname to be optional")
	}

	if o.Optional["id"] {
		t.Fatal("expected id to be required")
	}
}

func TestParseDefineCollection_RejectsDuplicateOptional(t *testing.T) {
	if _, err := ParseDefineCollection(`user { nickname: text @optional @optional }`); err == nil {
		t.Fatal("expected an error for a duplicate @optional annotation")
	}
}

func TestParseDefineCollection_RejectsOptionalID(t *testing.T) {
	for _, input := range []string{
		`user { id: int @id @optional }`,
		`user { id: int @optional @id }`,
		`user { id: int @id @auto @optional }`,
	} {
		_, err := ParseDefineCollection(input)
		if err == nil {
			t.Fatalf("expected an error for %q", input)
		}

		want := `@id field "id" cannot be @optional`
		if err.Error() != want {
			t.Fatalf("for %q expected error %q, got %q", input, want, err.Error())
		}
	}
}

func TestParseDefineCollection_ParsesEmbeddedBlock(t *testing.T) {
	o, err := ParseDefineCollection(`user { id: int @id address: { street: text city: text } }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if o.Data["address"] != TypeObject {
		t.Fatalf("expected address to be an object, got %v", o.Data["address"])
	}

	address := o.Objects["address"]
	if address == nil || address.Data["street"] != TypeText || address.Data["city"] != TypeText {
		t.Fatalf("expected address to hold street and city, got %+v", address)
	}
}

func TestParseDefineCollection_ParsesNestedBlocksAndOptional(t *testing.T) {
	o, err := ParseDefineCollection(`user { id: int @id address: { city: text geo: { lat: float lng: float } @optional } @optional }`)
	if err != nil {
		t.Fatalf("Failed to parse, err: %v", err)
	}

	if !o.Optional["address"] {
		t.Fatal("expected address to be optional")
	}

	address := o.Objects["address"]
	if !address.Optional["geo"] || address.Objects["geo"].Data["lat"] != TypeFloat {
		t.Fatalf("expected an optional geo block with lat, got %+v", address)
	}
}

func TestParseDefineCollection_RejectsIdentityInsideBlock(t *testing.T) {
	cases := map[string]string{
		`user { id: int @id address: { id: int @id } }`:                    `@id is not allowed inside an embedded block (field "address.id")`,
		`user { id: int @id address: { n: int @auto } }`:                   `@auto is not allowed inside an embedded block (field "address.n")`,
		`user { id: int @id a: { b: { c: int @id } } }`:                    `@id is not allowed inside an embedded block (field "a.b.c")`,
		`user { id: int @id address: { city: text } @id }`:                 `only @optional is allowed on an embedded block (field "address")`,
		`user { id: int @id address: { city: text @optional @optional } }`: `duplicate @optional annotation on field "address.city"`,
	}

	for input, want := range cases {
		_, err := ParseDefineCollection(input)
		if err == nil || err.Error() != want {
			t.Fatalf("for %q expected error %q, got %v", input, want, err)
		}
	}
}
