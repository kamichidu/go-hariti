package commands

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

func RunDeploy(ctx context.Context, gOpts GlobalOptions, args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

	configFile := gOpts.Paths.ConfigFile
	if fs.NArg() > 0 {
		configFile = fs.Arg(0)
	}

	g, err := dsl.LoadGraph(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse/resolve dsl: %w", err)
	}

	cfg := &hariti.HaritiConfig{
		Paths:     gOpts.Paths,
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
	}
	har := hariti.NewHariti(cfg)

	_, err = har.Deploy(ctx, g, hariti.DeployOptions{})
	return err
}
