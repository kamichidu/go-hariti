package commands_test

import (
	"context"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti/internal/cli/commands"
)

func TestRunDumpGraph_UnsupportedFormat(t *testing.T) {
	ctx := context.Background()
	opts := commands.GlobalOptions{
		Directory: "/tmp",
		Verbose:   false,
	}

	err := commands.RunDumpGraph(ctx, opts, []string{"bundles.yml"})
	if err == nil {
		t.Error("expected error for unsupported format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported config format: only .hariti is supported") {
		t.Errorf("expected error message to contain 'unsupported config format', got: %v", err)
	}
}
