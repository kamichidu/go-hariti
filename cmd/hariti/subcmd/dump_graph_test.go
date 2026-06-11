package subcmd_test

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti/cmd/hariti/subcmd"
	"github.com/urfave/cli"
)

func TestDumpGraph_UnsupportedFormat(t *testing.T) {
	app := cli.NewApp()
	app.Writer = new(bytes.Buffer)
	app.Commands = subcmd.Commands

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = set.Parse([]string{"bundles.yml"})
	c := cli.NewContext(app, set, nil)
	c.Command = *app.Command("dump-graph")

	err := app.Command("dump-graph").Action.(func(*cli.Context) error)(c)
	if err == nil {
		t.Error("expected error for unsupported format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported config format: only .hariti is supported") {
		t.Errorf("expected error message to contain 'unsupported config format', got: %v", err)
	}
}
