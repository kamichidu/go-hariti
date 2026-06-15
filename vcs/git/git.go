package git

import (
	"context"
	"net/url"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/graph"
	"github.com/kamichidu/go-hariti/internal/vcs"
)

type Git struct {
	impl vcs.Git
}

func (g *Git) Sync(c context.Context, bundle graph.Bundle) error {
	log := hariti.LoggerFromContextKey(c)
	out := hariti.WriterFromContext(c)
	errOut := hariti.ErrWriterFromContext(c)

	return g.impl.Sync(c, bundle, out, errOut, log)
}

func (g *Git) CanHandle(c context.Context, u *url.URL) bool {
	return g.impl.CanHandle(c, u.String())
}

func (g *Git) HeadRevision(c context.Context, bundle graph.Bundle) (string, error) {
	out := hariti.WriterFromContext(c)
	errOut := hariti.ErrWriterFromContext(c)

	return g.impl.HeadRevision(c, bundle, out, errOut)
}

func (g *Git) Archive(c context.Context, bundle graph.Bundle, revision string, destDir string) error {
	errOut := hariti.ErrWriterFromContext(c)

	return g.impl.Archive(c, bundle, revision, destDir, errOut)
}

var _ hariti.VCS = (*Git)(nil)

func init() {
	hariti.RegisterVCS(new(Git))
}
