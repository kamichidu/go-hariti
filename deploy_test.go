package hariti_test

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/graph"
	"github.com/kamichidu/go-hariti/vcs"
	_ "github.com/kamichidu/go-hariti/vcs/git"
)

func TestHariti_Deploy_Success(t *testing.T) {
	// Create a temporary directory for the entire test
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME for isolation
	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// 1. Setup mock remote repository on the local filesystem
	remoteRepoDir := filepath.Join(tmpDir, "remote_repo")
	if err := os.MkdirAll(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create mock remote dir: %v", err)
	}

	_ = runGitCmdInDir(t, remoteRepoDir, "init")
	_ = runGitCmdInDir(t, remoteRepoDir, "config", "user.email", "test@hariti.io")
	_ = runGitCmdInDir(t, remoteRepoDir, "config", "user.name", "Test Hariti")
	_ = runGitCmdInDir(t, remoteRepoDir, "commit", "--allow-empty", "-m", "initial commit")

	// 2. Setup local plugin directory
	localPluginDir := filepath.Join(tmpDir, "local_plugin")
	if err := os.MkdirAll(localPluginDir, 0755); err != nil {
		t.Fatalf("failed to create local plugin dir: %v", err)
	}
	// Add a dummy file inside local plugin to verify it gets exported
	localFile := filepath.Join(localPluginDir, "plugin.vim")
	if err := os.WriteFile(localFile, []byte("echo 'local'"), 0644); err != nil {
		t.Fatalf("failed to write local file: %v", err)
	}

	remoteURL, err := url.Parse("file://" + filepath.ToSlash(remoteRepoDir))
	if err != nil {
		t.Fatalf("failed to parse remote URL: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my/local-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: localPluginDir,
				},
				EnableIf: "has('python3')",
			},
			{
				ID: "my/remote-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my/remote-plugin")),
				},
				Build: []graph.BuildStep{
					{
						OS:  "all",
						Cmd: "echo 'built' > build_output.txt",
					},
				},
			},
		},
	}

	// Initialize Hariti
	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti home with spaces", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti home with spaces"),
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)

	// Step 1: Sync first to retrieve and write hariti.lock
	_, err = har.Sync(ctx, g, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Step 2: Run Deploy (Generation)
	genID, err := har.Deploy(ctx, g, hariti.DeployOptions{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if genID == "" {
		t.Fatal("expected non-empty generation ID")
	}

	// 3. Verify directory and files inside Generation
	genDir := filepath.Join(har.GenerationsDir(), genID)

	// Check metadata.json
	metaBytes, err := os.ReadFile(filepath.Join(genDir, "metadata.json"))
	if err != nil {
		t.Fatalf("failed to read metadata.json: %v", err)
	}
	var meta hariti.GenerationMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("failed to parse metadata.json: %v", err)
	}
	if meta.ID != genID {
		t.Errorf("expected metadata ID '%s', got '%s'", genID, meta.ID)
	}

	// Check lock.json
	lockBytes, err := os.ReadFile(filepath.Join(genDir, "lock.json"))
	if err != nil {
		t.Fatalf("failed to read lock.json: %v", err)
	}
	var lock hariti.Lockfile
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatalf("failed to parse lock.json: %v", err)
	}
	if len(lock.Bundles) != 2 {
		t.Errorf("expected 2 locked bundles inside generation lock.json, got %d", len(lock.Bundles))
	}

	// Verify current symlink atomic transition
	currentSym, err := os.Readlink(har.CurrentSymlinkPath())
	if err != nil {
		t.Fatalf("failed to read current symlink: %v", err)
	}
	expectedSymTarget := filepath.Join("generations", genID)
	if currentSym != expectedSymTarget {
		t.Errorf("expected symlink target '%s', got '%s'", expectedSymTarget, currentSym)
	}

	// 4. Verify bundle exports
	// Local bundle is NOT exported (its files are NOT copied)
	localExportPath := filepath.Join(genDir, "pack", "hariti", "opt", "my_local-plugin")
	if _, err := os.Stat(localExportPath); err == nil || !os.IsNotExist(err) {
		t.Error("unexpected local bundle directory found inside generation pack layout")
	}

	// Remote bundle exported
	remoteExportPath := filepath.Join(genDir, "pack", "hariti", "opt", "my_remote-plugin")
	if _, err := os.Stat(remoteExportPath); err != nil {
		t.Errorf("expected exported remote directory to exist, got error: %v", err)
	}

	// Verify build step executed inside the EXPORTED bundle folder and NOT in repo store
	buildFileInGen := filepath.Join(remoteExportPath, "build_output.txt")
	if _, err := os.Stat(buildFileInGen); err != nil {
		t.Errorf("expected build step output inside generation, got error: %v", err)
	}
	buildFileInRepo := filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my/remote-plugin"), "build_output.txt")
	if _, err := os.Stat(buildFileInRepo); err == nil {
		t.Error("unexpected build output file found inside remote repository store")
	}

	// 5. Verify generated packadd.vim
	packaddBytes, err := os.ReadFile(filepath.Join(genDir, "packadd.vim"))
	if err != nil {
		t.Fatalf("failed to read packadd.vim: %v", err)
	}
	packaddStr := string(packaddBytes)

	// Check un-braced enable_if wraps for local source
	expectedLocalWrap := "if has('python3')\n  set runtimepath+=" + localPluginDir + "\nendif"
	if !strings.Contains(packaddStr, expectedLocalWrap) {
		t.Errorf("expected packadd.vim to contain: \n%s\nGot:\n%s", expectedLocalWrap, packaddStr)
	}

	// Check simple packadd for no enable_if
	expectedRemoteWrap := "packadd my_remote-plugin"
	if !strings.Contains(packaddStr, expectedRemoteWrap) {
		t.Errorf("expected packadd.vim to contain: \n%s\nGot:\n%s", expectedRemoteWrap, packaddStr)
	}
}

func TestHariti_Deploy_Failure_BuildStep(t *testing.T) {
	// Create a temporary directory for the entire test
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME for isolation
	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// 1. Setup mock remote repository on the local filesystem
	remoteRepoDir := filepath.Join(tmpDir, "remote_repo")
	if err := os.MkdirAll(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create mock remote dir: %v", err)
	}

	_ = runGitCmdInDir(t, remoteRepoDir, "init")
	_ = runGitCmdInDir(t, remoteRepoDir, "config", "user.email", "test@hariti.io")
	_ = runGitCmdInDir(t, remoteRepoDir, "config", "user.name", "Test Hariti")
	_ = runGitCmdInDir(t, remoteRepoDir, "commit", "--allow-empty", "-m", "initial commit")

	remoteURL, err := url.Parse("file://" + filepath.ToSlash(remoteRepoDir))
	if err != nil {
		t.Fatalf("failed to parse remote URL: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my/remote-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my/remote-plugin")),
				},
				Build: []graph.BuildStep{
					{
						OS:  "all",
						Cmd: "exit 1",
					},
				},
			},
		},
	}

	// Initialize Hariti
	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti home with spaces", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti home with spaces"),
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)

	// Step 1: Sync first to retrieve and write hariti.lock
	_, err = har.Sync(ctx, g, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Step 2: Run Deploy (Generation) which should fail due to build step failure
	_, err = har.Deploy(ctx, g, hariti.DeployOptions{})
	if err == nil {
		t.Fatal("expected Deploy to fail due to build step exit 1, but it succeeded")
	}

	if !strings.Contains(err.Error(), "failed to run build step for bundle my/remote-plugin on all") {
		t.Errorf("expected error message to contain build step failure detail, got: %v", err)
	}
}

func TestHariti_Deploy_Failure_HelpTags(t *testing.T) {
	// Skip test if vim is not installed
	if _, err := exec.LookPath("vim"); err != nil {
		t.Skip("vim not installed")
	}

	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// Setup local plugin with doc/tags/ as a directory to trigger helptags failure
	localPluginDir := filepath.Join(tmpDir, "local_plugin")
	docTagsDir := filepath.Join(localPluginDir, "doc", "tags")
	if err := os.MkdirAll(docTagsDir, 0755); err != nil {
		t.Fatalf("failed to create doc/tags directory: %v", err)
	}

	// Write dummy txt help file so helptags attempts to run
	helpFile := filepath.Join(localPluginDir, "doc", "plugin.txt")
	if err := os.WriteFile(helpFile, []byte("*plugin.txt* dummy"), 0644); err != nil {
		t.Fatalf("failed to write help file: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my/local-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: localPluginDir,
				},
			},
		},
	}

	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "bundles.hariti"),
			ConfigDir:  tmpDir,
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
	}
	har := hariti.NewHariti(cfg)

	if err := har.SetupManagedDirectory(); err != nil {
		t.Fatalf("failed to setup managed directories: %v", err)
	}

	// Write dummy lockfile
	if err := os.WriteFile(har.LockfilePath(), []byte(`{"bundles": []}`), 0644); err != nil {
		t.Fatalf("failed to write dummy lockfile: %v", err)
	}

	ctx := context.Background()

	// Deploy should fail due to help tags generation failure
	_, err := har.Deploy(ctx, g, hariti.DeployOptions{})
	if err == nil {
		t.Fatal("expected Deploy to fail due to help tags generation failure, but it succeeded")
	}

	if !strings.Contains(err.Error(), "failed to generate help tags for bundle my/local-plugin") {
		t.Errorf("expected error message to contain help tags failure detail, got: %v", err)
	}
}

func TestHariti_Deploy_Success_HelpTags_NonDirectoryDoc(t *testing.T) {
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// Setup local plugin with doc as a regular file (not directory)
	localPluginDir := filepath.Join(tmpDir, "local_plugin")
	if err := os.MkdirAll(localPluginDir, 0755); err != nil {
		t.Fatalf("failed to create local plugin directory: %v", err)
	}

	docFile := filepath.Join(localPluginDir, "doc")
	if err := os.WriteFile(docFile, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("failed to write doc file: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my/local-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: localPluginDir,
				},
			},
		},
	}

	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "bundles.hariti"),
			ConfigDir:  tmpDir,
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
	}
	har := hariti.NewHariti(cfg)

	if err := har.SetupManagedDirectory(); err != nil {
		t.Fatalf("failed to setup managed directories: %v", err)
	}

	// Write dummy lockfile
	if err := os.WriteFile(har.LockfilePath(), []byte(`{"bundles": []}`), 0644); err != nil {
		t.Fatalf("failed to write dummy lockfile: %v", err)
	}

	ctx := context.Background()

	// Deploy should succeed, skipping help tags with a warning
	_, err := har.Deploy(ctx, g, hariti.DeployOptions{})
	if err != nil {
		t.Fatalf("expected Deploy to succeed with non-directory doc warning, but got error: %v", err)
	}
}
