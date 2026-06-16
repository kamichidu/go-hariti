package hariti

import (
	"context"
	"fmt"
	"os"

	"github.com/kamichidu/go-hariti/graph"
	"github.com/kamichidu/go-hariti/vcs"
)

type SyncOptions struct {
}

func (h *Hariti) Sync(ctx context.Context, g *graph.Graph, opts SyncOptions) ([]RepositoryFact, error) {
	logger := LoggerFromContextKey(ctx)
	logger.Infof("Starting repository synchronization...")
	facts := make([]RepositoryFact, 0, len(g.Bundles))

	for _, bundle := range g.Bundles {
		currentSource := getSourceString(bundle)

		switch bundle.Source.Type {
		case graph.SourceTypeLocal:
			// Local Source check
			_, err := os.Stat(bundle.Source.Path)
			if err != nil {
				return nil, fmt.Errorf("local source path does not exist for bundle %s: %w", bundle.ID, err)
			}

			facts = append(facts, RepositoryFact{
				BundleID: bundle.ID,
				Revision: "local",
			})
		case graph.SourceTypeRemote:
			// Source mismatch detection
			storedMeta, err := h.loadRepositoryMetadata(bundle.ID)
			if err != nil {
				return nil, err
			}

			if storedMeta != nil && storedMeta.Source != currentSource {
				// remove stale repo directory
				err := os.RemoveAll(bundle.Source.Path)
				if err != nil {
					return nil, fmt.Errorf("failed to remove repo directory on source mismatch for %s: %w", bundle.ID, err)
				}
			}

			vcs := vcs.Detect(bundle.Source.URL)
			if vcs == nil {
				return nil, fmt.Errorf("failed to detect VCS for bundle %s with URL %s", bundle.ID, bundle.Source.URL)
			}

			// Ensure directory structure
			if err := h.SetupManagedDirectory(); err != nil {
				return nil, err
			}

			// Clone / Fetch/Pull
			err = vcs.Sync(ctx, bundle)
			if err != nil {
				return nil, fmt.Errorf("failed to sync bundle %s: %w", bundle.ID, err)
			}

			// Revision observation
			rev, err := vcs.HeadRevision(ctx, bundle)
			if err != nil {
				return nil, fmt.Errorf("failed to observe HEAD revision for bundle %s: %w", bundle.ID, err)
			}

			// Write repository metadata
			meta := &RepositoryMetadata{
				BundleID: bundle.ID,
				Source:   currentSource,
			}
			if err := h.writeRepositoryMetadata(bundle.ID, meta); err != nil {
				return nil, fmt.Errorf("failed to write repository metadata for %s: %w", bundle.ID, err)
			}

			facts = append(facts, RepositoryFact{
				BundleID: bundle.ID,
				Revision: rev,
			})
		}
	}

	// Write hariti.lock
	if err := h.writeLockfile(facts, g); err != nil {
		return nil, fmt.Errorf("failed to write lockfile: %w", err)
	}

	return facts, nil
}
