package commands_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/kamichidu/go-flagshim"
	"github.com/kamichidu/go-hariti/internal/cli"
	"github.com/kamichidu/go-hariti/internal/cli/commands"
)

func TestRunDumpGraph_UnsupportedFormat(t *testing.T) {
	ctx := context.Background()
	global := &cli.GlobalFlags{
		ConfigFile: "/tmp/bundles.hariti",
		ConfigDir:  "/tmp",
		DataDir:    "/tmp",
		Verbose:    false,
	}
	ctx = flagshim.ContextWithFlag(ctx, global)
	ctx = flagshim.ContextWithStdout(ctx, io.Discard)
	ctx = flagshim.ContextWithStderr(ctx, io.Discard)

	cmd := &commands.DumpGraphCommand{}
	err := cmd.Run(ctx, []string{"bundles.yml"})
	if err == nil {
		t.Error("expected error for unsupported format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported config format: only .hariti is supported") {
		t.Errorf("expected error message to contain 'unsupported config format', got: %v", err)
	}
}
