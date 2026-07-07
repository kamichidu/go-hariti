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

	totalWidth := len(fmt.Sprintf("%d", event.Total))
	if totalWidth < 1 {
		totalWidth = 1
	}

	switch event.Type {
	case hariti.SyncEventStarted:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "Syncing repositories... (%d bundles, parallelism=%d)\n", event.Total, event.Parallelism)

	case hariti.SyncEventBundleStarted:
		status := fmt.Sprintf("%-10s", "[start]")
		progress := fmt.Sprintf("%-*s", totalWidth*2+3, "")
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "%s%s %s\n", status, progress, event.BundleID)

	case hariti.SyncEventBundleCompleted:
		status := fmt.Sprintf("%-10s", "[done]")
		progress := fmt.Sprintf("(%*d/%d)", totalWidth, event.Num, event.Total)
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "%s%s %s\n", status, progress, event.BundleID)

	case hariti.SyncEventBundleFailed:
		status := fmt.Sprintf("%-10s", "[fail]")
		progress := fmt.Sprintf("(%*d/%d)", totalWidth, event.Num, event.Total)
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "%s%s %s\n", status, progress, event.BundleID)
		if event.Output != "" {
			//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
			fmt.Fprintln(p.writer, event.Output)
		}

	case hariti.SyncEventCompleted:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "Sync completed. (%d/%d)\n", event.Total, event.Total)

	case hariti.SyncEventFailed:
		//nolint:errcheck // safe: writing progress state to terminal is presentation output; failures do not affect operational correctness
		fmt.Fprintf(p.writer, "Sync failed. (%d/%d)\n", event.Num, event.Total)
	}
}
