package hariti

import (
	"context"
	"fmt"
	"io"
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
	bundle, err := self.CreateRemoteBundle(repository)
	if err != nil {
		return err
	}

	vcs := DetectVCS(bundle.URL)
	if vcs == nil {
		return fmt.Errorf("Can't detect vcs type: %s", bundle.URL)
	}
	ctx := context.Background()
	ctx = WithWriter(ctx, self.config.Writer)
	ctx = WithErrWriter(ctx, self.config.ErrWriter)
	ctx = WithLogger(ctx, log.New(self.config.ErrWriter, "", 0x0))
	if err = vcs.Clone(ctx, bundle, update); err != nil {
		return err
	}

	if !enabled {
		return nil
	}

	return self.Enable(repository)
}

func (self *Hariti) Rm() error {
	return nil
}

func (self *Hariti) List() error {
	return nil
}

func (self *Hariti) Enable(repository string) error {
	bundle, err := self.CreateRemoteBundle(repository)
	if err != nil {
		return err
	}

	// create relative links
	filename := filepath.Join(self.config.Directory, "deploy", bundle.Name)
	relLink, err := filepath.Rel(filepath.Dir(filename), bundle.LocalPath)
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
	bundle, err := self.CreateRemoteBundle(repository)
	if err != nil {
		return err
	}

	// remove links
	filename := filepath.Join(self.config.Directory, "deploy", bundle.Name)
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

func (self *Hariti) CreateRemoteBundle(repository string) (*RemoteBundle, error) {
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

func (self *Hariti) CreateLocalBundle(repository string) (*LocalBundle, error) {
	bundle := &LocalBundle{
		LocalPath: repository,
	}
	return bundle, nil
}
