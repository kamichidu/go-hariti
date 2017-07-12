package hariti

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	metaEnableIfExpr = "enableIf"
	metaDependencies = "dependencies"
	metaAliases      = "aliases"
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
	// |  + meta/
	directories := []string{
		self.config.Directory,
		self.MetaDir(),
		self.RepositoriesDir(),
		self.DeployDir(),
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

func (self *Hariti) MetaDir() string {
	return filepath.Join(self.config.Directory, "meta")
}

func (self *Hariti) DeployDir() string {
	return filepath.Join(self.config.Directory, "deploy")
}

func (self *Hariti) RepositoriesDir() string {
	return filepath.Join(self.config.Directory, "repositories")
}

func (self *Hariti) WriteScript(w io.Writer, header []string) error {
	// write given header lines
	for _, line := range header {
		fmt.Fprintln(w, line)
	}

	rtp, afterRtp, err := self.vimNativeRuntimeDirs()
	if err != nil {
		return err
	}

	enabledBundles, err := ioutil.ReadDir(self.DeployDir())
	if err != nil {
		return err
	}
	for _, info := range enabledBundles {
		pluginDir := filepath.Join(self.DeployDir(), info.Name())
		rtp = append(rtp, pluginDir)

		// has after dir or not
		if info, err := os.Stat(filepath.Join(pluginDir, "after")); err == nil && info.IsDir() {
			afterRtp = append(afterRtp, filepath.Join(pluginDir, "after"))
		}
	}

	// generate vim script
	fmt.Fprintln(w, "set runtimepath=")
	for _, path := range append(rtp, afterRtp...) {
		enableIfExpr, err := self.getMetaString(filepath.Base(path), metaEnableIfExpr)
		if err != nil {
			return err
		}

		var prefix string
		if enableIfExpr != "" {
			fmt.Fprintf(w, "if %s\n", enableIfExpr)
			prefix = "  "
		}
		fmt.Fprintf(w, "%sset runtimepath+=%s\n", prefix, path)
		if enableIfExpr != "" {
			fmt.Fprintln(w, "endif")
		}
	}
	return nil
}

func (self *Hariti) vimNativeRuntimeDirs() (rtp []string, afterRtp []string, err error) {
	buf := new(bytes.Buffer)

	// vim -u NONE -i NONE -n -N --cmd "echo &runtimepath" --cmd "q!" 3>&1 1>&2 2>&3 3>&-
	cmd := exec.Command("vim", "--not-a-term", "-N", "-n", "--noplugin", "-i", "NONE", "-u", "NONE", "-U", "NONE", "--cmd", "echo &runtimepath", "--cmd", "q!")
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return nil, nil, err
	}
	paths := strings.Split(strings.TrimSpace(buf.String()), ",")
	rtp = make([]string, 0)
	afterRtp = make([]string, 0)
	for _, path := range paths {
		if filepath.Base(path) == "after" {
			afterRtp = append(afterRtp, path)
		} else {
			rtp = append(rtp, path)
		}
	}
	return rtp, afterRtp, nil
}

func (self *Hariti) IsEnabled(bundle Bundle) bool {
	if _, err := os.Stat(filepath.Join(self.DeployDir(), bundle.GetName())); err != nil {
		// not found in deploy dir
		return false
	} else {
		// found in deploy dir
		return true
	}
}

func (self *Hariti) AddAlias(repository string, alias string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	return self.addMetaVar(bundle.GetName(), metaAliases, append(bundle.GetAliases(), alias))
}

func (self *Hariti) RemoveAlias(repository string, alias string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	filtered := make([]string, 0)
	for _, other := range bundle.GetAliases() {
		if other != alias {
			filtered = append(filtered, other)
		}
	}
	return self.addMetaVar(bundle.GetName(), metaAliases, filtered)
}

func (self *Hariti) ClearAlias(repository string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	return self.addMetaVar(bundle.GetName(), metaAliases, []string{})
}

func (self *Hariti) AddDependency(repository string, dependency string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	if rbundle, ok := bundle.(*RemoteBundle); ok {
		dependencies := make([]string, 0)
		for _, dep := range rbundle.Dependencies {
			dependencies = append(dependencies, dep.URL.String())
		}
		return self.addMetaVar(bundle.GetName(), metaDependencies, append(dependencies, dependency))
	} else {
		return nil
	}
}

func (self *Hariti) RemoveDependency(repository string, dependency string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	if rbundle, ok := bundle.(*RemoteBundle); ok {
		removalBundle, err := self.CreateBundle(dependency)
		if err != nil {
			return err
		}
		removalRBundle, ok := removalBundle.(*RemoteBundle)
		if !ok {
			return nil
		}

		filtered := make([]string, 0)
		for _, dep := range rbundle.Dependencies {
			if dep.URL.String() != removalRBundle.URL.String() {
				filtered = append(filtered, dep.URL.String())
			}
		}
		return self.addMetaVar(rbundle.Name, metaDependencies, filtered)
	} else {
		return nil
	}
}

func (self *Hariti) ClearDependencies(repository string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	if rbundle, ok := bundle.(*RemoteBundle); ok {
		return self.addMetaVar(rbundle.Name, metaDependencies, []string{})
	} else {
		return nil
	}
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
	children, err := ioutil.ReadDir(self.RepositoriesDir())
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
	children, err = ioutil.ReadDir(self.DeployDir())
	if err != nil {
		return bundles, err
	}
	for _, child := range children {
		evalPath, err := filepath.EvalSymlinks(filepath.Join(self.DeployDir(), child.Name()))
		if err != nil {
			return bundles, err
		}
		if filepath.HasPrefix(evalPath, self.RepositoriesDir()) {
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

	// create link
	return mklink(
		bundle.GetLocalPath(),
		filepath.Join(self.DeployDir(), bundle.GetName()),
	)
}

func (self *Hariti) EnableIf(repository string, expr string) error {
	// add meta data
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}
	if err := self.addMetaVar(bundle.GetName(), metaEnableIfExpr, expr); err != nil {
		return err
	}
	return self.Enable(repository)
}

func (self *Hariti) addMetaVar(name string, key string, value interface{}) error {
	rw, err := os.OpenFile(filepath.Join(self.MetaDir(), name), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer rw.Close()

	meta := make(map[string]interface{})
	if err := json.NewDecoder(rw).Decode(&meta); err != nil && err != io.EOF {
		return err
	}
	if _, err = rw.Seek(0, os.SEEK_SET); err != nil {
		return err
	}
	if err = rw.Truncate(0); err != nil {
		return err
	}
	meta[key] = value
	if err := json.NewEncoder(rw).Encode(meta); err != nil {
		return err
	}
	return nil
}

func (self *Hariti) getMetaVar(name string, key string) (interface{}, error) {
	filename := filepath.Join(self.MetaDir(), name)
	if _, err := os.Stat(filename); err != nil {
		return nil, nil
	}

	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	meta := make(map[string]interface{})
	if err := json.NewDecoder(r).Decode(&meta); err != nil {
		return nil, err
	}
	if v, ok := meta[key]; ok {
		return v, nil
	} else {
		return nil, nil
	}
}

func (self *Hariti) getMetaString(name string, key string) (string, error) {
	value, err := self.getMetaVar(name, key)
	if err != nil {
		return "", err
	}

	if s, ok := value.(string); ok {
		return s, nil
	} else {
		return "", nil
	}
}
func (self *Hariti) getMetaStringSlice(name string, key string) ([]string, error) {
	value, err := self.getMetaVar(name, key)
	if err != nil {
		return nil, err
	}

	ss := make([]string, 0)
	if items, ok := value.([]interface{}); ok {
		for _, item := range items {
			if s, ok := item.(string); ok {
				ss = append(ss, s)
			}
		}
	}
	return ss, nil
}

func (self *Hariti) Disable(repository string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	// remove links
	filename := filepath.Join(self.DeployDir(), bundle.GetName())
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
	bundle.LocalPath = filepath.Join(self.RepositoriesDir(), url.QueryEscape(bundle.URL.String()))
	if bundle.EnableIfExpr, err = self.getMetaString(bundle.Name, metaEnableIfExpr); err != nil {
		return nil, err
	}
	if dependencies, err := self.getMetaStringSlice(bundle.Name, metaDependencies); err != nil {
		return nil, err
	} else {
		for _, dependency := range dependencies {
			depBundle, err := self.createRemoteBundle(dependency)
			if err != nil {
				return nil, err
			}
			bundle.Dependencies = append(bundle.Dependencies, depBundle)
		}
	}
	if aliases, err := self.getMetaStringSlice(bundle.Name, metaAliases); err != nil {
		return nil, err
	} else {
		bundle.Aliases = aliases
	}

	return bundle, nil
}

func (self *Hariti) createLocalBundle(repository string) (*LocalBundle, error) {
	bundle := &LocalBundle{
		LocalPath: repository,
		Aliases:   make([]string, 0),
	}
	bundle.Name = filepath.Base(bundle.LocalPath)
	if aliases, err := self.getMetaStringSlice(bundle.Name, metaAliases); err != nil {
		return nil, err
	} else {
		bundle.Aliases = aliases
	}
	return bundle, nil
}
