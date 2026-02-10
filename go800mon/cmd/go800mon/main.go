package main

import (
	"os"

	"go800mon/a800mon"
)

func main() {
	os.Exit(a800mon.Main(os.Args[1:]))
}
