package cli

import (
	"io"
	"os"

	"github.com/kamichidu/go-hariti"
)

type CLILogger struct {
	hariti.Logger
	verbose bool
}

func NewCLILogger(verbose bool) *CLILogger {
	writer := io.Discard
	if verbose {
		writer = os.Stdout
	}
	return &CLILogger{
		Logger:  hariti.NewStdLogger(writer),
		verbose: verbose,
	}
}
