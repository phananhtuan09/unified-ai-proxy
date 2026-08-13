package main

import (
	"os"

	"github.com/tuanp-github/unified-ai-proxy/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
