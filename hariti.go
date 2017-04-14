package hariti

import (
	"context"
	"fmt"
	"github.com/kr/pretty"
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
}

type Hariti struct {
	config *HaritiConfig
}

func NewHariti(config *HaritiConfig) *Hariti {
	return &Hariti{config}
}

func (self *Hariti) SetupEnv() error {
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

func (self *Hariti) Get(repository string, updateFlag bool) error {
	bundle, err := self.CreateRemoteBundle(repository)
	if err != nil {
		return err
	}
	pretty.Printf("bundle = %# v\n", bundle)

	vcs := DetectVCS(bundle.URL)
	if vcs == nil {
		return fmt.Errorf("Can't detect vcs type: %s", bundle.URL)
	}
	ctx := &Context{
		Context:   context.Background(),
		Writer:    self.config.Writer,
		ErrWriter: self.config.ErrWriter,
		Logger:    log.New(self.config.ErrWriter, "", 0x0),
	}
	ctx.SetFlag("update", updateFlag)
	ctx.SetFlag("verbose", false)
	if err = vcs.Clone(ctx, bundle); err != nil {
		return err
	}

	// create relative links
	filename := filepath.Join(self.config.Directory, "deploy", bundle.Name)
	relLink, err := filepath.Rel(filepath.Dir(filename), bundle.LocalPath)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(filename); err != nil {
		// there's no link, just create new one
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
	}

	return nil
}

func (self *Hariti) Rm() error {
	return nil
}

func (self *Hariti) CreateRemoteBundle(repository string) (*RemoteBundle, error) {
	var err error

	bundle := &RemoteBundle{
		Aliases:      make([]string, 0),
		Dependencies: make([]Bundle, 0),
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
		Name:         filepath.Base(repository),
		LocalPath:    repository,
		Aliases:      make([]string, 0),
		Dependencies: make([]Bundle, 0),
	}
	return bundle, nil
}
