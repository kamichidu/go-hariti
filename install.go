package hariti

import (
	"context"
	"fmt"

	"github.com/kamichidu/go-hariti/internal/graph"
)

type InstallOptions struct {
	Repository string
	Update     bool
	Enabled    bool
}

func (self *Hariti) Install(ctx context.Context, opts InstallOptions) error {
	logger := LoggerFromContextKey(ctx)
	errCh := make(chan error, 1)
	go func() {
		bundle, err := self.CreateBundle(opts.Repository)
		if err != nil {
			errCh <- err
			return
		}

		if bundle.Source.Type == graph.SourceTypeRemote {
			vcs := DetectVCS(bundle.Source.URL)
			if vcs == nil {
				errCh <- fmt.Errorf("Can't detect vcs type: %s", bundle.Source.URL)
				return
			}
			ctx := context.Background()
			ctx = WithWriter(ctx, self.config.Writer)
			ctx = WithErrWriter(ctx, self.config.ErrWriter)
			ctx = WithLogger(ctx, logger)
			if err = vcs.Clone(ctx, bundle, opts.Update); err != nil {
				errCh <- err
				return
			}
		}

		if !opts.Enabled {
			errCh <- nil
			return
		}

		errCh <- self.Enable(opts.Repository)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}
