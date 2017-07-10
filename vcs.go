package hariti

import (
	"context"
	"net/url"
)

type VCS interface {
	CanHandle(c context.Context, u *url.URL) bool
	Clone(c context.Context, bundle *RemoteBundle, update bool) error
	Remove(c context.Context, bundle *RemoteBundle) error
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
