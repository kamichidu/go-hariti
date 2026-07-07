package cli

import (
	"fmt"
	"io"
	"sync"

	"github.com/kamichidu/go-hariti"
)

type ProgressReporter struct {
	mu     sync.Mutex
	writer io.Writer
}

func NewProgressReporter(w io.Writer) *ProgressReporter {
	return &ProgressReporter{
		writer: w,
	}
}

func (p *ProgressReporter) OnProgress(event hariti.SyncProgressEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch event.Type {
	case hariti.SyncEventStarted:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintln(p.writer, "Syncing repositories...")
	case hariti.SyncEventBundleStarted:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "→ %s\n", event.BundleID)
	case hariti.SyncEventBundleCompleted:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "[%d/%d] completed\n✓ %s\n", event.Num, event.Total, event.BundleID)
	case hariti.SyncEventBundleFailed:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "[%d/%d] failed\n✗ %s: %v\n", event.Num, event.Total, event.BundleID, event.Err)
	case hariti.SyncEventCompleted:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintln(p.writer, "Sync completed successfully.")
	case hariti.SyncEventFailed:
		if event.Output != "" {
			//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
			fmt.Fprintf(p.writer, "%s\n", event.Output)
		}
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "Sync failed: %v\n", event.Err)
	}
}
