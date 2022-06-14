package main

import (
	"os"

	"github.com/froov/froov/froov"
)

// this will build a directory .froov inside the source directory (add to .gitignore)
// then serve the result website.

func main() {
	//compile(".", false)
	argv := "."
	if len(os.Args) > 1 {
		argv = os.Args[1]
	}
	froov.Compile(argv, false)
	//serve(argv + "/froov")
}
