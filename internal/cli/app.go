package cli

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kamichidu/go-hariti"
)

//go:embed assets/hariti.txt
var rootUsage string

func Run(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("hariti", flag.ContinueOnError)

	var configFlag string
	var configDirFlag string
	var dataDirFlag string
	var verbose bool

	fs.StringVar(&configFlag, "config", "", "path to bundles.hariti configuration file")
	fs.StringVar(&configFlag, "c", "", "path to bundles.hariti configuration file")
	fs.StringVar(&configDirFlag, "config-dir", "", "path to configuration directory")
	fs.StringVar(&dataDirFlag, "data-dir", "", "path to data directory")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.BoolVar(&verbose, "v", false, "enable verbose output")

	fs.Usage = func() {
		//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
		fmt.Fprint(os.Stderr, rootUsage)
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	// 1. Resolve Config Dir
	var configDir string
	if configDirFlag != "" {
		configDir = configDirFlag
	} else {
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, _ := os.UserHomeDir()
			xdgConfig = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(xdgConfig, "hariti")
	}

	// 2. Resolve Config File Path
	var configFile string
	if configFlag != "" {
		configFile = configFlag
	} else if os.Getenv("HARITI_CONFIG") != "" {
		configFile = os.Getenv("HARITI_CONFIG")
	} else {
		configFile = filepath.Join(configDir, "bundles.hariti")
	}

	// 3. Resolve Data Dir
	var dataDir string
	if dataDirFlag != "" {
		dataDir = dataDirFlag
	} else {
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			home, _ := os.UserHomeDir()
			xdgData = filepath.Join(home, ".local", "share")
		}
		dataDir = filepath.Join(xdgData, "hariti")
	}

	logger := NewCLILogger(verbose)
	ctx = hariti.WithLogger(ctx, logger)

	remaining := fs.Args()
	if len(remaining) == 0 {
		fs.Usage()
		return 1
	}

	subcmd := remaining[0]
	subcmdArgs := remaining[1:]

	cliCtx := &Context{
		Context: ctx,
		Global: &GlobalFlags{
			ConfigFile: configFile,
			ConfigDir:  configDir,
			DataDir:    dataDir,
			Verbose:    verbose,
		},
		Logger: logger,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cmdMap := make(map[string]Command)
	for _, cmd := range All() {
		cmdMap[cmd.Name()] = cmd
	}

	cmd, found := cmdMap[subcmd]
	if !found {
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcmd)
		fs.Usage()
		return 1
	}

	if err := cmd.Run(cliCtx, subcmdArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}
