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
		Type:        hariti.SyncEventStarted,
		Total:       61,
		Parallelism: 8,
	})
	if !strings.Contains(buf.String(), "Syncing repositories... (61 bundles, parallelism=8)") {
		t.Errorf("expected output to contain Syncing repositories... (61 bundles, parallelism=8), got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventBundleStarted
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:     hariti.SyncEventBundleStarted,
		BundleID: "folke/lazy.nvim",
		Total:    61,
	})
	if !strings.Contains(buf.String(), "[start]") || !strings.Contains(buf.String(), "folke/lazy.nvim") {
		t.Errorf("expected output to contain [start] and folke/lazy.nvim, got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventBundleCompleted
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:     hariti.SyncEventBundleCompleted,
		BundleID: "tpope/vim-fugitive",
		Num:      12,
		Total:    61,
	})
	if !strings.Contains(buf.String(), "[done]") || !strings.Contains(buf.String(), "(12/61)") || !strings.Contains(buf.String(), "tpope/vim-fugitive") {
		t.Errorf("expected output to contain [done], (12/61), and tpope/vim-fugitive, got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventBundleFailed
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:     hariti.SyncEventBundleFailed,
		BundleID: "nvim-treesitter/nvim-treesitter",
		Num:      13,
		Total:    61,
		Err:      errors.New("some git error"),
		Output:   "error detail log",
	})
	if !strings.Contains(buf.String(), "[fail]") || !strings.Contains(buf.String(), "(13/61)") || !strings.Contains(buf.String(), "nvim-treesitter/nvim-treesitter") || !strings.Contains(buf.String(), "error detail log") {
		t.Errorf("expected output to contain [fail], (13/61), nvim-treesitter, and log detail, got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventCompleted
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:  hariti.SyncEventCompleted,
		Total: 61,
	})
	if !strings.Contains(buf.String(), "Sync completed. (61/61)") {
		t.Errorf("expected output to contain Sync completed. (61/61), got: %q", buf.String())
	}
	buf.Reset()

	// Test SyncEventFailed
	reporter.OnProgress(hariti.SyncProgressEvent{
		Type:  hariti.SyncEventFailed,
		Num:   13,
		Total: 61,
		Err:   errors.New("overall error"),
	})
	if !strings.Contains(buf.String(), "Sync failed. (13/61)") {
		t.Errorf("expected output to contain Sync failed. (13/61), got: %q", buf.String())
	}
}
