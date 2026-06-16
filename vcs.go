package hariti

import (
	"context"
	"net/url"

	"github.com/kamichidu/go-hariti/graph"
)

type VCS interface {
	CanHandle(c context.Context, u *url.URL) bool
	Sync(c context.Context, bundle graph.Bundle) error
	HeadRevision(c context.Context, bundle graph.Bundle) (string, error)
	Archive(c context.Context, bundle graph.Bundle, revision string, destDir string) error
}

var vcsList []VCS

func RegisterVCS(vcs VCS) {
	vcsList = append(vcsList, vcs)
}

func DetectVCS(u *url.URL) VCS {
	for _, vcs := range vcsList {
		if vcs.CanHandle(context.Background(), u) {
			return vcs
		}
	}
	return nil
}
