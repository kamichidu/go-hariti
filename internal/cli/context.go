package cli

import (
	"context"
	"flag"
	"io"

	"github.com/kamichidu/go-hariti"
)

type GlobalFlags struct {
	ConfigFile string
	ConfigDir  string
	DataDir    string
	Verbose    bool
}

func (g *GlobalFlags) Register(fs *flag.FlagSet) {
	fs.StringVar(&g.ConfigFile, "config", g.ConfigFile, "")
	fs.StringVar(&g.ConfigFile, "c", g.ConfigFile, "")
	fs.StringVar(&g.ConfigDir, "config-dir", g.ConfigDir, "")
	fs.StringVar(&g.DataDir, "data-dir", g.DataDir, "")
	fs.BoolVar(&g.Verbose, "verbose", g.Verbose, "")
	fs.BoolVar(&g.Verbose, "v", g.Verbose, "")
}

type Context struct {
	context.Context

	Global *GlobalFlags
	Logger hariti.Logger
	Stdout io.Writer
	Stderr io.Writer
}
