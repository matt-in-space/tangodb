package main

import (
	"os"

	"github.com/matt-in-space/tangodb/repl"
)

func main() {
	repl.RunREPL(os.Stdin, os.Stdout)
}
