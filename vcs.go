package hariti

import (
	"net/url"
)

type VCS interface {
	CanHandle(u *url.URL) bool
	Clone(bundle *RemoteBundle) error
	Remove(bundle *RemoteBundle) error
}

var vcsList []VCS

func RegisterVCS(vcs VCS) {
	vcsList = append(vcsList, vcs)
}

func DetectVCS(u *url.URL) VCS {
	for _, vcs := range vcsList {
		if vcs.CanHandle(u) {
			return vcs
		}
	}
	return nil
}
