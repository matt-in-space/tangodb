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

	if !strings.Contains(output, "Name:user") {
		t.Fatalf("expected the parsed operation to be printed, got: %q", output)
	}

	if !strings.Contains(output, "PrimaryKey:id") {
		t.Fatalf("expected the primary key to be printed, got: %q", output)
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

	if !strings.Contains(output, "Name:user") {
		t.Fatalf("expected the parsed operation to be printed, got: %q", output)
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

	if !strings.Contains(output, "Name:user") {
		t.Fatalf("expected the REPL to recover and parse the next statement, got: %q", output)
	}
}
