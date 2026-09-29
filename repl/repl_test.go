package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunREPL_DefinesInsertsMergesThenReadsInOneSession(t *testing.T) {
	in := strings.NewReader(
		"user { id: int @id name: text age: int }\n" +
			">> user {id: 1, name: \"Sam\", age: 40}\n" +
			">> user {id: 2, name: \"Pat\", age: 40}\n" +
			"~> user(id: 1) {name: \"Matt\"};\n" +
			"<< user => {*};\n",
	)
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "> 1\n") {
		t.Fatalf("expected the merge to report a bare count of 1, got: %q", output)
	}

	if !strings.Contains(output, "40   1   Matt") {
		t.Fatalf("expected the merged field to change while age survived, got: %q", output)
	}

	if !strings.Contains(output, "40   2   Pat") {
		t.Fatalf("expected the non-matching record to remain untouched, got: %q", output)
	}
}

func TestRunREPL_DefinesInsertsDeletesThenReadsInOneSession(t *testing.T) {
	in := strings.NewReader(
		"user { id: int @id name: text }\n" +
			">> user {id: 1, name: \"Matt\"}\n" +
			">> user {id: 2, name: \"Sam\"}\n" +
			"!> user(name: \"Matt\");\n" +
			"<< user => {*};\n",
	)
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "> 1\n") {
		t.Fatalf("expected the delete to report a bare count of 1, got: %q", output)
	}

	if !strings.Contains(output, "id  name\n2   Sam") {
		t.Fatalf("expected the final read to show the surviving record, got: %q", output)
	}

	if strings.Contains(output, "1   Matt") {
		t.Fatalf("expected the deleted record to no longer appear as a table row, got: %q", output)
	}
}

func TestRunREPL_ExitStopsTheLoop(t *testing.T) {
	in := strings.NewReader("exit\nuser { id: int @id }\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if strings.Contains(output, "id: int @id") {
		t.Fatalf("expected input after 'exit' to be ignored, got: %q", output)
	}
}

func TestRunREPL_ExitIsCaseInsensitiveAndTrimsWhitespace(t *testing.T) {
	in := strings.NewReader("  EXIT  \nuser { id: int @id }\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if strings.Contains(output, "id: int @id") {
		t.Fatalf("expected 'EXIT' with whitespace to stop the loop, got: %q", output)
	}
}

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
	in := strings.NewReader("user { id: int @id name: text }\n>> user {id: 1, name: \"Matt\"}\n")
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
	in := strings.NewReader("user { id: int @id }\n>> user {id: 1}\n>> user {id: 1}\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "error: duplicate primary key") {
		t.Fatalf("expected a duplicate primary key error, got: %q", output)
	}
}

func TestRunREPL_InsertReportsUnknownCollectionError(t *testing.T) {
	in := strings.NewReader(">> user {id: 1}\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, `error: collection "user" does not exist`) {
		t.Fatalf("expected an unknown collection error, got: %q", output)
	}
}

func TestRunREPL_DefinesInsertsThenReadsInOneSession(t *testing.T) {
	in := strings.NewReader(
		"user { id: int @id name: text }\n" +
			">> user {id: 1, name: \"Matt\"}\n" +
			">> user {id: 2, name: \"Sam\"}\n" +
			"<< user(name: \"Sam\") => {id, name}\n",
	)
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "id  name") {
		t.Fatalf("expected a table header, got: %q", output)
	}

	if !strings.Contains(output, "2   Sam") {
		t.Fatalf("expected the filtered row, got: %q", output)
	}

	if strings.Contains(output, "1   Matt") {
		t.Fatalf("expected the filter to exclude the non-matching row, got: %q", output)
	}
}

func TestRunREPL_BareReadWithSemicolonReturnsCountOnly(t *testing.T) {
	in := strings.NewReader(
		"user { id: int @id name: text }\n" +
			">> user {id: 1, name: \"Matt\"}\n" +
			">> user {id: 2, name: \"Sam\"}\n" +
			"<< user;\n",
	)
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "> 2\n") {
		t.Fatalf("expected a bare count of 2, got: %q", output)
	}

	if strings.Contains(output, "id  name") {
		t.Fatalf("did not expect a table for a bare read with no projection, got: %q", output)
	}
}

func TestRunREPL_WildcardProjectionReturnsEverythingExplicitly(t *testing.T) {
	in := strings.NewReader(
		"user { id: int @id name: text }\n" +
			">> user {id: 1, name: \"Matt\"}\n" +
			">> user {id: 2, name: \"Sam\"}\n" +
			"<< user => {*};\n",
	)
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "id  name") {
		t.Fatalf("expected a table header with both fields, got: %q", output)
	}

	if !strings.Contains(output, "1   Matt") || !strings.Contains(output, "2   Sam") {
		t.Fatalf("expected both records, got: %q", output)
	}
}

func TestRunREPL_BareReadWithoutSemicolonWaitsForMore(t *testing.T) {
	in := strings.NewReader("user { id: int @id }\n<< user\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.HasSuffix(strings.TrimRight(output, "\n"), "... ") {
		t.Fatalf("expected the REPL to still be waiting on a continuation prompt, got: %q", output)
	}
}

func TestRunREPL_ReadReportsNoRecordsFound(t *testing.T) {
	in := strings.NewReader("user { id: int @id }\n<< user() => {id}\n")
	var out bytes.Buffer

	RunREPL(in, &out)

	output := out.String()

	if !strings.Contains(output, "no records found") {
		t.Fatalf("expected \"no records found\", got: %q", output)
	}
}
