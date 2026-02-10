package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jsvensson/paletteswap/internal/lsp"
)

var version = "dev"

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&showVersion, "v", false, "Print version and exit (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	s := lsp.NewServer(version)
	if err := s.Run(); err != nil {
		os.Exit(1)
	}
}
