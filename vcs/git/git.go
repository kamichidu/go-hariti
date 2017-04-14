package git

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/kamichidu/go-hariti"
)

type Git struct{}

func (self *Git) Clone(bundle *hariti.RemoteBundle) error {
	if info, err := os.Stat(bundle.LocalPath); err != nil {
		log.Printf("Cloning %s to %s\n", bundle.URL, bundle.LocalPath)
		cmd := exec.Command("git", "clone", "--recursive", bundle.URL.String(), bundle.LocalPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else if info.IsDir() {
		log.Printf("Pulling in %s", bundle.LocalPath)
		cmd := exec.Command("git", "pull", "--ff", "--ff-only")
		cmd.Dir = bundle.LocalPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else {
		return fmt.Errorf("Can't work with local path: %s", bundle.LocalPath)
	}
}

func (self *Git) Remove(bundle *hariti.RemoteBundle) error {
	return nil
}

func (self *Git) CanHandle(u *url.URL) bool {
	cmd := exec.Command("git", "ls-remote", u.String())
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard

	return cmd.Run() == nil
}

func init() {
	hariti.RegisterVCS(new(Git))
}
