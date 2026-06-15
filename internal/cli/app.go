package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/cli/commands"
)

func Run(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("hariti", flag.ContinueOnError)

	var configFlag string
	var configDirFlag string
	var dataDirFlag string
	var dirFlag string // deprecated -d / --directory
	var verbose bool

	fs.StringVar(&configFlag, "config", "", "path to bundles.hariti configuration file")
	fs.StringVar(&configFlag, "c", "", "path to bundles.hariti configuration file")
	fs.StringVar(&configDirFlag, "config-dir", "", "path to configuration directory")
	fs.StringVar(&dataDirFlag, "data-dir", "", "path to data directory")
	fs.StringVar(&dirFlag, "directory", "", "deprecated: use --config-dir instead")
	fs.StringVar(&dirFlag, "d", "", "deprecated: use --config-dir instead")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.BoolVar(&verbose, "v", false, "enable verbose output")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: hariti [global flags] <subcommand> [subcommand flags]\n\n")
		fmt.Fprintf(os.Stderr, "Global Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSubcommands:\n")
		fmt.Fprintf(os.Stderr, "  install       Get and install a bundle\n")
		fmt.Fprintf(os.Stderr, "  sync          Synchronize repositories and lock revisions\n")
		fmt.Fprintf(os.Stderr, "  deploy        Deploy the active generation\n")
		fmt.Fprintf(os.Stderr, "  dump-graph    Dump the resolved graph as JSON\n")
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
	} else if dirFlag != "" {
		fmt.Fprintf(os.Stderr, "Warning: --directory and -d are deprecated, please use --config-dir instead\n")
		configDir = dirFlag
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

	opts := commands.GlobalOptions{
		Paths: hariti.Paths{
			ConfigFile: configFile,
			ConfigDir:  configDir,
			DataDir:    dataDir,
		},
		Verbose: verbose,
	}

	var err error
	switch subcmd {
	case "install":
		err = commands.RunInstall(ctx, opts, subcmdArgs)
	case "sync":
		err = commands.RunSync(ctx, opts, subcmdArgs)
	case "deploy":
		err = commands.RunDeploy(ctx, opts, subcmdArgs)
	case "dump-graph":
		err = commands.RunDumpGraph(ctx, opts, subcmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcmd)
		fs.Usage()
		return 1
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}
