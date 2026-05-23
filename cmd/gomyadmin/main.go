package main

import (
	"os"

	"github.com/darwvin/gomyadmin/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
