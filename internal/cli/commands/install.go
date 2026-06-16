package commands

import (
	_ "embed"
	"flag"
	"fmt"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/cli"
	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

//go:embed assets/install.txt
var installUsage string

type InstallCommand struct{}

func (c *InstallCommand) Name() string {
	return "install"
}

func (c *InstallCommand) Run(ctx *cli.Context, args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(ctx.Stderr)
	fs.Usage = func() {
		//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
		fmt.Fprint(ctx.Stderr, installUsage)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	configFile := ctx.Global.ConfigFile
	if fs.NArg() > 0 {
		configFile = fs.Arg(0)
	}

	g, err := dsl.LoadGraph(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse/resolve dsl: %w", err)
	}

	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: ctx.Global.ConfigFile,
			ConfigDir:  ctx.Global.ConfigDir,
			DataDir:    ctx.Global.DataDir,
		},
		Writer:    ctx.Stdout,
		ErrWriter: ctx.Stderr,
	}
	har := hariti.NewHariti(cfg)

	return har.Install(ctx.Context, g, hariti.InstallOptions{
		Sync:   hariti.SyncOptions{},
		Deploy: hariti.DeployOptions{},
	})
}

func init() {
	cli.Register(&InstallCommand{})
}
