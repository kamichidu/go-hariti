package subcmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/internal/config/dsl"
	"github.com/kamichidu/go-hariti/internal/config/yaml"
	"github.com/kamichidu/go-hariti/internal/graph"
	"github.com/urfave/cli"
)

func dumpGraphAction(c *cli.Context) error {
	filePath := "bundles.yml"
	if c.NArg() > 0 {
		filePath = c.Args().First()
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	var g *graph.Graph
	var err error

	if ext == ".hariti" {
		src, rErr := ioutil.ReadFile(filePath)
		if rErr != nil {
			return cli.NewExitError(fmt.Errorf("failed to read dsl file: %w", rErr), 1)
		}
		g, err = dsl.ParseGraph(filePath, src)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed to parse/convert dsl: %w", err), 1)
		}
	} else {
		f, rErr := os.Open(filePath)
		if rErr != nil {
			return cli.NewExitError(fmt.Errorf("failed to open config file: %w", rErr), 1)
		}
		defer f.Close()

		bundles, uErr := yaml.UnmarshalBundles(f)
		if uErr != nil {
			return cli.NewExitError(fmt.Errorf("failed to unmarshal bundles: %w", uErr), 1)
		}

		bundlesFile := &yaml.BundlesFile{
			Version: "0.0",
			Bundles: bundles,
		}

		g, err = bundlesFile.ToGraph()
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed to convert/validate graph: %w", err), 1)
		}
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
