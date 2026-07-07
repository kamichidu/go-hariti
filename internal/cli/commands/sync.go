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

//go:embed assets/sync.txt
var syncUsage string

type SyncFlags struct {
	Parallelism int
}

type SyncCommand struct{}

func (c *SyncCommand) Name() string {
	return "sync"
}

func (c *SyncCommand) RegisterFlags(ctx context.Context, fs *flagshim.FlagSet) context.Context {
	fs.Usage = func() {
		//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
		fmt.Fprint(fs.Output(), syncUsage)
	}
	if global, ok := flagshim.FlagFromContext[cli.GlobalFlags](ctx); ok {
		global.Register(ctx, fs)
	}
	flags := &SyncFlags{}
	fs.IntVar(&flags.Parallelism, "parallelism", 0, "")
	fs.Alias("parallelism", "p")
	return flagshim.ContextWithFlag(ctx, flags)
}

func (c *SyncCommand) Run(ctx context.Context, args []string) error {
	global := cli.GetGlobalFlags(ctx)
	stdout := cli.GetStdout(ctx)
	stderr := cli.GetStderr(ctx)
	logger := cli.GetLogger(ctx)
	flags := flagshim.MustFlagFromContext[SyncFlags](ctx)

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

	reporter := cli.NewProgressReporter(stdout)

	_, err = har.Sync(ctx, g, hariti.SyncOptions{
		Parallelism: flags.Parallelism,
		OnProgress:  reporter.OnProgress,
	})
	return err
}

func init() {
	cli.Register(&SyncCommand{})
}
