package cli

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/kamichidu/go-hariti"
)

type CLILogger struct {
	hariti.Logger
	verbose bool
}

func NewCLILogger(verbose bool) *CLILogger {
	var writer io.Writer = ioutil.Discard
	if verbose {
		writer = os.Stdout
	}
	return &CLILogger{
		Logger:  hariti.NewStdLogger(writer),
		verbose: verbose,
	}
}
