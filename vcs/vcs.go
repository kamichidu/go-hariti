package vcs

import (
	"context"
	"net/url"
	"sync"

	"github.com/kamichidu/go-hariti/graph"
)

type VCS interface {
	CanHandle(c context.Context, u *url.URL) bool
	Sync(c context.Context, bundle graph.Bundle) error
	HeadRevision(c context.Context, bundle graph.Bundle) (string, error)
	Archive(c context.Context, bundle graph.Bundle, revision string, destDir string) error
}

var (
	vcsMu   sync.RWMutex
	vcsList []VCS
)

func Register(vcs VCS) {
	vcsMu.Lock()
	defer vcsMu.Unlock()
	vcsList = append(vcsList, vcs)
}

func Detect(u *url.URL) VCS {
	vcsMu.RLock()
	defer vcsMu.RUnlock()
	for _, vcs := range vcsList {
		if vcs.CanHandle(context.Background(), u) {
			return vcs
		}
	}
	return nil
}
