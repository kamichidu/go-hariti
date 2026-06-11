package hariti_test

import (
	"context"
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

func TestHariti_Sync_LocalAndRemote(t *testing.T) {
	// Create a temporary directory for the entire test
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME for isolation
	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// 1. Setup a mock Git remote repository on the local filesystem
	remoteRepoDir := filepath.Join(tmpDir, "remote_repo")
	if err := os.MkdirAll(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create mock remote dir: %v", err)
	}

	runGitCmd := func(dir string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(out))
		}
		return strings.TrimSpace(string(out))
	}

	_ = runGitCmd(remoteRepoDir, "init")
	_ = runGitCmd(remoteRepoDir, "config", "user.email", "test@hariti.io")
	_ = runGitCmd(remoteRepoDir, "config", "user.name", "Test Hariti")
	_ = runGitCmd(remoteRepoDir, "commit", "--allow-empty", "-m", "initial commit")
	expectedRevision := runGitCmd(remoteRepoDir, "rev-parse", "HEAD")

	// 2. Setup a local plugin directory
	localPluginDir := filepath.Join(tmpDir, "local_plugin")
	if err := os.MkdirAll(localPluginDir, 0755); err != nil {
		t.Fatalf("failed to create local plugin dir: %v", err)
	}

	// 3. Define the Resolved Graph
	remoteURL, err := url.Parse("file://" + filepath.ToSlash(remoteRepoDir))
	if err != nil {
		t.Fatalf("failed to parse remote URL: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my-local-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: localPluginDir,
				},
			},
			{
				ID: "my-remote-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape(remoteURL.String())),
				},
			},
		},
	}

	// 4. Initialize Hariti and trigger Sync
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

	// Verify we got 2 facts back
	if len(facts) != 2 {
		t.Fatalf("expected 2 repository facts, got %d", len(facts))
	}

	// Verify local fact
	if facts[0].BundleID != "my-local-plugin" {
		t.Errorf("expected bundle ID 'my-local-plugin', got '%s'", facts[0].BundleID)
	}
	if facts[0].Revision != "" {
		t.Errorf("expected empty revision for local bundle, got '%s'", facts[0].Revision)
	}

	// Verify remote fact
	if facts[1].BundleID != "my-remote-plugin" {
		t.Errorf("expected bundle ID 'my-remote-plugin', got '%s'", facts[1].BundleID)
	}
	if facts[1].Revision != expectedRevision {
		t.Errorf("expected observed revision '%s', got '%s'", expectedRevision, facts[1].Revision)
	}

	// Verify that remote was cloned under $XDG_DATA_HOME/hariti/repos/
	clonedPath := filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape(remoteURL.String()))
	if _, err := os.Stat(clonedPath); err != nil {
		t.Errorf("expected remote clone to exist at %s, but got error: %v", clonedPath, err)
	}

	// 5. Test Remote Update (Fetch/Pull)
	_ = runGitCmd(remoteRepoDir, "commit", "--allow-empty", "-m", "second commit")
	newExpectedRevision := runGitCmd(remoteRepoDir, "rev-parse", "HEAD")

	facts2, err := har.Sync(ctx, g, true) // with update = true
	if err != nil {
		t.Fatalf("Sync update failed: %v", err)
	}

	if facts2[1].Revision != newExpectedRevision {
		t.Errorf("expected updated revision '%s', got '%s'", newExpectedRevision, facts2[1].Revision)
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
