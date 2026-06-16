package hariti_test

import (
	"context"
	"encoding/json"
	"fmt"
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

func runGitCmdInDir(t *testing.T, dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if t != nil {
			t.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(out))
		} else {
			panic(fmt.Sprintf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(out)))
		}
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
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti_home", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti_home"),
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Logger:    hariti.NewStdLogger(io.Discard),
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(io.Discard))

	facts, err := har.Sync(ctx, g, hariti.SyncOptions{})
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

	metaBytes, err := os.ReadFile(metaPath)
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

	lockBytes, err := os.ReadFile(lockPath)
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
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti_home", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti_home"),
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Logger:    hariti.NewStdLogger(io.Discard),
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(io.Discard))

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

	facts1, err := har.Sync(ctx, g1, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if facts1[0].Revision != rev1 {
		t.Errorf("expected revision %s, got %s", rev1, facts1[0].Revision)
	}

	// Verify metadata points to remoteURL1
	metaPath := filepath.Join(xdgHome, "hariti", "metadata", url.QueryEscape("my-mismatch-plugin"))
	metaBytes, _ := os.ReadFile(metaPath)
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

	facts2, err := har.Sync(ctx, g2, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}
	if facts2[0].Revision != rev2 {
		t.Errorf("expected revision %s (re-cloned from remote2), got %s", rev2, facts2[0].Revision)
	}

	// Verify metadata updated to remoteURL2
	metaBytes2, _ := os.ReadFile(metaPath)
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
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti_home", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti_home"),
			DataDir:    filepath.Join(tmpDir, "hariti_data"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Logger:    hariti.NewStdLogger(io.Discard),
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(io.Discard))

	_, err := har.Sync(ctx, g, hariti.SyncOptions{})
	if err == nil {
		t.Error("expected sync to fail for missing local source path, but got nil")
	}
}

func TestHariti_Sync_UpstreamResetAdvance(t *testing.T) {
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// 1. Setup mock Git remote repo
	remoteRepoDir := filepath.Join(tmpDir, "remote_repo")
	if err := os.MkdirAll(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create mock remote dir: %v", err)
	}

	_ = runGitCmdInDir(nil, remoteRepoDir, "init", "--initial-branch=main")
	_ = runGitCmdInDir(nil, remoteRepoDir, "config", "user.email", "test@hariti.io")
	_ = runGitCmdInDir(nil, remoteRepoDir, "config", "user.name", "Test Hariti")

	// Create and commit a tracked file
	trackedFile := filepath.Join(remoteRepoDir, "tracked_file.txt")
	if err := os.WriteFile(trackedFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("failed to write tracked file: %v", err)
	}
	_ = runGitCmdInDir(nil, remoteRepoDir, "add", "tracked_file.txt")
	_ = runGitCmdInDir(nil, remoteRepoDir, "commit", "-m", "initial commit with tracked file")
	rev1 := runGitCmdInDir(nil, remoteRepoDir, "rev-parse", "HEAD")

	remoteURL, err := url.Parse("file://" + filepath.ToSlash(remoteRepoDir))
	if err != nil {
		t.Fatalf("failed to parse remote URL: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my-tracked-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-tracked-plugin")),
				},
			},
		},
	}

	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti_home", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti_home"),
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Logger:    hariti.NewStdLogger(io.Discard),
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(io.Discard))

	// Step 1: Clone initially
	facts1, err := har.Sync(ctx, g, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("initial Sync failed: %v", err)
	}
	if facts1[0].Revision != rev1 {
		t.Errorf("expected revision %s, got %s", rev1, facts1[0].Revision)
	}

	// Step 2: Make local uncommitted modifications to the tracked file in the cache repository
	cacheRepoPath := filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-tracked-plugin"))
	dirtyFile := filepath.Join(cacheRepoPath, "tracked_file.txt")
	if err := os.WriteFile(dirtyFile, []byte("local modifications"), 0644); err != nil {
		t.Fatalf("failed to write dirty file: %v", err)
	}

	// Step 3: Advance remote repository by committing a new changes
	_ = runGitCmdInDir(nil, remoteRepoDir, "commit", "--allow-empty", "-m", "remote advance commit")
	rev2 := runGitCmdInDir(nil, remoteRepoDir, "rev-parse", "HEAD")

	// Step 4: Run Sync again. It must fetch, wipe local changes (hard reset), and advance HEAD to rev2!
	facts2, err := har.Sync(ctx, g, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("second Sync failed: %v", err)
	}

	if facts2[0].Revision != rev2 {
		t.Errorf("expected revision to advance to %s, got %s", rev2, facts2[0].Revision)
	}

	// Verify uncommitted modifications were cleanly wiped out (tracked file content is restored/reset)
	contentBytes, err := os.ReadFile(dirtyFile)
	if err != nil {
		t.Fatalf("failed to read tracked file: %v", err)
	}
	if string(contentBytes) == "local modifications" {
		t.Error("expected uncommitted local modifications to be wiped out by hard reset, but tracked file still has dirty changes")
	}
}

func TestHariti_Sync_NoUpstreamError(t *testing.T) {
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}()

	// 1. Create a local git repository inside the cache folder without tracked upstream branch
	cacheRepoPath := filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-untracked-plugin"))
	if err := os.MkdirAll(cacheRepoPath, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	_ = runGitCmdInDir(nil, cacheRepoPath, "init")
	_ = runGitCmdInDir(nil, cacheRepoPath, "config", "user.email", "test@hariti.io")
	_ = runGitCmdInDir(nil, cacheRepoPath, "config", "user.name", "Test Hariti")
	_ = runGitCmdInDir(nil, cacheRepoPath, "commit", "--allow-empty", "-m", "local only commit")

	// 2. Set up sync graph representing a remote sync
	remoteURL, _ := url.Parse("file://" + filepath.ToSlash(filepath.Join(tmpDir, "some_fake_remote")))
	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my-untracked-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  remoteURL,
					Path: cacheRepoPath,
				},
			},
		},
	}

	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti_home", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti_home"),
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Logger:    hariti.NewStdLogger(io.Discard),
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(io.Discard))

	// Step 3: Run Sync. Since cache repo exists but does NOT have upstream tracked, it must fail with an explicit error!
	_, err := har.Sync(ctx, g, hariti.SyncOptions{})
	if err == nil {
		t.Error("expected Sync to fail due to missing tracked upstream ref, but got nil")
	}
}

func TestHariti_Sync_SubmoduleUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg_home")
	_ = os.Setenv("XDG_DATA_HOME", xdgHome)
	_ = os.Setenv("GIT_ALLOW_PROTOCOL", "file:git:http:https:ssh")
	defer func() {
		_ = os.Unsetenv("XDG_DATA_HOME")
		_ = os.Unsetenv("GIT_ALLOW_PROTOCOL")
	}()

	// 1. Setup mock Git remote submodule repo
	subRepoDir := filepath.Join(tmpDir, "sub_repo")
	if err := os.MkdirAll(subRepoDir, 0755); err != nil {
		t.Fatalf("failed to create submodule dir: %v", err)
	}
	_ = runGitCmdInDir(nil, subRepoDir, "init", "--initial-branch=main")
	_ = runGitCmdInDir(nil, subRepoDir, "config", "user.email", "test@hariti.io")
	_ = runGitCmdInDir(nil, subRepoDir, "config", "user.name", "Test Hariti")
	_ = runGitCmdInDir(nil, subRepoDir, "config", "protocol.file.allow", "always")
	subFile := filepath.Join(subRepoDir, "sub_file.txt")
	_ = os.WriteFile(subFile, []byte("submodule commit 1"), 0644)
	_ = runGitCmdInDir(nil, subRepoDir, "add", "sub_file.txt")
	_ = runGitCmdInDir(nil, subRepoDir, "commit", "-m", "initial sub commit")

	// 2. Setup mock Git remote main repo
	mainRepoDir := filepath.Join(tmpDir, "main_repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatalf("failed to create main dir: %v", err)
	}
	_ = runGitCmdInDir(nil, mainRepoDir, "init", "--initial-branch=main")
	_ = runGitCmdInDir(nil, mainRepoDir, "config", "user.email", "test@hariti.io")
	_ = runGitCmdInDir(nil, mainRepoDir, "config", "user.name", "Test Hariti")
	_ = runGitCmdInDir(nil, mainRepoDir, "config", "protocol.file.allow", "always")
	_ = os.WriteFile(filepath.Join(mainRepoDir, "main_file.txt"), []byte("main content"), 0644)
	_ = runGitCmdInDir(nil, mainRepoDir, "add", "main_file.txt")
	_ = runGitCmdInDir(nil, mainRepoDir, "commit", "-m", "initial main commit")

	// Add submodule to the remote main repository
	subURLStr := "file://" + filepath.ToSlash(subRepoDir)
	_ = runGitCmdInDir(nil, mainRepoDir, "submodule", "add", subURLStr, "my_submodule")
	_ = runGitCmdInDir(nil, mainRepoDir, "commit", "-m", "add submodule")
	mainRev1 := runGitCmdInDir(nil, mainRepoDir, "rev-parse", "HEAD")

	mainURL, err := url.Parse("file://" + filepath.ToSlash(mainRepoDir))
	if err != nil {
		t.Fatalf("failed to parse main URL: %v", err)
	}

	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my-submodule-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  mainURL,
					Path: filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-submodule-plugin")),
				},
			},
		},
	}

	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(tmpDir, "hariti_home", "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "hariti_home"),
			DataDir:    filepath.Join(xdgHome, "hariti"),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Logger:    hariti.NewStdLogger(io.Discard),
	}
	har := hariti.NewHariti(cfg)

	ctx := context.Background()
	ctx = vcs.WithWriter(ctx, io.Discard)
	ctx = vcs.WithErrWriter(ctx, io.Discard)
	ctx = hariti.WithLogger(ctx, hariti.NewStdLogger(io.Discard))

	// Step 1: Sync first to clone main and recursively clone submodule
	facts1, err := har.Sync(ctx, g, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("first Sync failed: %v", err)
	}
	if facts1[0].Revision != mainRev1 {
		t.Errorf("expected revision %s, got %s", mainRev1, facts1[0].Revision)
	}

	// Verify submodule file was recursively cloned initially
	clonedCachePath := filepath.Join(xdgHome, "hariti", "repos", url.QueryEscape("my-submodule-plugin"))
	submoduleFilePath := filepath.Join(clonedCachePath, "my_submodule", "sub_file.txt")
	subBytes1, err := os.ReadFile(submoduleFilePath)
	if err != nil {
		t.Fatalf("failed to read cloned submodule file: %v", err)
	}
	if string(subBytes1) != "submodule commit 1" {
		t.Errorf("expected submodule file content 'submodule commit 1', got '%s'", string(subBytes1))
	}

	// Step 2: Advance submodule remote repository
	_ = os.WriteFile(subFile, []byte("submodule commit 2"), 0644)
	_ = runGitCmdInDir(nil, subRepoDir, "add", "sub_file.txt")
	_ = runGitCmdInDir(nil, subRepoDir, "commit", "-m", "advance submodule content")
	subRev2 := runGitCmdInDir(nil, subRepoDir, "rev-parse", "HEAD")

	// Step 3: Commit the advanced submodule pointer in the main remote repository
	_ = runGitCmdInDir(nil, mainRepoDir, "submodule", "update", "--remote", "my_submodule")
	_ = runGitCmdInDir(nil, mainRepoDir, "commit", "-am", "advance submodule pointer in main")
	mainRev2 := runGitCmdInDir(nil, mainRepoDir, "rev-parse", "HEAD")

	// Step 4: Run Sync again. It must fetch, hard reset, find `.gitmodules` exists, and run submodule update!
	facts2, err := har.Sync(ctx, g, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("second Sync failed: %v", err)
	}
	if facts2[0].Revision != mainRev2 {
		t.Errorf("expected revision to advance to %s, got %s", mainRev2, facts2[0].Revision)
	}

	// Verify that the submodule worktree inside our cache directory successfully updated and points to subRev2!
	subBytes2, err := os.ReadFile(submoduleFilePath)
	if err != nil {
		t.Fatalf("failed to read updated submodule file: %v", err)
	}
	if string(subBytes2) != "submodule commit 2" {
		t.Errorf("expected submodule file content to update to 'submodule commit 2', got '%s'", string(subBytes2))
	}

	subHeadRev := runGitCmdInDir(nil, filepath.Join(clonedCachePath, "my_submodule"), "rev-parse", "HEAD")
	if subHeadRev != subRev2 {
		t.Errorf("expected submodule HEAD revision to point to %s, got %s", subRev2, subHeadRev)
	}
}
