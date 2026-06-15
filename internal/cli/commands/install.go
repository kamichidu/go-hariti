package commands

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/kamichidu/go-hariti"
)

func RunInstall(ctx context.Context, gOpts GlobalOptions, args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	var update bool
	var enabled bool
	fs.BoolVar(&update, "update", false, "update if exists")
	fs.BoolVar(&enabled, "enabled", true, "enable bundle after getting")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return errors.New("missing repository argument for install command")
	}
	repo := fs.Arg(0)

	cfg := &hariti.HaritiConfig{
		Paths:     gOpts.Paths,
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
	}
	har := hariti.NewHariti(cfg)

	return har.Install(ctx, hariti.InstallOptions{
		Repository: repo,
		Update:     update,
		Enabled:    enabled,
	})
}
