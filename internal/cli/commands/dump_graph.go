package commands

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-flagshim"
	"github.com/kamichidu/go-hariti/internal/cli"
	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

//go:embed assets/dump-graph.txt
var dumpGraphUsage string

type DumpGraphCommand struct{}

func (c *DumpGraphCommand) Name() string {
	return "dump-graph"
}

func (c *DumpGraphCommand) RegisterFlags(ctx context.Context, fs *flagshim.FlagSet) context.Context {
	fs.Usage = func() {
		//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
		fmt.Fprint(fs.Output(), dumpGraphUsage)
	}
	if global, ok := flagshim.FlagFromContext[cli.GlobalFlags](ctx); ok {
		global.Register(ctx, fs)
	}
	return ctx
}

func (c *DumpGraphCommand) Run(ctx context.Context, args []string) error {
	global := cli.GetGlobalFlags(ctx)
	stdout := cli.GetStdout(ctx)

	configFile := global.ConfigFile
	if len(args) > 0 {
		configFile = args[0]
	}

	ext := strings.ToLower(filepath.Ext(configFile))
	if ext != ".hariti" {
		return fmt.Errorf("unsupported config format: only .hariti is supported")
	}

	g, err := dsl.LoadGraph(configFile)
	if err != nil {
		return fmt.Errorf("failed to load/convert dsl graph: %w", err)
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		return fmt.Errorf("failed to encode graph to JSON: %w", err)
	}

	return nil
}

func init() {
	cli.Register(&DumpGraphCommand{})
}
