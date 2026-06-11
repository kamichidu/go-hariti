package subcmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kamichidu/go-hariti/internal/config/yaml"
	"github.com/urfave/cli"
)

func dumpGraphAction(c *cli.Context) error {
	filePath := "bundles.yml"
	if c.NArg() > 0 {
		filePath = c.Args().First()
	}

	f, err := os.Open(filePath)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to open config file: %w", err), 1)
	}
	defer f.Close()

	bundles, err := yaml.UnmarshalBundles(f)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to unmarshal bundles: %w", err), 1)
	}

	bundlesFile := &yaml.BundlesFile{
		Version: "0.0",
		Bundles: bundles,
	}

	g, err := bundlesFile.ToGraph()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to convert/validate graph: %w", err), 1)
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
