package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/matt-in-space/tangodb/core"
)

// The prompts avoid the query language's operator characters, so a prompt
// never reads as part of the statement (`tango> >> user {...}`). The
// continuation prompt is padded to the same width so multi-line input lines up.
const (
	prompt             = "tango> "
	continuationPrompt = "  ...> "
)

func RunREPL(in io.Reader, out io.Writer) {
	db := core.NewDatabase("")
	scanner := bufio.NewScanner(in)
	var buffer strings.Builder

	fmt.Fprint(out, prompt)

	for scanner.Scan() {
		buffer.WriteString(scanner.Text())
		buffer.WriteString("\n")

		trimmed := strings.TrimSpace(buffer.String())

		if trimmed == "" {
			buffer.Reset()
			fmt.Fprint(out, prompt)
			continue
		}

		if strings.ToLower(trimmed) == "exit" {
			break
		}

		op, err := core.Parse(buffer.String())

		switch {
		case err == nil:
			result, runErr := db.Run(op)
			if runErr != nil {
				fmt.Fprintf(out, "error: %v\n", runErr)
			} else {
				fmt.Fprintf(out, "%v\n", result)
			}
			buffer.Reset()
			fmt.Fprint(out, prompt)

		case errors.Is(err, core.ErrIncompleteInput):
			fmt.Fprint(out, continuationPrompt)

		default:
			fmt.Fprintf(out, "error: %v\n", err)
			buffer.Reset()
			fmt.Fprint(out, prompt)
		}
	}

	fmt.Fprintln(out)

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(out, "error reading input: %v\n", err)
	}
}
