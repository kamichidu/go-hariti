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

func run(name string, args ...string) error {
	fmt.Printf("%s %s", name, strings.Join(args, " "))
	return nil
	// var stderr bytes.Buffer
	// cmd := exec.Command(name, args...)
	// cmd.Stderr = &stderr
	// err := cmd.Run()
	// if err != nil {
	// 	log.Printf("Got an error: %s\n---\n%s\n", err, stderr.String())
	// 	return fmt.Errorf("%s\n%v", stderr.String(), err)
	// }
	// return nil
}

func (self *Git) Clone(bundle *hariti.Bundle) error {
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

func (self *Git) Remove(bundle *hariti.Bundle) error {
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
