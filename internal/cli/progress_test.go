package cli_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/cli"
)

func TestProgressReporter_OnProgress(t *testing.T) {
	var buf bytes.Buffer
	reporter := cli.NewProgressReporter(&buf)

	// Test SyncEventStarted
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type: hariti.SyncEventStarted,
	})
	if !strings.Contains(buf.String(), "Syncing repositories...") {
		t.Errorf("expected output to contain Syncing repositories..., got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventBundleStarted
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:     hariti.SyncEventBundleStarted,
		BundleID: "folke/lazy.nvim",
	})
	if !strings.Contains(buf.String(), "→ folke/lazy.nvim") {
		t.Errorf("expected output to contain → folke/lazy.nvim, got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventBundleCompleted
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:     hariti.SyncEventBundleCompleted,
		BundleID: "tpope/vim-fugitive",
		Num:      12,
		Total:    38,
	})
	if !strings.Contains(buf.String(), "[12/38] completed") || !strings.Contains(buf.String(), "✓ tpope/vim-fugitive") {
		t.Errorf("expected output to contain [12/38] completed and ✓ tpope/vim-fugitive, got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventBundleFailed
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:     hariti.SyncEventBundleFailed,
		BundleID: "nvim-treesitter/nvim-treesitter",
		Num:      13,
		Total:    38,
		Err:      errors.New("some git error"),
	})
	if !strings.Contains(buf.String(), "[13/38] failed") || !strings.Contains(buf.String(), "✗ nvim-treesitter/nvim-treesitter") {
		t.Errorf("expected output to contain [13/38] failed and ✗ nvim-treesitter/nvim-treesitter, got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventCompleted
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type: hariti.SyncEventCompleted,
	})
	if !strings.Contains(buf.String(), "Sync completed successfully.") {
		t.Errorf("expected output to contain Sync completed successfully., got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventFailed
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type: hariti.SyncEventFailed,
		Err:  errors.New("overall error"),
	})
	if !strings.Contains(buf.String(), "Sync failed: overall error") {
		t.Errorf("expected output to contain Sync failed: overall error, got: %q", buf.String())
	}
}
