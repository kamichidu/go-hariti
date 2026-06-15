package cli

import (
	"context"
	"io"

	"github.com/kamichidu/go-hariti"
)

type GlobalFlags struct {
	ConfigFile string
	ConfigDir  string
	DataDir    string
	Verbose    bool
}

type Context struct {
	context.Context

	Global *GlobalFlags
	Logger hariti.Logger
	Stdout io.Writer
	Stderr io.Writer
}
