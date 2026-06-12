package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

func RunDumpGraph(ctx context.Context, gOpts GlobalOptions, args []string) error {
	fs := flag.NewFlagSet("dump-graph", flag.ContinueOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

	configFile := "bundles.hariti"
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		return fmt.Errorf("failed to encode graph to JSON: %w", err)
	}

	return nil
}
