package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/matt-in-space/tangodb/core"
)

func RunREPL(in io.Reader, out io.Writer) {
	db := core.NewDatabase("")
	scanner := bufio.NewScanner(in)
	var buffer strings.Builder

	fmt.Fprint(out, "> ")

	for scanner.Scan() {
		buffer.WriteString(scanner.Text())
		buffer.WriteString("\n")

		trimmed := strings.TrimSpace(buffer.String())

		if trimmed == "" {
			buffer.Reset()
			fmt.Fprint(out, "> ")
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
			fmt.Fprint(out, "> ")

		case errors.Is(err, core.ErrIncompleteInput):
			fmt.Fprint(out, "... ")

		default:
			fmt.Fprintf(out, "error: %v\n", err)
			buffer.Reset()
			fmt.Fprint(out, "> ")
		}
	}

	fmt.Fprintln(out)

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(out, "error reading input: %v\n", err)
	}
}
