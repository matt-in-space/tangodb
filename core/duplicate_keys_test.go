package core

import "testing"

func TestDuplicateKeys_AreRejected(t *testing.T) {
	cases := map[string]string{
		`>> user {id: 1 name: "A" name: "B"};`:                     `field "name" is given more than once`,
		`>> user {id: 1 address: {city: "A" city: "B"}};`:          `field "address.city" is given more than once`,
		`>> user {id: 1 a: {b: {c: 1 c: 2}}};`:                     `field "a.b.c" is given more than once`,
		`>> user {id: 1 name: "A"} & {id: 2 name: "B" name: "C"};`: `field "name" is given more than once`,
		`!> user(id: 1 id: 2);`:                                    `field "id" is given more than once`,
		`<< user(address.city: "A" address.city: "B");`:            `field "address.city" is given more than once`,
		`~> user(id: 1) {name: "A" name: "B"};`:                    `field "name" is given more than once`,
		`~> user(id: 1 id: 1) {name: "A"};`:                        `field "id" is given more than once`,
		`user { id: int id: text };`:                               `field "id" is declared more than once`,
		`user { id: int @id address: { city: text city: text } };`: `field "address.city" is declared more than once`,
	}

	for input, want := range cases {
		_, err := Parse(input)
		if err == nil || err.Error() != want {
			t.Fatalf("for %q expected error %q, got %v", input, want, err)
		}
	}
}

func TestDuplicateKeys_RejectedStatementsChangeNothing(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`, `>> user {id: 1 name: "Matt"};`)

	for _, statement := range []string{
		`>> user {id: 2 name: "A" name: "B"};`,
		`!> user(id: 1 id: 1);`,
		`~> user(id: 1) {name: "A" name: "B"};`,
	} {
		if _, err := Parse(statement); err == nil {
			t.Fatalf("expected %q to be rejected", statement)
		}
	}

	expectCount(t, d, `<< user(name: "Matt");`, "1")
}

func TestDuplicateKeys_SameNameInDifferentBlocksIsFine(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d,
		`user { id: int @id name: text address: { name: text } };`,
		`>> user {id: 1 name: "Matt" address: {name: "Home"}};`,
	)

	expectCount(t, d, `<< user;`, "1")
}

func TestDuplicateKeys_ProjectionNamingAColumnTwiceStillWorks(t *testing.T) {
	d := NewDatabase("test")
	runStatements(t, d, `user { id: int @id name: text };`, `>> user {id: 1 name: "Matt"};`)

	got := readTable(t, d, `<< user => {id name id};`)

	want := "id  name\n1   Matt"
	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
