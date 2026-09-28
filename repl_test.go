package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunREPL_SubmitsOnlyOnceStatementIsComplete(t *testing.T) {
	in := strings.NewReader("user {\n  id: int @id\n}\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if strings.Count(output, "... ") != 2 {
		t.Fatalf("expected 2 continuation prompts, got output: %q", output)
	}

	if !strings.Contains(output, "id: int @id") {
		t.Fatalf("expected the defined collection to be printed, got: %q", output)
	}
}

func TestRunREPL_SubmitsASingleLineStatementImmediately(t *testing.T) {
	in := strings.NewReader("user { name: text }\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if strings.Contains(output, "... ") {
		t.Fatalf("did not expect a continuation prompt for a complete single-line statement, got: %q", output)
	}

	if !strings.Contains(output, "name: text") {
		t.Fatalf("expected the defined collection to be printed, got: %q", output)
	}
}

func TestRunREPL_ReportsAnErrorAndRecovers(t *testing.T) {
	in := strings.NewReader(": nonsense\nuser { name: text }\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "error:") {
		t.Fatalf("expected an error to be printed, got: %q", output)
	}

	if !strings.Contains(output, "name: text") {
		t.Fatalf("expected the REPL to recover and parse the next statement, got: %q", output)
	}
}

func TestRunREPL_DefinesThenInsertsInOneSession(t *testing.T) {
	in := strings.NewReader("user { id: int @id name: text }\n>> user => {id: 1, name: \"Matt\"}\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "id:1") {
		t.Fatalf("expected the inserted record to be printed, got: %q", output)
	}

	if !strings.Contains(output, "name:Matt") {
		t.Fatalf("expected the inserted record to be printed, got: %q", output)
	}
}

func TestRunREPL_InsertReportsDuplicateKeyError(t *testing.T) {
	in := strings.NewReader("user { id: int @id }\n>> user => {id: 1}\n>> user => {id: 1}\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "error: duplicate primary key") {
		t.Fatalf("expected a duplicate primary key error, got: %q", output)
	}
}

func TestRunREPL_InsertReportsUnknownCollectionError(t *testing.T) {
	in := strings.NewReader(">> user => {id: 1}\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, `error: collection "user" does not exist`) {
		t.Fatalf("expected an unknown collection error, got: %q", output)
	}
}
