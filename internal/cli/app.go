package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/cli/commands"
)

func Run(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("hariti", flag.ContinueOnError)

	var dir string
	var verbose bool
	fs.StringVar(&dir, "directory", "", "directory managed by hariti")
	fs.StringVar(&dir, "d", "", "directory managed by hariti")
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

	if dir == "" {
		dir = os.Getenv("HARITI_HOME")
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = fmt.Sprintf("%s/.hariti", home)
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
		Directory: dir,
		Verbose:   verbose,
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
