package cli

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed assets/hariti.txt
var rootUsage string

func Run(ctx context.Context, args []string) int {
	cmdMap := make(map[string]Command)
	for _, cmd := range All() {
		cmdMap[cmd.Name()] = cmd
	}

	fs := flag.NewFlagSet("hariti", flag.ContinueOnError)

	global := &GlobalFlags{}
	global.Register(fs)

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
	if global.ConfigDir == "" {
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, _ := os.UserHomeDir()
			xdgConfig = filepath.Join(home, ".config")
		}
		global.ConfigDir = filepath.Join(xdgConfig, "hariti")
	}

	// 2. Resolve Config File Path
	if global.ConfigFile == "" {
		if os.Getenv("HARITI_CONFIG") != "" {
			global.ConfigFile = os.Getenv("HARITI_CONFIG")
		} else {
			global.ConfigFile = filepath.Join(global.ConfigDir, "bundles.hariti")
		}
	}

	// 3. Resolve Data Dir
	if global.DataDir == "" {
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			home, _ := os.UserHomeDir()
			xdgData = filepath.Join(home, ".local", "share")
		}
		global.DataDir = filepath.Join(xdgData, "hariti")
	}

	logger := NewCLILogger(global.Verbose)

	remaining := fs.Args()
	if len(remaining) == 0 {
		fs.Usage()
		return 1
	}

	subcmd := remaining[0]
	subcmdArgs := remaining[1:]

	cliCtx := &Context{
		Context: ctx,
		Global:  global,
		Logger:  logger,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}

	cmd, found := cmdMap[subcmd]
	if !found {
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcmd)
		fs.Usage()
		return 1
	}

	if err := cmd.Run(cliCtx, subcmdArgs); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}
