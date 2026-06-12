package commands

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

func RunSync(ctx context.Context, gOpts GlobalOptions, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	var update bool
	fs.BoolVar(&update, "update", false, "update if exists")

	if err := fs.Parse(args); err != nil {
		return err
	}

	configFile := "bundles.hariti"
	if fs.NArg() > 0 {
		configFile = fs.Arg(0)
	}

	g, err := dsl.LoadGraph(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse/resolve dsl: %w", err)
	}

	cfg := &hariti.HaritiConfig{
		Directory: gOpts.Directory,
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
	}
	har := hariti.NewHariti(cfg)

	_, err = har.Sync(ctx, g, hariti.SyncOptions{Update: update})
	return err
}
