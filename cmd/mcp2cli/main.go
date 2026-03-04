package main

import (
	"context"
	"os"

	"github.com/hengyunabc/mcp2cli/internal/app"
)

var version = "dev"

func main() {
	code := app.Run(context.Background(), os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr, version)
	os.Exit(code)
}
