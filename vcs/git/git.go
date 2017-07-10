package git

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"

	"github.com/kamichidu/go-hariti"
)

type Git struct{}

func (self *Git) Clone(c context.Context, bundle *hariti.RemoteBundle, update bool) error {
	log := hariti.LoggerFromContextKey(c)
	out := hariti.WriterFromContext(c)
	errOut := hariti.ErrWriterFromContext(c)

	var cmd *exec.Cmd
	if info, err := os.Stat(bundle.LocalPath); err != nil {
		log.Printf("Cloning %s to %s\n", bundle.URL, bundle.LocalPath)
		cmd = exec.Command("git", "clone", "--recursive", bundle.URL.String(), bundle.LocalPath)
		cmd.Stdout = out
		cmd.Stderr = errOut
	} else if info.IsDir() && update {
		log.Printf("Pulling in %s", bundle.LocalPath)
		cmd = exec.Command("git", "pull", "--ff", "--ff-only")
		cmd.Dir = bundle.LocalPath
		cmd.Stdout = out
		cmd.Stderr = errOut
	} else {
		return nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-c.Done():
		cmd.Process.Kill()
		return c.Err()
	case err := <-errCh:
		return err
	}
}

func (self *Git) IsModified(c context.Context, bundle *hariti.RemoteBundle) (bool, error) {
	out := hariti.WriterFromContext(c)
	errOut := hariti.ErrWriterFromContext(c)

	var cmd *exec.Cmd
	if info, err := os.Stat(bundle.LocalPath); err != nil {
		return false, fmt.Errorf("Repository %s not cloned into %s", bundle.URL, bundle.LocalPath)
	} else if !info.IsDir() {
		return false, fmt.Errorf("%s doesn't seems like a repository %s", bundle.LocalPath, bundle.URL)
	} else {
		cmd = exec.Command("git", "diff", "--exit-code")
		cmd.Dir = bundle.LocalPath
		cmd.Stdout = out
		cmd.Stderr = errOut
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-c.Done():
		cmd.Process.Kill()
		return false, c.Err()
	case err := <-errCh:
		if err != nil {
			return true, err
		} else {
			return false, nil
		}
	}
}

func (self *Git) CanHandle(c context.Context, u *url.URL) bool {
	cmd := exec.Command("git", "ls-remote", u.String())
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-c.Done():
		cmd.Process.Kill()
		return false
	case err := <-errCh:
		return err == nil
	}
}

var _ hariti.VCS = (*Git)(nil)

func init() {
	hariti.RegisterVCS(new(Git))
}
