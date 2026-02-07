package main

import (
	"os"

	"github.com/jsvensson/paletteswap/internal/lsp"
)

var version = "dev"

func main() {
	s := lsp.NewServer(version)
	if err := s.Run(); err != nil {
		os.Exit(1)
	}
}
