package vcs

import (
	"context"
	_ "github.com/kamichidu/go-hariti"
)

type VCS interface {
	CanHandle() bool
	Install(ctx context.Context) error
	Update(ctx context.Context) error
	Uninstall(ctx context.Context) error
}

var vcsImpls []VCS

func RegisterVCS(impl VCS) error {
	vcsImpls = append(vcsImpls, impl)
	return nil
}

func Install(url string) error {
}
