package cli

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kamichidu/go-flagshim"
)

//go:embed assets/hariti.txt
var rootUsage string

var ErrNoSubcommand = errors.New("no subcommand provided")

type RootCommand struct{}

func (c *RootCommand) Name() string {
	return "hariti"
}

func (c *RootCommand) RegisterFlags(ctx context.Context, fs *flagshim.FlagSet) context.Context {
	fs.Usage = func() {
		//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
		fmt.Fprint(fs.Output(), rootUsage)
	}
	global := &GlobalFlags{}
	global.Register(ctx, fs)
	return flagshim.ContextWithFlag(ctx, global)
}

func (c *RootCommand) PreRun(ctx context.Context, args []string) (context.Context, error) {
	global := GetGlobalFlags(ctx)

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
	ctx = ContextWithLogger(ctx, logger)

	return ctx, nil
}

func (c *RootCommand) Run(ctx context.Context, args []string) error {
	stderr := flagshim.MustStderrFromContext(ctx)
	//nolint:errcheck // safe: writing help/usage text to stderr is a presentation output; failures do not affect logic or durability
	fmt.Fprint(stderr, rootUsage)
	return ErrNoSubcommand
}

func (c *RootCommand) Commands() []flagshim.Command {
	return All()
}

func (c *RootCommand) MapError(err error) int {
	if errors.Is(err, ErrNoSubcommand) {
		return 1
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	var parseErr *flagshim.ParseError
	if errors.As(err, &parseErr) {
		if errors.Is(parseErr.Err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	return 1
}

func Run(ctx context.Context, argv []string) int {
	root := &RootCommand{}
	return flagshim.Run(ctx, os.Stdin, os.Stdout, os.Stderr, root, argv)
}
