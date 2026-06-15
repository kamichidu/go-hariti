package main

import (
	"context"
	"os"

	"github.com/kamichidu/go-hariti/internal/cli"
	_ "github.com/kamichidu/go-hariti/internal/cli/commands"
	_ "github.com/kamichidu/go-hariti/vcs/git"
)

func main() {
	ctx := context.Background()
	os.Exit(cli.Run(ctx, os.Args[1:]))
}
