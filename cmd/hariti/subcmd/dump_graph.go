package subcmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/internal/config/dsl"
	"github.com/urfave/cli"
)

func dumpGraphAction(c *cli.Context) error {
	filePath := "bundles.hariti"
	if c.NArg() > 0 {
		filePath = c.Args().First()
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".hariti" {
		return cli.NewExitError("unsupported config format: only .hariti is supported", 1)
	}

	g, err := dsl.LoadGraph(filePath)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to load/convert dsl graph: %w", err), 1)
	}

	enc := json.NewEncoder(c.App.Writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		return cli.NewExitError(fmt.Errorf("failed to encode graph to JSON: %w", err), 1)
	}

	return nil
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "dump-graph",
		Usage:     "Dump the parsed Plugin Graph IR in JSON format",
		ArgsUsage: "[bundles file]",
		Action:    dumpGraphAction,
	})
}
