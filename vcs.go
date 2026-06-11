package hariti

import (
	"context"
	"net/url"

	"github.com/kamichidu/go-hariti/internal/graph"
)

type VCS interface {
	CanHandle(c context.Context, u *url.URL) bool
	Clone(c context.Context, bundle graph.Bundle, update bool) error
	IsModified(c context.Context, bundle graph.Bundle) (bool, error)
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
