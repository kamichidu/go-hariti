package e2e_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

type normalizeVars struct {
	TestDir    string
	HomeDir    string
	VimRuntime string
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func normalizeRuntimepath(t *testing.T, raw string, vars normalizeVars) []string {
	testDir := filepath.ToSlash(filepath.Clean(vars.TestDir))
	homeDir := filepath.ToSlash(filepath.Clean(vars.HomeDir))
	vimRuntime := filepath.ToSlash(filepath.Clean(vars.VimRuntime))
	vimParent := filepath.ToSlash(filepath.Clean(filepath.Dir(vimRuntime)))

	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		cleaned := filepath.ToSlash(filepath.Clean(p))

		if strings.HasPrefix(cleaned, testDir) {
			cleaned = "<TESTDIR>" + cleaned[len(testDir):]
		} else if strings.HasPrefix(cleaned, vimRuntime) {
			cleaned = "<VIMRUNTIME>" + cleaned[len(vimRuntime):]
		} else if strings.HasPrefix(cleaned, vimParent) {
			cleaned = "<VIMPARENT>" + cleaned[len(vimParent):]
		} else if strings.HasPrefix(cleaned, homeDir) {
			cleaned = "<HOME>" + cleaned[len(homeDir):]
		}

		result = append(result, cleaned)
	}
	return result
}

func TestE2E_VimRuntimepathProjection(t *testing.T) {
	// Skip test if vim is not installed
	if _, err := exec.LookPath("vim"); err != nil {
		t.Skip("vim not installed")
	}

	tmpDir := t.TempDir()
	if tmpDir == "" {
		t.Fatalf("test temporary directory is empty")
	}

	// Locate fixture source under e2e/testdata/simple
	fixtureSrc := filepath.Join("testdata", "simple")
	fixtureSrcAbs, err := filepath.Abs(fixtureSrc)
	if err != nil {
		t.Fatalf("failed to get absolute path of fixture source: %v", err)
	}

	// Copy fixture into isolated temporary directory for reproducibility
	fixtureDst := filepath.Join(tmpDir, "simple")
	if err := copyDir(fixtureSrcAbs, fixtureDst); err != nil {
		t.Fatalf("failed to copy fixture directory: %v", err)
	}

	// Setup E2E_PLUGINS_DIR env var pointing to copied simple/plugins
	oldPluginsDir, exists := os.LookupEnv("E2E_PLUGINS_DIR")
	_ = os.Setenv("E2E_PLUGINS_DIR", filepath.Join(fixtureDst, "plugins"))
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv("E2E_PLUGINS_DIR", oldPluginsDir)
		} else {
			_ = os.Unsetenv("E2E_PLUGINS_DIR")
		}
	})

	// Configure and initialize Hariti
	cfg := &hariti.HaritiConfig{
		Paths: hariti.Paths{
			ConfigFile: filepath.Join(fixtureDst, "bundles.hariti"),
			ConfigDir:  filepath.Join(tmpDir, "config"),
			DataDir:    filepath.Join(tmpDir, "data"),
		},
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
	}
	har := hariti.NewHariti(cfg)

	if err := har.SetupManagedDirectory(); err != nil {
		t.Fatalf("failed to setup managed directories: %v", err)
	}

	// 1. Load & Parse DSL to compile Graph
	g, err := dsl.LoadGraph(cfg.Paths.ConfigFile)
	if err != nil {
		t.Fatalf("failed to load graph: %v", err)
	}

	// 2. Sync to write lockfile
	ctx := context.Background()
	_, err = har.Sync(ctx, g, hariti.SyncOptions{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 3. Deploy to generate layout and packadd.vim
	genID, err := har.Deploy(ctx, g, hariti.DeployOptions{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if genID == "" {
		t.Fatal("expected non-empty generation ID")
	}

	// 4. Assert current generation pointer exists
	currentPath := har.CurrentSymlinkPath()
	if _, err := os.Stat(currentPath); err != nil {
		t.Fatalf("expected current generation pointer link to exist: %v", err)
	}

	// 5. Spawn Vim in Ex silent mode to source packadd.vim and observe runtimepath
	runVimScript := filepath.Join(tmpDir, "run.vim")
	outFile := filepath.Join(tmpDir, "output.txt")

	scriptContent := fmt.Sprintf(`set nomore
set packpath+=%s
source %s/packadd.vim
try
  help local-plugin
  let help_ok = "SUCCESS"
catch
  let help_ok = "FAILURE: " . v:exception
endtry
redir! > %s
silent echo &runtimepath
silent echo $VIMRUNTIME
silent echo help_ok
redir END
qa!
`, filepath.ToSlash(currentPath), filepath.ToSlash(currentPath), filepath.ToSlash(outFile))

	if err := os.WriteFile(runVimScript, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to write run.vim script: %v", err)
	}

	cmd := exec.Command("vim", "-Nu", "NONE", "-n", "-e", "-s", "-S", runVimScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to execute vim: %v\nOutput: %s", err, string(out))
	}

	// Read & parse output
	outputBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read vim output file: %v", err)
	}

	outputLines := strings.Split(strings.TrimSpace(string(outputBytes)), "\n")
	var lines []string
	for _, line := range outputLines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	if len(lines) < 3 {
		t.Fatalf("unexpected vim output structure (expected at least 3 lines): %v", lines)
	}

	rawRuntimepath := lines[0]
	if rawRuntimepath == "" {
		t.Fatalf("failed to retrieve non-empty &runtimepath from Vim")
	}

	vimRuntime := lines[1]
	if vimRuntime == "" {
		t.Fatalf("failed to retrieve non-empty $VIMRUNTIME from Vim")
	}

	helpStatus := lines[2]
	if helpStatus != "SUCCESS" {
		t.Errorf("Vim help verification failed: %s", helpStatus)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		t.Fatalf("failed to get non-empty user home directory: %v", err)
	}

	vars := normalizeVars{
		TestDir:    tmpDir,
		HomeDir:    homeDir,
		VimRuntime: vimRuntime,
	}

	gotRTP := normalizeRuntimepath(t, rawRuntimepath, vars)

	// Explicit contains assertion for the generated plugin paths
	depPluginFound := false
	localPluginFound := false
	expectedDepPath := "<TESTDIR>/simple/plugins/dep-plugin"
	expectedLocalPath := "<TESTDIR>/simple/plugins/local-plugin"

	for _, entry := range gotRTP {
		if entry == expectedDepPath {
			depPluginFound = true
		}
		if entry == expectedLocalPath {
			localPluginFound = true
		}
	}

	if !depPluginFound {
		t.Errorf("expected dep-plugin path %q to appear in runtimepath; actual runtimepaths: %v", expectedDepPath, gotRTP)
	}
	if !localPluginFound {
		t.Errorf("expected local-plugin path %q to appear in runtimepath; actual runtimepaths: %v", expectedLocalPath, gotRTP)
	}

	// Load expected snapshot
	snapshotPath := filepath.Join(fixtureSrcAbs, "expected", "runtimepath.snap")

	if os.Getenv("UPDATE_SNAPSHOT") != "" {
		snapContent := strings.Join(gotRTP, "\n") + "\n"
		if err := os.WriteFile(snapshotPath, []byte(snapContent), 0644); err != nil {
			t.Fatalf("failed to update snapshot: %v", err)
		}
		t.Logf("Snapshot updated successfully.")
		return
	}

	snapBytes, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("failed to read snapshot file: %v", err)
	}

	var expectedRTP []string
	for _, line := range strings.Split(string(snapBytes), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			expectedRTP = append(expectedRTP, trimmed)
		}
	}

	// Compare with snapshot
	mismatch := false
	if len(gotRTP) != len(expectedRTP) {
		mismatch = true
	} else {
		for i := range gotRTP {
			if gotRTP[i] != expectedRTP[i] {
				mismatch = true
				break
			}
		}
	}

	if mismatch {
		t.Errorf("observed runtimepath does not match snapshot %s\nGot:\n%s\nExpected:\n%s",
			snapshotPath, strings.Join(gotRTP, "\n"), strings.Join(expectedRTP, "\n"))
	}
}
