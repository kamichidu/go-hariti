package hariti

import (
	"context"
	"io/ioutil"
	"net/url"
)

type VCS interface {
	CanHandle(c *Context, u *url.URL) bool
	Clone(c *Context, bundle *RemoteBundle) error
	Remove(c *Context, bundle *RemoteBundle) error
}

var vcsList []VCS

func RegisterVCS(vcs VCS) {
	vcsList = append(vcsList, vcs)
}

func DetectVCS(u *url.URL) VCS {
	ctx := &Context{
		Context:   context.Background(),
		Writer:    ioutil.Discard,
		ErrWriter: ioutil.Discard,
	}
	for _, vcs := range vcsList {
		if vcs.CanHandle(ctx, u) {
			return vcs
		}
	}
	return nil
}
