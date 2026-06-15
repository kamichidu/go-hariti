package hariti

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/internal/graph"
)

const (
	metaEnableIfExpr = "enableIf"
	metaDependencies = "dependencies"
	metaAliases      = "aliases"
)

type Paths struct {
	ConfigFile string
	ConfigDir  string
	DataDir    string
}

type HaritiConfig struct {
	Paths     Paths
	Writer    io.Writer
	ErrWriter io.Writer
	Verbose   bool
}

type Hariti struct {
	config *HaritiConfig

	Logger Logger
}

func NewHariti(config *HaritiConfig) *Hariti {
	return &Hariti{config, NewStdLogger(io.Discard)}
}

func (h *Hariti) SetupManagedDirectory() error {
	directories := []string{
		h.config.Paths.ConfigDir,
		h.config.Paths.DataDir,
		h.MetaDir(),
		h.RepositoriesDir(),
		h.MetadataDir(),
		h.GenerationsDir(),
		h.DeployDir(),
	}
	for _, directory := range directories {
		if info, err := os.Stat(directory); err != nil {
			if err = os.MkdirAll(directory, 0755); err != nil {
				return err
			}
		} else if !info.IsDir() {
			return fmt.Errorf("it's looks like a file: %s", directory)
		}
	}
	return nil
}

func (h *Hariti) ConfigDir() string {
	return h.config.Paths.ConfigDir
}

func (h *Hariti) DataDir() string {
	return h.config.Paths.DataDir
}

func (h *Hariti) MetaDir() string {
	return filepath.Join(h.config.Paths.ConfigDir, "meta")
}

func (h *Hariti) DeployDir() string {
	return filepath.Join(h.config.Paths.ConfigDir, "deploy")
}

func (h *Hariti) RepositoriesDir() string {
	return filepath.Join(h.config.Paths.DataDir, "repos")
}

func (h *Hariti) MetadataDir() string {
	return filepath.Join(h.config.Paths.DataDir, "metadata")
}

func (h *Hariti) GenerationsDir() string {
	return filepath.Join(h.config.Paths.DataDir, "generations")
}

func (h *Hariti) CurrentSymlinkPath() string {
	return filepath.Join(h.config.Paths.DataDir, "current")
}

func (h *Hariti) LockfilePath() string {
	return filepath.Join(h.config.Paths.ConfigDir, "hariti.lock")
}

func (h *Hariti) WriteScript(w io.Writer, header []string) error {
	// write given header lines
	for _, line := range header {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	rtp, afterRtp, err := h.vimNativeRuntimeDirs()
	if err != nil {
		return err
	}

	var appendRtp func(map[string]struct{}, string) error
	appendRtp = func(memo map[string]struct{}, name string) error {
		if _, dup := memo[name]; dup {
			return fmt.Errorf("circular dependency: %s", name)
		}
		memo[name] = struct{}{}

		dependencies, err := h.getMetaStringSlice(name, metaDependencies)
		if err != nil {
			return err
		}
		for _, dependency := range dependencies {
			if err = appendRtp(memo, path.Base(dependency)); err != nil {
				return err
			}
		}

		pluginDir := filepath.Join(h.DeployDir(), name)
		rtp = append(rtp, pluginDir)

		// has after dir or not
		if info, err := os.Stat(filepath.Join(pluginDir, "after")); err == nil && info.IsDir() {
			afterRtp = append(afterRtp, filepath.Join(pluginDir, "after"))
		}
		return nil
	}
	enabledBundles, err := os.ReadDir(h.DeployDir())
	if err != nil {
		return err
	}
	for _, info := range enabledBundles {
		if err = appendRtp(map[string]struct{}{}, info.Name()); err != nil {
			return err
		}
	}

	// generate vim script
	if _, err := fmt.Fprintln(w, "set runtimepath="); err != nil {
		return err
	}
	for _, path := range append(rtp, afterRtp...) {
		enableIfExpr, err := h.getMetaString(filepath.Base(path), metaEnableIfExpr)
		if err != nil {
			return err
		}

		var prefix string
		if enableIfExpr != "" {
			if _, err := fmt.Fprintf(w, "if %s\n", enableIfExpr); err != nil {
				return err
			}
			prefix = "  "
		}
		if _, err := fmt.Fprintf(w, "%sset runtimepath+=%s\n", prefix, path); err != nil {
			return err
		}
		if enableIfExpr != "" {
			if _, err := fmt.Fprintln(w, "endif"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Hariti) vimNativeRuntimeDirs() (rtp []string, afterRtp []string, err error) {
	buf := new(bytes.Buffer)

	cmd := exec.Command("vim", "--not-a-term", "-N", "-n", "--noplugin", "-i", "NONE", "-u", "NONE", "-U", "NONE", "--cmd", "echo &runtimepath", "--cmd", "q!")
	cmd.Stdout = io.Discard
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

func (h *Hariti) IsEnabled(bundle Bundle) bool {
	if _, err := os.Stat(filepath.Join(h.DeployDir(), bundle.GetName())); err != nil {
		// not found in deploy dir
		return false
	} else {
		// found in deploy dir
		return true
	}
}

func (h *Hariti) AddAlias(repository string, alias string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	return h.addMetaVar(bundle.GetName(), metaAliases, append(bundle.GetAliases(), alias))
}

func (h *Hariti) RemoveAlias(repository string, alias string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	filtered := make([]string, 0)
	for _, other := range bundle.GetAliases() {
		if other != alias {
			filtered = append(filtered, other)
		}
	}
	return h.addMetaVar(bundle.GetName(), metaAliases, filtered)
}

func (h *Hariti) ClearAlias(repository string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	return h.addMetaVar(bundle.GetName(), metaAliases, []string{})
}

func (h *Hariti) AddDependency(repository string, dependency string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	if bundle.Source.Type == graph.SourceTypeRemote {
		dependencies := make([]string, 0)
		dependencies = append(dependencies, bundle.Dependencies...)
		return h.addMetaVar(bundle.GetName(), metaDependencies, append(dependencies, dependency))
	} else {
		return nil
	}
}

func (h *Hariti) RemoveDependency(repository string, dependency string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	if bundle.Source.Type == graph.SourceTypeRemote {
		removalBundle, err := h.CreateBundle(dependency)
		if err != nil {
			return err
		}
		if removalBundle.Source.Type != graph.SourceTypeRemote {
			return nil
		}

		filtered := make([]string, 0)
		for _, dep := range bundle.Dependencies {
			if dep != removalBundle.ID {
				filtered = append(filtered, dep)
			}
		}
		return h.addMetaVar(bundle.ID, metaDependencies, filtered)
	} else {
		return nil
	}
}

func (h *Hariti) ClearDependencies(repository string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	if bundle.Source.Type == graph.SourceTypeRemote {
		return h.addMetaVar(bundle.ID, metaDependencies, []string{})
	} else {
		return nil
	}
}

func (h *Hariti) Remove(repository string, force bool) error {
	if err := h.Disable(repository); err != nil {
		return err
	}

	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	if bundle.Source.Type != graph.SourceTypeRemote {
		return nil
	}

	if !force {
		// check repository modified
		vcs := DetectVCS(bundle.Source.URL)
		if vcs == nil {
			return fmt.Errorf("can't detect vcs type: %s", bundle.Source.URL)
		}
		ctx := context.Background()
		ctx = WithWriter(ctx, h.config.Writer)
		ctx = WithErrWriter(ctx, h.config.ErrWriter)
		ctx = WithLogger(ctx, h.Logger)
		if modified, err := vcs.IsModified(ctx, bundle); err != nil {
			return fmt.Errorf("modification check failure: %s", err)
		} else if modified {
			return fmt.Errorf("can't remove modified bundle %s", bundle.Source.Path)
		}
	}
	if err := os.RemoveAll(bundle.Source.Path); err != nil {
		return err
	}
	return nil
}

func (h *Hariti) List() ([]graph.Bundle, error) {
	// under the repositories dir, that's remote bundles
	children, err := os.ReadDir(h.RepositoriesDir())
	if err != nil {
		return nil, err
	}
	bundles := make([]graph.Bundle, 0)
	for _, child := range children {
		u, err := url.QueryUnescape(child.Name())
		if err != nil {
			return bundles, err
		}
		if bundle, err := h.createRemoteBundle(u); err != nil {
			return bundles, err
		} else {
			bundles = append(bundles, bundle)
		}
	}
	// under the deploy dir and not pointed to repositories dir, that's local bundles
	children, err = os.ReadDir(h.DeployDir())
	if err != nil {
		return bundles, err
	}
	for _, child := range children {
		evalPath, err := filepath.EvalSymlinks(filepath.Join(h.DeployDir(), child.Name()))
		if err != nil {
			return bundles, err
		}
		rel, err := filepath.Rel(h.RepositoriesDir(), evalPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			continue
		}
		if bundle, err := h.createLocalBundle(evalPath); err != nil {
			return bundles, err
		} else {
			bundles = append(bundles, bundle)
		}
	}
	return bundles, nil
}

func (h *Hariti) Enable(repository string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	// create link
	return mklink(
		bundle.GetLocalPath(),
		filepath.Join(h.DeployDir(), bundle.GetName()),
	)
}

func (h *Hariti) EnableIf(repository string, expr string) error {
	// add meta data
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}
	if err := h.addMetaVar(bundle.GetName(), metaEnableIfExpr, expr); err != nil {
		return err
	}
	return h.Enable(repository)
}

func (h *Hariti) addMetaVar(name string, key string, value interface{}) error {
	rw, err := os.OpenFile(filepath.Join(h.MetaDir(), name), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	//nolint:errcheck // safe: metadata file is synchronized on write; write-time errors are checked explicitly
	defer rw.Close()

	meta := make(map[string]interface{})
	if err := json.NewDecoder(rw).Decode(&meta); err != nil && err != io.EOF {
		return err
	}
	if _, err = rw.Seek(0, io.SeekStart); err != nil {
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

func (h *Hariti) getMetaVar(name string, key string) (interface{}, error) {
	filename := filepath.Join(h.MetaDir(), name)
	if _, err := os.Stat(filename); err != nil {
		return nil, nil
	}

	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck // safe: r is read-only; closing error cannot affect file integrity or durability
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

func (h *Hariti) getMetaString(name string, key string) (string, error) {
	value, err := h.getMetaVar(name, key)
	if err != nil {
		return "", err
	}

	if s, ok := value.(string); ok {
		return s, nil
	} else {
		return "", nil
	}
}
func (h *Hariti) getMetaStringSlice(name string, key string) ([]string, error) {
	value, err := h.getMetaVar(name, key)
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

func (h *Hariti) Disable(repository string) error {
	bundle, err := h.CreateBundle(repository)
	if err != nil {
		return err
	}

	// remove links
	filename := filepath.Join(h.DeployDir(), bundle.GetName())
	if info, err := os.Lstat(filename); err != nil {
		// there's no file, just ignore it
	} else if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		// there's a link, delete it
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("can't remove symlink: %s", err)
		}
	} else {
		// there's non-link file
		return fmt.Errorf("%s is not a symlink, ignore it", filename)
	}
	return nil
}

func (h *Hariti) CreateBundle(repository string) (graph.Bundle, error) {
	if strings.HasPrefix(repository, "file://") {
		return h.createLocalBundle(repository)
	} else if _, err := os.Stat(repository); err == nil {
		return h.createLocalBundle(repository)
	} else {
		return h.createRemoteBundle(repository)
	}
}

func (h *Hariti) createRemoteBundle(repository string) (graph.Bundle, error) {
	var err error

	var bundle graph.Bundle
	bundle.Source.Type = graph.SourceTypeRemote

	var parsedURL *url.URL
	if strings.HasPrefix(repository, "https://") || strings.HasPrefix(repository, "http://") {
		// fqdn like "https://github.com/kamichidu/vim-hariti"
		parsedURL, err = url.ParseRequestURI(repository)
		if err != nil {
			return bundle, err
		}
	} else if matched, err := path.Match("*/*", repository); matched || err != nil {
		if err != nil {
			// program error
			panic(err)
		}
		parsedURL, err = url.ParseRequestURI("https://" + path.Join("github.com", repository))
		if err != nil {
			return bundle, err
		}
	} else {
		// shortest form like "vim-hariti"
		parsedURL, err = url.ParseRequestURI("https://" + path.Join("github.com", "vim-scripts", repository))
		if err != nil {
			return bundle, err
		}
	}

	bundle.ID = path.Base(parsedURL.String())
	bundle.Source.URL = parsedURL
	bundle.Source.Path = filepath.Join(h.RepositoriesDir(), url.QueryEscape(bundle.ID))
	if bundle.EnableIf, err = h.getMetaString(bundle.ID, metaEnableIfExpr); err != nil {
		return bundle, err
	}
	if dependencies, err := h.getMetaStringSlice(bundle.ID, metaDependencies); err != nil {
		return bundle, err
	} else {
		bundle.Dependencies = dependencies
	}
	if aliases, err := h.getMetaStringSlice(bundle.ID, metaAliases); err != nil {
		return bundle, err
	} else {
		bundle.Aliases = aliases
	}

	return bundle, nil
}

func (h *Hariti) createLocalBundle(repository string) (graph.Bundle, error) {
	var bundle graph.Bundle
	bundle.Source.Type = graph.SourceTypeLocal
	bundle.Source.Path = repository
	bundle.ID = filepath.Base(bundle.Source.Path)
	if aliases, err := h.getMetaStringSlice(bundle.ID, metaAliases); err != nil {
		return bundle, err
	} else {
		bundle.Aliases = aliases
	}
	return bundle, nil
}

type RepositoryFact struct {
	BundleID string
	Revision string
}

type RepositoryMetadata struct {
	BundleID string `json:"bundle_id"`
	Source   string `json:"source"`
}

type Lockfile struct {
	Bundles []LockfileEntry `json:"bundles"`
}

type LockfileEntry struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Revision string `json:"revision"`
}

func (h *Hariti) loadRepositoryMetadata(bundleID string) (*RepositoryMetadata, error) {
	path := filepath.Join(h.MetadataDir(), url.QueryEscape(bundleID))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	meta := new(RepositoryMetadata)
	if err := json.Unmarshal(data, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func (h *Hariti) writeRepositoryMetadata(bundleID string, meta *RepositoryMetadata) error {
	path := filepath.Join(h.MetadataDir(), url.QueryEscape(bundleID))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (h *Hariti) writeLockfile(facts []RepositoryFact, g *graph.Graph) error {
	lock := Lockfile{
		Bundles: make([]LockfileEntry, 0, len(facts)),
	}

	factsMap := make(map[string]string)
	for _, f := range facts {
		factsMap[f.BundleID] = f.Revision
	}

	for _, bundle := range g.Bundles {
		sourceExpr := ""
		if bundle.Source.Type == graph.SourceTypeLocal {
			sourceExpr = bundle.Source.Path
		} else if bundle.Source.URL != nil {
			sourceExpr = bundle.Source.URL.String()
		}

		lock.Bundles = append(lock.Bundles, LockfileEntry{
			ID:       bundle.ID,
			Source:   sourceExpr,
			Revision: factsMap[bundle.ID],
		})
	}

	data, err := json.MarshalIndent(&lock, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(h.ConfigDir(), 0755); err != nil {
		return err
	}

	return os.WriteFile(h.LockfilePath(), data, 0644)
}

func getSourceString(b graph.Bundle) string {
	if b.Source.Type == graph.SourceTypeLocal {
		return b.Source.Path
	}
	if b.Source.URL != nil {
		return b.Source.URL.String()
	}
	return ""
}
