package hariti_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/graph"
	_ "github.com/kamichidu/go-hariti/vcs/git"
)

func runGitCmdInDir(t *testing.T, dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func TestHariti_Sync_LocalAndRemoteAndMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// 1. Setup mock Git remote
	remoteRepoDir := filepath.Join(tmpDir, "remote_repo")
	if err := os.MkdirAll(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create mock remote dir: %v", err)
	}

	_ = runGitCmdInDir(t, remoteRepoDir, "init")
	_ = runGitCmdInDir(t, remoteRepoDir, "config", "user.email", "test@hariti.io")
	_ = runGitCmdInDir(t, remoteRepoDir, "config", "user.name", "Test Hariti")
	_ = runGitCmdInDir(t, remoteRepoDir, "commit", "--allow-empty", "-m", "initial commit")
	expectedRevision := runGitCmdInDir(t, remoteRepoDir, "rev-parse", "HEAD")

	// 2. Setup local git repo
	localGitDir := filepath.Join(tmpDir, "local_git_repo")
	if err := os.MkdirAll(localGitDir, 0755); err != nil {
		t.Fatalf("failed to create local git repo dir: %v", err)
	}
	_ = runGitCmdInDir(t, localGitDir, "init")

	remoteURL, err := url.Parse("file://" + filepath.ToSlash(remoteRepoDir))
	if err != nil {
		t.Fatalf("failed to parse remote URL: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my-local-git-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: localGitDir,
				},
			},
			{
				ID: "my-remote-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-remote-plugin")),
				},
			},
		},
	}

	cfg := &hariti.HaritiConfig{
		Directory: filepath.Join(tmpDir, "hariti_home"),
		Writer:    ioutil.Discard,
		ErrWriter: ioutil.Discard,
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = hariti.WithWriter(ctx, ioutil.Discard)
	ctx = hariti.WithErrWriter(ctx, ioutil.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(ioutil.Discard))

	facts, err := har.Sync(ctx, g, false)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify observed local Git revision
	if facts[0].Revision != "local" {
		t.Errorf("expected local Git revision 'local', got '%s'", facts[0].Revision)
	}

	// Verify observed remote Git revision
	if facts[1].Revision != expectedRevision {
		t.Errorf("expected remote Git revision '%s', got '%s'", expectedRevision, facts[1].Revision)
	}

	// 3. Verify Repository Metadata was written to $XDG_DATA_HOME/hariti/metadata/
	metaPath := filepath.Join(xdgHome, "hariti", "metadata", url.QueryEscape("my-remote-plugin"))
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected metadata file to exist, got error: %v", err)
	}

	metaBytes, err := ioutil.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
	}

	var meta hariti.RepositoryMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if meta.BundleID != "my-remote-plugin" {
		t.Errorf("expected metadata bundle ID 'my-remote-plugin', got '%s'", meta.BundleID)
	}
	if meta.Source != remoteURL.String() {
		t.Errorf("expected metadata source '%s', got '%s'", remoteURL.String(), meta.Source)
	}

	// Verify no metadata file was written under repos/
	badMetaPath := filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-remote-plugin"), "metadata")
	if _, err := os.Stat(badMetaPath); err == nil {
		t.Error("unexpected metadata found inside repository directory")
	}

	// 4. Verify lockfile was written to config/hariti.lock
	lockPath := har.LockfilePath()
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected hariti.lock to exist, got error: %v", err)
	}

	lockBytes, err := ioutil.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}

	var lock hariti.Lockfile
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatalf("failed to unmarshal lockfile: %v", err)
	}

	if len(lock.Bundles) != 2 {
		t.Fatalf("expected 2 bundles locked, got %d", len(lock.Bundles))
	}

	if lock.Bundles[0].ID != "my-local-git-plugin" || lock.Bundles[0].Revision != "local" {
		t.Errorf("invalid lock entry for local: %+v", lock.Bundles[0])
	}
	if lock.Bundles[1].ID != "my-remote-plugin" || lock.Bundles[1].Revision != expectedRevision {
		t.Errorf("invalid lock entry for remote: %+v", lock.Bundles[1])
	}
}

func TestHariti_Sync_SourceMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// Create 2 mock Git remote repositories
	remoteDir1 := filepath.Join(tmpDir, "remote1")
	remoteDir2 := filepath.Join(tmpDir, "remote2")
	_ = os.MkdirAll(remoteDir1, 0755)
	_ = os.MkdirAll(remoteDir2, 0755)

	setupRepo := func(dir string, msg string) string {
		_ = runGitCmdInDir(nil, dir, "init")
		_ = runGitCmdInDir(nil, dir, "config", "user.email", "test@hariti.io")
		_ = runGitCmdInDir(nil, dir, "config", "user.name", "Test Hariti")
		_ = runGitCmdInDir(nil, dir, "commit", "--allow-empty", "-m", msg)
		return runGitCmdInDir(nil, dir, "rev-parse", "HEAD")
	}

	rev1 := setupRepo(remoteDir1, "repo 1 commit")
	rev2 := setupRepo(remoteDir2, "repo 2 commit")

	remoteURL1, _ := url.Parse("file://" + filepath.ToSlash(remoteDir1))
	remoteURL2, _ := url.Parse("file://" + filepath.ToSlash(remoteDir2))

	cfg := &hariti.HaritiConfig{
		Directory: filepath.Join(tmpDir, "hariti_home"),
		Writer:    ioutil.Discard,
		ErrWriter: ioutil.Discard,
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = hariti.WithWriter(ctx, ioutil.Discard)
	ctx = hariti.WithErrWriter(ctx, ioutil.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(ioutil.Discard))

	// Step 1: Sync with remoteURL1
	g1 := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my-mismatch-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL1,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-mismatch-plugin")),
				},
			},
		},
	}

	facts1, err := har.Sync(ctx, g1, false)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if facts1[0].Revision != rev1 {
		t.Errorf("expected revision %s, got %s", rev1, facts1[0].Revision)
	}

	// Verify metadata points to remoteURL1
	metaPath := filepath.Join(xdgHome, "hariti", "metadata", url.QueryEscape("my-mismatch-plugin"))
	metaBytes, _ := ioutil.ReadFile(metaPath)
	var meta hariti.RepositoryMetadata
	_ = json.Unmarshal(metaBytes, &meta)
	if meta.Source != remoteURL1.String() {
		t.Errorf("expected meta source %s, got %s", remoteURL1.String(), meta.Source)
	}

	// Step 2: Sync with remoteURL2 (mismatch!)
	g2 := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my-mismatch-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL2,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-mismatch-plugin")),
				},
			},
		},
	}

	facts2, err := har.Sync(ctx, g2, false)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}
	if facts2[0].Revision != rev2 {
		t.Errorf("expected revision %s (re-cloned from remote2), got %s", rev2, facts2[0].Revision)
	}

	// Verify metadata updated to remoteURL2
	metaBytes2, _ := ioutil.ReadFile(metaPath)
	_ = json.Unmarshal(metaBytes2, &meta)
	if meta.Source != remoteURL2.String() {
		t.Errorf("expected updated meta source %s, got %s", remoteURL2.String(), meta.Source)
	}
}

func TestHariti_Sync_LocalSourceMissingError(t *testing.T) {
	tmpDir := t.TempDir()

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "missing-local-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: filepath.Join(tmpDir, "non_existent_folder"),
				},
			},
		},
	}

	cfg := &hariti.HaritiConfig{
		Directory: filepath.Join(tmpDir, "hariti_home"),
		Writer:    ioutil.Discard,
		ErrWriter: ioutil.Discard,
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = hariti.WithWriter(ctx, ioutil.Discard)
	ctx = hariti.WithErrWriter(ctx, ioutil.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(ioutil.Discard))

	_, err := har.Sync(ctx, g, false)
	if err == nil {
		t.Error("expected sync to fail for missing local source path, but got nil")
	}
}
