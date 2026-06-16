package hariti

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/kamichidu/go-hariti/graph"
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
	Logger    Logger
}

type Hariti struct {
	config *HaritiConfig

	Logger Logger
}

func NewHariti(config *HaritiConfig) *Hariti {
	logger := config.Logger
	if logger == nil {
		logger = NewStdLogger(io.Discard)
	}
	return &Hariti{config, logger}
}

func (h *Hariti) SetupManagedDirectory() error {
	directories := []string{
		h.config.Paths.ConfigDir,
		h.config.Paths.DataDir,
		h.RepositoriesDir(),
		h.MetadataDir(),
		h.GenerationsDir(),
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
