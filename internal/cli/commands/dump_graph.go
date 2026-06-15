package commands

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/internal/cli"
	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

//go:embed assets/dump-graph.txt
var dumpGraphUsage string

type DumpGraphCommand struct{}

func (c *DumpGraphCommand) Name() string {
	return "dump-graph"
}

func (c *DumpGraphCommand) Run(ctx *cli.Context, args []string) error {
	fs := flag.NewFlagSet("dump-graph", flag.ContinueOnError)
	fs.SetOutput(ctx.Stderr)
	fs.Usage = func() {
		//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
		fmt.Fprint(ctx.Stderr, dumpGraphUsage)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	configFile := ctx.Global.ConfigFile
	if fs.NArg() > 0 {
		configFile = fs.Arg(0)
	}

	ext := strings.ToLower(filepath.Ext(configFile))
	if ext != ".hariti" {
		return fmt.Errorf("unsupported config format: only .hariti is supported")
	}

	g, err := dsl.LoadGraph(configFile)
	if err != nil {
		return fmt.Errorf("failed to load/convert dsl graph: %w", err)
	}

	enc := json.NewEncoder(ctx.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		return fmt.Errorf("failed to encode graph to JSON: %w", err)
	}

	return nil
}

func init() {
	cli.Register(&DumpGraphCommand{})
}
