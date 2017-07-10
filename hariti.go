package hariti

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type HaritiConfig struct {
	Directory string
	Writer    io.Writer
	ErrWriter io.Writer
	Verbose   bool
}

type Hariti struct {
	config *HaritiConfig
}

func NewHariti(config *HaritiConfig) *Hariti {
	return &Hariti{config}
}

func (self *Hariti) SetupManagedDirectory() error {
	// + {dir}/
	// |  + repositories/
	// |  | + repo/
	// |  | + repo/
	// |  + deploy/
	// |  | - link
	// |  | - link
	directories := []string{
		self.config.Directory,
		filepath.Join(self.config.Directory, "repositories"),
		filepath.Join(self.config.Directory, "deploy"),
	}
	for _, directory := range directories {
		if info, err := os.Stat(directory); err != nil {
			if err = os.MkdirAll(directory, 0755); err != nil {
				return err
			}
		} else if !info.IsDir() {
			return fmt.Errorf("It's looks like a file: %s", directory)
		}
	}
	return nil
}

func (self *Hariti) Get(repository string, update bool, enabled bool) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	// when bundle is a local bundle, no need to clone it
	if rbundle, ok := bundle.(*RemoteBundle); ok {
		vcs := DetectVCS(rbundle.URL)
		if vcs == nil {
			return fmt.Errorf("Can't detect vcs type: %s", rbundle.URL)
		}
		ctx := context.Background()
		ctx = WithWriter(ctx, self.config.Writer)
		ctx = WithErrWriter(ctx, self.config.ErrWriter)
		ctx = WithLogger(ctx, log.New(self.config.ErrWriter, "", 0x0))
		if err = vcs.Clone(ctx, rbundle, update); err != nil {
			return err
		}
	}

	if !enabled {
		return nil
	}

	return self.Enable(repository)
}

func (self *Hariti) Remove(repository string, force bool) error {
	if err := self.Disable(repository); err != nil {
		return err
	}

	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	// when bundle is a local bundle, we never delete an original
	rbundle, ok := bundle.(*RemoteBundle)
	if !ok {
		return nil
	}

	if !force {
		// check repository modified
		vcs := DetectVCS(rbundle.URL)
		if vcs == nil {
			return fmt.Errorf("Can't detect vcs type: %s", rbundle.URL)
		}
		ctx := context.Background()
		ctx = WithWriter(ctx, self.config.Writer)
		ctx = WithErrWriter(ctx, self.config.ErrWriter)
		ctx = WithLogger(ctx, log.New(self.config.ErrWriter, "", 0x0))
		if modified, err := vcs.IsModified(ctx, rbundle); err != nil {
			return fmt.Errorf("Modification check failure: %s", err)
		} else if modified {
			return fmt.Errorf("Can't remove modified bundle %s", rbundle.LocalPath)
		}
	}
	if err := os.RemoveAll(rbundle.LocalPath); err != nil {
		return err
	}
	return nil
}

func (self *Hariti) List() ([]Bundle, error) {
	// under the repositories dir, that's remote bundles
	children, err := ioutil.ReadDir(filepath.Join(self.config.Directory, "repositories"))
	if err != nil {
		return nil, err
	}
	bundles := make([]Bundle, 0)
	for _, child := range children {
		u, err := url.QueryUnescape(child.Name())
		if err != nil {
			return bundles, err
		}
		if bundle, err := self.createRemoteBundle(u); err != nil {
			return bundles, err
		} else {
			bundles = append(bundles, bundle)
		}
	}
	// under the deploy dir and not pointed to repositories dir, that's local bundles
	children, err = ioutil.ReadDir(filepath.Join(self.config.Directory, "deploy"))
	if err != nil {
		return bundles, err
	}
	for _, child := range children {
		evalPath, err := filepath.EvalSymlinks(filepath.Join(self.config.Directory, "deploy", child.Name()))
		if err != nil {
			return bundles, err
		}
		if filepath.HasPrefix(evalPath, filepath.Join(self.config.Directory, "repositories")) {
			continue
		}
		if bundle, err := self.createLocalBundle(evalPath); err != nil {
			return bundles, err
		} else {
			bundles = append(bundles, bundle)
		}
	}
	return bundles, nil
}

func (self *Hariti) Enable(repository string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	// create relative links
	filename := filepath.Join(self.config.Directory, "deploy", bundle.GetName())
	relLink, err := filepath.Rel(filepath.Dir(filename), bundle.GetLocalPath())
	if err != nil {
		return err
	}
	if info, err := os.Lstat(filename); err != nil {
		// there's no file, just create new one
		if err = os.Symlink(relLink, filename); err != nil {
			return err
		}
	} else if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		// there's a link, just check its state
		state, err := os.Readlink(filename)
		if err != nil {
			return err
		} else if state != relLink {
			return fmt.Errorf("%s should be point to %s, but %s", filename, relLink, state)
		}
	} else {
		// there's non-link file
		return fmt.Errorf("%s is already exists, ignored", filename)
	}
	return nil
}

func (self *Hariti) Disable(repository string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	// remove links
	filename := filepath.Join(self.config.Directory, "deploy", bundle.GetName())
	if info, err := os.Lstat(filename); err != nil {
		// there's no file, just ignore it
	} else if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		// there's a link, delete it
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("Can't remove symlink: %s", err)
		}
	} else {
		// there's non-link file
		return fmt.Errorf("%s is not a symlink, ignore it", filename)
	}
	return nil
}

func (self *Hariti) CreateBundle(repository string) (Bundle, error) {
	if strings.HasPrefix(repository, "file://") {
		return self.createLocalBundle(repository)
	} else if _, err := os.Stat(repository); err == nil {
		return self.createLocalBundle(repository)
	} else {
		return self.createRemoteBundle(repository)
	}
}

func (self *Hariti) createRemoteBundle(repository string) (*RemoteBundle, error) {
	var err error

	bundle := &RemoteBundle{
		Aliases:      make([]string, 0),
		Dependencies: make([]*RemoteBundle, 0),
	}
	if strings.HasPrefix(repository, "https://") || strings.HasPrefix(repository, "http://") {
		// fqdn like "https://github.com/kamichidu/vim-hariti"
		bundle.URL, err = url.ParseRequestURI(repository)
		if err != nil {
			return nil, err
		}
	} else if matched, err := path.Match("*/*", repository); matched || err != nil {
		// generally form like "kamichidu/vim-hariti"
		if err != nil {
			// program error
			panic(err)
		}
		bundle.URL, err = url.ParseRequestURI("https://" + path.Join("github.com", repository))
		if err != nil {
			return nil, err
		}
	} else {
		// shortest form like "vim-hariti"
		bundle.URL, err = url.ParseRequestURI("https://" + path.Join("github.com", "vim-scripts", repository))
		if err != nil {
			return nil, err
		}
	}

	bundle.Name = path.Base(bundle.URL.String())
	bundle.LocalPath = filepath.Join(self.config.Directory, "repositories", url.QueryEscape(bundle.URL.String()))

	return bundle, nil
}

func (self *Hariti) createLocalBundle(repository string) (*LocalBundle, error) {
	bundle := &LocalBundle{
		LocalPath: repository,
	}
	return bundle, nil
}
