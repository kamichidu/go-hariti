package git

import (
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"

	"github.com/kamichidu/go-hariti"
)

type Git struct{}

func (self *Git) Clone(c *hariti.Context, bundle *hariti.RemoteBundle) error {
	var cmd *exec.Cmd
	if info, err := os.Stat(bundle.LocalPath); err != nil {
		c.Logger.Printf("Cloning %s to %s\n", bundle.URL, bundle.LocalPath)
		cmd = exec.Command("git", "clone", "--recursive", bundle.URL.String(), bundle.LocalPath)
		cmd.Stdout = c.Writer
		cmd.Stderr = c.ErrWriter
	} else if info.IsDir() && c.BoolFlag("update") {
		c.Logger.Printf("Pulling in %s", bundle.LocalPath)
		cmd = exec.Command("git", "pull", "--ff", "--ff-only")
		cmd.Dir = bundle.LocalPath
		cmd.Stdout = c.Writer
		cmd.Stderr = c.ErrWriter
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

func (self *Git) Remove(c *hariti.Context, bundle *hariti.RemoteBundle) error {
	return nil
}

func (self *Git) CanHandle(c *hariti.Context, u *url.URL) bool {
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
