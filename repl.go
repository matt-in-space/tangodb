package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func RunREPL(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	var buffer strings.Builder

	fmt.Fprint(out, "> ")

	for scanner.Scan() {
		buffer.WriteString(scanner.Text())
		buffer.WriteString("\n")

		if strings.TrimSpace(buffer.String()) == "" {
			buffer.Reset()
			fmt.Fprint(out, "> ")
			continue
		}

		op, err := Parse(buffer.String())

		switch {
		case err == nil:
			fmt.Fprintf(out, "%+v\n", op)
			buffer.Reset()
			fmt.Fprint(out, "> ")

		case errors.Is(err, ErrIncompleteInput):
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
