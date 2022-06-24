package main

import (
	"fmt"
	"os"

	"github.com/froov/froov/froov"
)

// this will build a directory .froov inside the source directory (add to .gitignore)
// then serve the result website.

func main() {
	//compile(".", false)
	argv := "."
	//argv = "/Users/jimhurd/dev/ironwood/shop500/ironshop"
	if len(os.Args) > 1 {
		argv = os.Args[1]
	}
	fmt.Printf("Compiling %s", argv)
	froov.Compile(argv, false)
	//serve(argv + "/froov")
}
