package hariti

import (
	"context"
	"fmt"
	"os"

	"github.com/kamichidu/go-hariti/graph"
	"github.com/kamichidu/go-hariti/vcs"
	"golang.org/x/sync/errgroup"
)

type SyncOptions struct {
	Parallelism int
}

func (h *Hariti) Sync(ctx context.Context, g *graph.Graph, opts SyncOptions) ([]RepositoryFact, error) {
	rg := h.newRuntimeGraph(g)
	h.logger.Infof("sync started")

	parallelism := opts.Parallelism
	if parallelism <= 0 {
		parallelism = 8
	}

	sem := make(chan struct{}, parallelism)
	facts := make([]RepositoryFact, len(rg.bundles))

	eg, egCtx := errgroup.WithContext(ctx)

	for i, bundle := range rg.bundles {
		i, bundle := i, bundle
		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-egCtx.Done():
				return egCtx.Err()
			}
			defer func() {
				<-sem
			}()

			currentSource := getSourceString(bundle)

			switch bundle.Source.Type {
			case graph.SourceTypeLocal:
				h.logger.Debugf("local bundle %s check: source path %s exists", bundle.ID, bundle.Source.Path)
				// Local Source check
				_, err := os.Stat(bundle.Source.Path)
				if err != nil {
					return fmt.Errorf("local source path does not exist for bundle %s: %w", bundle.ID, err)
				}

				facts[i] = RepositoryFact{
					BundleID: bundle.ID,
					Revision: "local",
				}
			case graph.SourceTypeRemote:
				h.logger.Debugf("resolved repository path for bundle %s to %s", bundle.ID, bundle.Source.Path)
				// Source mismatch detection
				storedMeta, err := h.loadRepositoryMetadata(bundle.ID)
				if err != nil {
					return err
				}

				if storedMeta != nil && storedMeta.Source != currentSource {
					h.logger.Infof("source mismatch detected for bundle %s, removing stale repository directory", bundle.ID)
					// remove stale repo directory
					err := os.RemoveAll(bundle.Source.Path)
					if err != nil {
						return fmt.Errorf("failed to remove repo directory on source mismatch for %s: %w", bundle.ID, err)
					}
				}

				v := vcs.Detect(bundle.Source.URL)
				if v == nil {
					return fmt.Errorf("failed to detect VCS for bundle %s with URL %s", bundle.ID, bundle.Source.URL)
				}
				h.logger.Debugf("detected VCS adapter for URL: %s", bundle.Source.URL)

				// Ensure directory structure
				if err := h.SetupManagedDirectory(); err != nil {
					return err
				}

				// Clone / Fetch/Pull
				vcsCtx := vcs.WithLogger(egCtx, h.logger)
				vcsCtx = vcs.WithWriter(vcsCtx, h.config.Writer)
				vcsCtx = vcs.WithErrWriter(vcsCtx, h.config.ErrWriter)
				err = v.Sync(vcsCtx, bundle)
				if err != nil {
					return fmt.Errorf("failed to sync bundle %s: %w", bundle.ID, err)
				}

				// Revision observation
				rev, err := v.HeadRevision(vcsCtx, bundle)
				if err != nil {
					return fmt.Errorf("failed to observe HEAD revision for bundle %s: %w", bundle.ID, err)
				}
				h.logger.Debugf("resolved repository revision for bundle %s to %s", bundle.ID, rev)

				// Write repository metadata
				meta := &RepositoryMetadata{
					BundleID: bundle.ID,
					Source:   currentSource,
				}
				if err := h.writeRepositoryMetadata(bundle.ID, meta); err != nil {
					return fmt.Errorf("failed to write repository metadata for %s: %w", bundle.ID, err)
				}

				facts[i] = RepositoryFact{
					BundleID: bundle.ID,
					Revision: rev,
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// Write hariti.lock
	if err := h.writeLockfile(facts, g); err != nil {
		return nil, fmt.Errorf("failed to write lockfile: %w", err)
	}

	h.logger.Infof("sync completed")
	return facts, nil
}
