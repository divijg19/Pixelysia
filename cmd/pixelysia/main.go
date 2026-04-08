package main

import (
	"os"

	"pixelysia/internal/pixelysia"
)

func main() {
	cli := pixelysia.NewCLI(os.Stdout, os.Stderr)
	os.Exit(cli.Run(os.Args[1:]))
}
