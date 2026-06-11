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

func (self *Git) Clone(c context.Context, bundle graph.Bundle, update bool) error {
	log := hariti.LoggerFromContextKey(c)
	out := hariti.WriterFromContext(c)
	errOut := hariti.ErrWriterFromContext(c)

	return self.impl.Clone(c, bundle, update, out, errOut, log)
}

func (self *Git) IsModified(c context.Context, bundle graph.Bundle) (bool, error) {
	out := hariti.WriterFromContext(c)
	errOut := hariti.ErrWriterFromContext(c)

	return self.impl.IsModified(c, bundle, out, errOut)
}

func (self *Git) CanHandle(c context.Context, u *url.URL) bool {
	return self.impl.CanHandle(c, u.String())
}

func (self *Git) HeadRevision(c context.Context, bundle graph.Bundle) (string, error) {
	out := hariti.WriterFromContext(c)
	errOut := hariti.ErrWriterFromContext(c)

	return self.impl.HeadRevision(c, bundle, out, errOut)
}

var _ hariti.VCS = (*Git)(nil)

func init() {
	hariti.RegisterVCS(new(Git))
}
