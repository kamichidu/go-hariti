package hariti

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kamichidu/go-hariti/internal/graph"
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

	Logger Logger
}

func NewHariti(config *HaritiConfig) *Hariti {
	return &Hariti{config, NewStdLogger(ioutil.Discard)}
}

func (self *Hariti) SetupManagedDirectory() error {
	directories := []string{
		self.config.Directory,
		self.MetaDir(),
		self.RepositoriesDir(),
		self.MetadataDir(),
		self.GenerationsDir(),
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
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdg, "hariti", "repos")
}

func (self *Hariti) MetadataDir() string {
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdg, "hariti", "metadata")
}

func (self *Hariti) GenerationsDir() string {
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdg, "hariti", "generations")
}

func (self *Hariti) CurrentSymlinkPath() string {
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdg, "hariti", "current")
}

func (self *Hariti) LockfilePath() string {
	return filepath.Join(self.config.Directory, "hariti.lock")
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

	var appendRtp func(map[string]struct{}, string) error
	appendRtp = func(memo map[string]struct{}, name string) error {
		if _, dup := memo[name]; dup {
			return fmt.Errorf("Circular dependency: %s", name)
		}
		memo[name] = struct{}{}

		dependencies, err := self.getMetaStringSlice(name, metaDependencies)
		if err != nil {
			return err
		}
		for _, dependency := range dependencies {
			if err = appendRtp(memo, path.Base(dependency)); err != nil {
				return err
			}
		}

		pluginDir := filepath.Join(self.DeployDir(), name)
		rtp = append(rtp, pluginDir)

		// has after dir or not
		if info, err := os.Stat(filepath.Join(pluginDir, "after")); err == nil && info.IsDir() {
			afterRtp = append(afterRtp, filepath.Join(pluginDir, "after"))
		}
		return nil
	}
	enabledBundles, err := ioutil.ReadDir(self.DeployDir())
	if err != nil {
		return err
	}
	for _, info := range enabledBundles {
		if err = appendRtp(map[string]struct{}{}, info.Name()); err != nil {
			return err
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

	if bundle.Source.Type == graph.SourceTypeRemote {
		dependencies := make([]string, 0)
		for _, dep := range bundle.Dependencies {
			dependencies = append(dependencies, dep)
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

	if bundle.Source.Type == graph.SourceTypeRemote {
		removalBundle, err := self.CreateBundle(dependency)
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
		return self.addMetaVar(bundle.ID, metaDependencies, filtered)
	} else {
		return nil
	}
}

func (self *Hariti) ClearDependencies(repository string) error {
	bundle, err := self.CreateBundle(repository)
	if err != nil {
		return err
	}

	if bundle.Source.Type == graph.SourceTypeRemote {
		return self.addMetaVar(bundle.ID, metaDependencies, []string{})
	} else {
		return nil
	}
}

func (self *Hariti) Get(ctx context.Context, repository string, update bool, enabled bool) error {
	logger := LoggerFromContextKey(ctx)
	errCh := make(chan error, 1)
	go func() {
		bundle, err := self.CreateBundle(repository)
		if err != nil {
			errCh <- err
			return
		}

		// when bundle is a local bundle, no need to clone it
		if bundle.Source.Type == graph.SourceTypeRemote {
			vcs := DetectVCS(bundle.Source.URL)
			if vcs == nil {
				errCh <- fmt.Errorf("Can't detect vcs type: %s", bundle.Source.URL)
				return
			}
			ctx := context.Background()
			ctx = WithWriter(ctx, self.config.Writer)
			ctx = WithErrWriter(ctx, self.config.ErrWriter)
			ctx = WithLogger(ctx, logger)
			if err = vcs.Clone(ctx, bundle, update); err != nil {
				errCh <- err
				return
			}
		}

		if !enabled {
			errCh <- nil
			return
		}

		errCh <- self.Enable(repository)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (self *Hariti) Remove(repository string, force bool) error {
	if err := self.Disable(repository); err != nil {
		return err
	}

	bundle, err := self.CreateBundle(repository)
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
			return fmt.Errorf("Can't detect vcs type: %s", bundle.Source.URL)
		}
		ctx := context.Background()
		ctx = WithWriter(ctx, self.config.Writer)
		ctx = WithErrWriter(ctx, self.config.ErrWriter)
		ctx = WithLogger(ctx, self.Logger)
		if modified, err := vcs.IsModified(ctx, bundle); err != nil {
			return fmt.Errorf("Modification check failure: %s", err)
		} else if modified {
			return fmt.Errorf("Can't remove modified bundle %s", bundle.Source.Path)
		}
	}
	if err := os.RemoveAll(bundle.Source.Path); err != nil {
		return err
	}
	return nil
}

func (self *Hariti) List() ([]graph.Bundle, error) {
	// under the repositories dir, that's remote bundles
	children, err := ioutil.ReadDir(self.RepositoriesDir())
	if err != nil {
		return nil, err
	}
	bundles := make([]graph.Bundle, 0)
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

func (self *Hariti) CreateBundle(repository string) (graph.Bundle, error) {
	if strings.HasPrefix(repository, "file://") {
		return self.createLocalBundle(repository)
	} else if _, err := os.Stat(repository); err == nil {
		return self.createLocalBundle(repository)
	} else {
		return self.createRemoteBundle(repository)
	}
}

func (self *Hariti) createRemoteBundle(repository string) (graph.Bundle, error) {
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
	bundle.Source.Path = filepath.Join(self.RepositoriesDir(), url.QueryEscape(bundle.ID))
	if bundle.EnableIf, err = self.getMetaString(bundle.ID, metaEnableIfExpr); err != nil {
		return bundle, err
	}
	if dependencies, err := self.getMetaStringSlice(bundle.ID, metaDependencies); err != nil {
		return bundle, err
	} else {
		bundle.Dependencies = dependencies
	}
	if aliases, err := self.getMetaStringSlice(bundle.ID, metaAliases); err != nil {
		return bundle, err
	} else {
		bundle.Aliases = aliases
	}

	return bundle, nil
}

func (self *Hariti) createLocalBundle(repository string) (graph.Bundle, error) {
	var bundle graph.Bundle
	bundle.Source.Type = graph.SourceTypeLocal
	bundle.Source.Path = repository
	bundle.ID = filepath.Base(bundle.Source.Path)
	if aliases, err := self.getMetaStringSlice(bundle.ID, metaAliases); err != nil {
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

func (self *Hariti) loadRepositoryMetadata(bundleID string) (*RepositoryMetadata, error) {
	path := filepath.Join(self.MetadataDir(), url.QueryEscape(bundleID))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	meta := new(RepositoryMetadata)
	if err := json.Unmarshal(data, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func (self *Hariti) writeRepositoryMetadata(bundleID string, meta *RepositoryMetadata) error {
	path := filepath.Join(self.MetadataDir(), url.QueryEscape(bundleID))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

func (self *Hariti) writeLockfile(facts []RepositoryFact, g *graph.Graph) error {
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

	if err := os.MkdirAll(self.config.Directory, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(self.LockfilePath(), data, 0644)
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

func (self *Hariti) Sync(ctx context.Context, g *graph.Graph, update bool) ([]RepositoryFact, error) {
	logger := LoggerFromContextKey(ctx)
	if logger != nil {
		logger.Infof("Starting repository synchronization...")
	}
	facts := make([]RepositoryFact, 0, len(g.Bundles))

	for _, bundle := range g.Bundles {
		currentSource := getSourceString(bundle)

		if bundle.Source.Type == graph.SourceTypeLocal {
			// Local Source check
			_, err := os.Stat(bundle.Source.Path)
			if err != nil {
				return nil, fmt.Errorf("local source path does not exist for bundle %s: %w", bundle.ID, err)
			}

			facts = append(facts, RepositoryFact{
				BundleID: bundle.ID,
				Revision: "local",
			})
		} else if bundle.Source.Type == graph.SourceTypeRemote {
			// Source mismatch detection
			storedMeta, err := self.loadRepositoryMetadata(bundle.ID)
			if err != nil {
				return nil, err
			}

			if storedMeta != nil && storedMeta.Source != currentSource {
				// remove stale repo directory
				err := os.RemoveAll(bundle.Source.Path)
				if err != nil {
					return nil, fmt.Errorf("failed to remove repo directory on source mismatch for %s: %w", bundle.ID, err)
				}
			}

			vcs := DetectVCS(bundle.Source.URL)
			if vcs == nil {
				return nil, fmt.Errorf("failed to detect VCS for bundle %s with URL %s", bundle.ID, bundle.Source.URL)
			}

			// Ensure directory structure
			if err := self.SetupManagedDirectory(); err != nil {
				return nil, err
			}

			// Clone / Fetch/Pull
			err = vcs.Clone(ctx, bundle, update)
			if err != nil {
				return nil, fmt.Errorf("failed to clone/update bundle %s: %w", bundle.ID, err)
			}

			// Revision observation
			rev, err := vcs.HeadRevision(ctx, bundle)
			if err != nil {
				return nil, fmt.Errorf("failed to observe HEAD revision for bundle %s: %w", bundle.ID, err)
			}

			// Write repository metadata
			meta := &RepositoryMetadata{
				BundleID: bundle.ID,
				Source:   currentSource,
			}
			if err := self.writeRepositoryMetadata(bundle.ID, meta); err != nil {
				return nil, fmt.Errorf("failed to write repository metadata for %s: %w", bundle.ID, err)
			}

			facts = append(facts, RepositoryFact{
				BundleID: bundle.ID,
				Revision: rev,
			})
		}
	}

	// Write hariti.lock
	if err := self.writeLockfile(facts, g); err != nil {
		return nil, fmt.Errorf("failed to write lockfile: %w", err)
	}

	return facts, nil
}

type GenerationMetadata struct {
	ID        string    `json:"id"`
	CreatedAt string    `json:"created_at"`
	LockHash  string    `json:"lock_hash"`
}

func getExportedBundleDirName(bundleID string) string {
	return strings.ReplaceAll(bundleID, "/", "_")
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, rel)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

func matchOS(stepOS, currentOS string) bool {
	if stepOS == "all" || stepOS == "*" {
		return true
	}
	if stepOS == "mac" && currentOS == "darwin" {
		return true
	}
	return strings.ToLower(stepOS) == strings.ToLower(currentOS)
}

func (self *Hariti) Deploy(ctx context.Context, g *graph.Graph) (string, error) {
	logger := LoggerFromContextKey(ctx)
	if logger != nil {
		logger.Infof("Starting generation deployment...")
	}

	// Read project-side hariti.lock content
	lockBytes, err := ioutil.ReadFile(self.LockfilePath())
	if err != nil {
		return "", fmt.Errorf("failed to read lockfile: %w", err)
	}

	// Calculate generation semantic identity (hash of lockfile contents)
	hash := sha256.Sum256(lockBytes)
	shortHash := fmt.Sprintf("%x", hash)[:8]
	timestamp := time.Now().Format("20060102-150405")
	genID := fmt.Sprintf("%s-%s", timestamp, shortHash)

	genDir := filepath.Join(self.GenerationsDir(), genID)
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create generation directory: %w", err)
	}

	// Parse lockfile to get mapped revisions
	var lock Lockfile
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		return "", fmt.Errorf("failed to parse lockfile: %w", err)
	}

	revisionsMap := make(map[string]string)
	for _, entry := range lock.Bundles {
		revisionsMap[entry.ID] = entry.Revision
	}

	for _, bundle := range g.Bundles {
		if bundle.Source.Type == graph.SourceTypeLocal {
			// Local source: Skip folder creation and copying completely
			continue
		}

		destDir := filepath.Join(genDir, "pack", "hariti", "opt", getExportedBundleDirName(bundle.ID))
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory for bundle %s: %w", bundle.ID, err)
		}

		if bundle.Source.Type == graph.SourceTypeRemote {
			// Remote source: Export bundle from the revision in lockfile
			revision, exists := revisionsMap[bundle.ID]
			if !exists || revision == "" {
				return "", fmt.Errorf("no revision found in lockfile for remote bundle %s", bundle.ID)
			}

			// Try git archive, fallback to recursive copy if git archive is unavailable
			cmd := exec.Command("sh", "-c", fmt.Sprintf("git archive %s | tar -x -C %s", revision, destDir))
			cmd.Dir = bundle.Source.Path
			err := cmd.Run()
			if err != nil {
				// TODO: Fallback to working tree copy as per specification
				if err := copyDir(bundle.Source.Path, destDir); err != nil {
					return "", fmt.Errorf("failed to copy remote bundle %s working copy: %w", bundle.ID, err)
				}
			}
		}

		// Run build steps inside the EXPORTED bundle directory in the generation
		for _, step := range bundle.Build {
			if matchOS(step.OS, runtime.GOOS) {
				var buildCmd *exec.Cmd
				if runtime.GOOS == "windows" {
					buildCmd = exec.Command("cmd", "/c", step.Cmd)
				} else {
					buildCmd = exec.Command("sh", "-c", step.Cmd)
				}
				buildCmd.Dir = destDir
				buildCmd.Stdout = ioutil.Discard
				buildCmd.Stderr = ioutil.Discard
				_ = buildCmd.Run()
			}
		}
	}

	// Generate packadd.vim inside the generation dir
	var packaddContent strings.Builder
	for _, bundle := range g.Bundles {
		if bundle.Source.Type == graph.SourceTypeLocal {
			localPath := bundle.Source.Path
			if bundle.EnableIf != "" {
				packaddContent.WriteString(fmt.Sprintf("if %s\n  set runtimepath+=%s\n", bundle.EnableIf, localPath))
				afterPath := filepath.Join(localPath, "after")
				if _, err := os.Stat(afterPath); err == nil {
					packaddContent.WriteString(fmt.Sprintf("  set runtimepath+=%s\n", afterPath))
				}
				packaddContent.WriteString("endif\n")
			} else {
				packaddContent.WriteString(fmt.Sprintf("set runtimepath+=%s\n", localPath))
				afterPath := filepath.Join(localPath, "after")
				if _, err := os.Stat(afterPath); err == nil {
					packaddContent.WriteString(fmt.Sprintf("set runtimepath+=%s\n", afterPath))
				}
			}
		} else {
			bundleName := getExportedBundleDirName(bundle.ID)
			if bundle.EnableIf != "" {
				packaddContent.WriteString(fmt.Sprintf("if %s\n  packadd %s\nendif\n", bundle.EnableIf, bundleName))
			} else {
				packaddContent.WriteString(fmt.Sprintf("packadd %s\n", bundleName))
			}
		}
	}
	if err := ioutil.WriteFile(filepath.Join(genDir, "packadd.vim"), []byte(packaddContent.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write packadd.vim: %w", err)
	}

	// Copy hariti.lock to lock.json
	if err := ioutil.WriteFile(filepath.Join(genDir, "lock.json"), lockBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to copy lock snapshot: %w", err)
	}

	// Write metadata.json
	meta := &GenerationMetadata{
		ID:        genID,
		CreatedAt: time.Now().Format(time.RFC3339),
		LockHash:  fmt.Sprintf("%x", hash),
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize generation metadata: %w", err)
	}
	if err := ioutil.WriteFile(filepath.Join(genDir, "metadata.json"), metaBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write metadata.json: %w", err)
	}

	// Switch current symlink atomically using temporary symlink and rename
	currentPath := self.CurrentSymlinkPath()
	tempSymlink := filepath.Join(filepath.Dir(currentPath), "current.tmp")
	_ = os.Remove(tempSymlink) // remove stale temp symlink

	// Point to the relative or absolute target
	targetPath := filepath.Join("generations", genID)
	if err := os.Symlink(targetPath, tempSymlink); err != nil {
		return "", fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	if err := os.Rename(tempSymlink, currentPath); err != nil {
		// Fallback for environments where atomic rename of symlink has OS constraints
		_ = os.Remove(currentPath)
		if err := os.Symlink(targetPath, currentPath); err != nil {
			return "", fmt.Errorf("failed to switch current symlink: %w", err)
		}
	}

	return genID, nil
}
