package hariti

import (
	"context"
	"fmt"
	"os"

	"github.com/kamichidu/go-hariti/internal/graph"
)

type SyncOptions struct {
	Update bool
}

func (self *Hariti) Sync(ctx context.Context, g *graph.Graph, opts SyncOptions) ([]RepositoryFact, error) {
	logger := LoggerFromContextKey(ctx)
	if logger != nil {
		logger.Infof("Starting repository synchronization...")
	}
	facts := make([]RepositoryFact, 0, len(g.Bundles))

	for _, bundle := range g.Bundles {
		currentSource := getSourceString(bundle)

		if bundle.Source.Type == graph.SourceTypeLocal {
			// Local Source check
			_, err := os.Stat(bundle.Source.Path)
			if err != nil {
				return nil, fmt.Errorf("local source path does not exist for bundle %s: %w", bundle.ID, err)
			}

			facts = append(facts, RepositoryFact{
				BundleID: bundle.ID,
				Revision: "local",
			})
		} else if bundle.Source.Type == graph.SourceTypeRemote {
			// Source mismatch detection
			storedMeta, err := self.loadRepositoryMetadata(bundle.ID)
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

			vcs := DetectVCS(bundle.Source.URL)
			if vcs == nil {
				return nil, fmt.Errorf("failed to detect VCS for bundle %s with URL %s", bundle.ID, bundle.Source.URL)
			}

			// Ensure directory structure
			if err := self.SetupManagedDirectory(); err != nil {
				return nil, err
			}

			// Clone / Fetch/Pull
			err = vcs.Clone(ctx, bundle, opts.Update)
			if err != nil {
				return nil, fmt.Errorf("failed to clone/update bundle %s: %w", bundle.ID, err)
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
			if err := self.writeRepositoryMetadata(bundle.ID, meta); err != nil {
				return nil, fmt.Errorf("failed to write repository metadata for %s: %w", bundle.ID, err)
			}

			facts = append(facts, RepositoryFact{
				BundleID: bundle.ID,
				Revision: rev,
			})
		}
	}

	// Write hariti.lock
	if err := self.writeLockfile(facts, g); err != nil {
		return nil, fmt.Errorf("failed to write lockfile: %w", err)
	}

	return facts, nil
}
