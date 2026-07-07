package commands

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/kamichidu/go-flagshim"
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

func (c *InstallCommand) RegisterFlags(ctx context.Context, fs *flagshim.FlagSet) context.Context {
	fs.Usage = func() {
		//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
		fmt.Fprint(fs.Output(), installUsage)
	}
	if global, ok := flagshim.FlagFromContext[cli.GlobalFlags](ctx); ok {
		global.Register(ctx, fs)
	}
	return ctx
}

func (c *InstallCommand) Run(ctx context.Context, args []string) error {
	global := cli.GetGlobalFlags(ctx)
	stdout := cli.GetStdout(ctx)
	stderr := cli.GetStderr(ctx)
	logger := cli.GetLogger(ctx)

	configFile := global.ConfigFile
	if len(args) > 0 {
		configFile = args[0]
	}

	g, err := dsl.LoadGraph(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse/resolve dsl: %w", err)
	}

	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: global.ConfigFile,
			ConfigDir:  global.ConfigDir,
			DataDir:    global.DataDir,
		},
		Writer:    stdout,
		ErrWriter: stderr,
		Logger:    logger,
	}
	har := hariti.NewHariti(cfg)

	return har.Install(ctx, g, hariti.InstallOptions{
		Sync:   hariti.SyncOptions{},
		Deploy: hariti.DeployOptions{},
	})
}

func init() {
	cli.Register(&InstallCommand{})
}
