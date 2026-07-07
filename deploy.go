package hariti

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kamichidu/go-hariti/graph"
	"github.com/kamichidu/go-hariti/vcs"
)

type DeployOptions struct {
	// Options can be expanded in the future
}

type GenerationMetadata struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	LockHash  string `json:"lock_hash"`
}

func getExportedBundleDirName(bundleID string) string {
	return strings.ReplaceAll(bundleID, "/", "_")
}

func matchOS(stepOS, currentOS string) bool {
	if stepOS == "all" || stepOS == "*" {
		return true
	}
	if stepOS == "mac" && currentOS == "darwin" {
		return true
	}
	return strings.EqualFold(stepOS, currentOS)
}

func (h *Hariti) Deploy(ctx context.Context, g *graph.Graph, opts DeployOptions) (string, error) {
	rg := h.newRuntimeGraph(g)
	h.logger.Infof("deploy started")

	// Read project-side hariti.lock content
	lockBytes, err := os.ReadFile(h.LockfilePath())
	if err != nil {
		return "", fmt.Errorf("failed to read lockfile: %w", err)
	}

	// Calculate generation semantic identity (hash of lockfile contents)
	hash := sha256.Sum256(lockBytes)
	shortHash := fmt.Sprintf("%x", hash)[:8]
	timestamp := time.Now().Format("20060102-150405")
	genID := fmt.Sprintf("%s-%s", timestamp, shortHash)

	genDir := filepath.Join(h.GenerationsDir(), genID)
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create generation directory: %w", err)
	}
	h.logger.Infof("generation created: %s", genID)

	// Parse lockfile to get mapped revisions
	var lock Lockfile
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		return "", fmt.Errorf("failed to parse lockfile: %w", err)
	}

	revisionsMap := make(map[string]string)
	for _, entry := range lock.Bundles {
		revisionsMap[entry.ID] = entry.Revision
	}

	for _, bundle := range rg.bundles {
		if bundle.Source.Type == graph.SourceTypeLocal {
			h.logger.Debugf("local bundle %s: skipped folder creation and copying", bundle.ID)
			docDir := filepath.Join(bundle.Source.Path, "doc")
			if err := h.buildHelpTags(bundle.ID, docDir); err != nil {
				return "", err
			}
			continue
		}

		destDir := filepath.Join(genDir, "pack", "hariti", "opt", getExportedBundleDirName(bundle.ID))
		h.logger.Debugf("resolved repository path for bundle %s to %s", bundle.ID, destDir)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory for bundle %s: %w", bundle.ID, err)
		}

		if bundle.Source.Type == graph.SourceTypeRemote {
			// Remote source: Export bundle from the revision in lockfile
			revision, exists := revisionsMap[bundle.ID]
			if !exists || revision == "" {
				return "", fmt.Errorf("no revision found in lockfile for remote bundle %s", bundle.ID)
			}
			h.logger.Debugf("resolved revision for bundle %s to %s", bundle.ID, revision)

			v := vcs.Detect(bundle.Source.URL)
			if v == nil {
				return "", fmt.Errorf("failed to detect VCS for remote bundle %s", bundle.ID)
			}

			vcsCtx := vcs.WithLogger(ctx, h.logger)
			vcsCtx = vcs.WithWriter(vcsCtx, h.config.Writer)
			vcsCtx = vcs.WithErrWriter(vcsCtx, h.config.ErrWriter)
			h.logger.Debugf("archive source %s/destination %s", bundle.ID, destDir)
			err := v.Archive(vcsCtx, bundle, revision, destDir)
			if err != nil {
				return "", fmt.Errorf("failed to archive remote bundle %s at revision %s: %w", bundle.ID, revision, err)
			}
		}

		// Run build steps inside the EXPORTED bundle directory in the generation
		for _, step := range bundle.Build {
			if matchOS(step.OS, runtime.GOOS) {
				h.logger.Debugf("running build step for %s: %s", bundle.ID, step.Cmd)
				var buildCmd *exec.Cmd
				if runtime.GOOS == "windows" {
					buildCmd = exec.Command("cmd", "/c", step.Cmd)
				} else {
					buildCmd = exec.Command("sh", "-c", step.Cmd)
				}
				buildCmd.Dir = destDir
				buildCmd.Stdout = h.config.Writer
				buildCmd.Stderr = h.config.ErrWriter
				if err := buildCmd.Run(); err != nil {
					return "", fmt.Errorf("failed to run build step for bundle %s on %s: %w", bundle.ID, step.OS, err)
				}
			}
		}

		// Help tag generation for remote/exported bundle
		docDir := filepath.Join(destDir, "doc")
		if err := h.buildHelpTags(bundle.ID, docDir); err != nil {
			return "", err
		}
	}

	// Generate packadd.vim inside the generation dir
	var packaddContent strings.Builder
	packaddContent.WriteString(`function! s:add_rtp(path, after_path) abort
  let l:rtps = split(&runtimepath, ',')
  let l:idx = -1
  for l:i in range(len(l:rtps))
    if fnamemodify(l:rtps[l:i], ':t') ==# 'after'
      let l:idx = l:i
      break
    endif
  endfor
  if l:idx >= 0
    call insert(l:rtps, a:path, l:idx)
  else
    call add(l:rtps, a:path)
  endif
  if a:after_path !=# ''
    call add(l:rtps, a:after_path)
  endif
  let &runtimepath = join(l:rtps, ',')
endfunction

`)

	for _, bundle := range rg.bundles {
		if bundle.Source.Type == graph.SourceTypeLocal {
			localPath := bundle.Source.Path
			if bundle.EnableIf != "" {
				afterPath := filepath.Join(localPath, "after")
				if _, err := os.Stat(afterPath); err == nil {
					fmt.Fprintf(&packaddContent, "if %s\n  call s:add_rtp(%q, %q)\nendif\n", bundle.EnableIf, filepath.ToSlash(localPath), filepath.ToSlash(afterPath))
				} else {
					fmt.Fprintf(&packaddContent, "if %s\n  call s:add_rtp(%q, '')\nendif\n", bundle.EnableIf, filepath.ToSlash(localPath))
				}
			} else {
				afterPath := filepath.Join(localPath, "after")
				if _, err := os.Stat(afterPath); err == nil {
					fmt.Fprintf(&packaddContent, "call s:add_rtp(%q, %q)\n", filepath.ToSlash(localPath), filepath.ToSlash(afterPath))
				} else {
					fmt.Fprintf(&packaddContent, "call s:add_rtp(%q, '')\n", filepath.ToSlash(localPath))
				}
			}
		} else {
			bundleName := getExportedBundleDirName(bundle.ID)
			if bundle.EnableIf != "" {
				fmt.Fprintf(&packaddContent, "if %s\n  packadd %s\nendif\n", bundle.EnableIf, bundleName)
			} else {
				fmt.Fprintf(&packaddContent, "packadd %s\n", bundleName)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(genDir, "packadd.vim"), []byte(packaddContent.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write packadd.vim: %w", err)
	}
	h.logger.Infof("runtimepath projection generated")
	h.logger.Debugf("generated file path: %s", filepath.Join(genDir, "packadd.vim"))

	// Copy hariti.lock to lock.json
	if err := os.WriteFile(filepath.Join(genDir, "lock.json"), lockBytes, 0644); err != nil {
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
	if err := os.WriteFile(filepath.Join(genDir, "metadata.json"), metaBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write metadata.json: %w", err)
	}

	// Switch current symlink atomically using temporary symlink and rename
	currentPath := h.CurrentSymlinkPath()
	tempSymlink := filepath.Join(filepath.Dir(currentPath), "current.tmp")
	_ = os.Remove(tempSymlink) // remove stale temp symlink

	// Point to the relative or absolute target
	targetPath := filepath.Join("generations", genID)
	if err := createGenerationLink(targetPath, tempSymlink); err != nil {
		return "", fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	if err := os.Rename(tempSymlink, currentPath); err != nil {
		// Fallback for environments where atomic rename of symlink has OS constraints
		_ = os.Remove(currentPath)
		if err := createGenerationLink(targetPath, currentPath); err != nil {
			return "", fmt.Errorf("failed to switch current symlink: %w", err)
		}
	}
	h.logger.Infof("current generation switched to: %s", genID)

	return genID, nil
}

func (h *Hariti) buildHelpTags(bundleID, docDir string) error {
	info, err := os.Stat(docDir)
	if err != nil {
		if os.IsNotExist(err) {
			// doc/ does not exist => skip silently
			return nil
		}
		return fmt.Errorf("failed to inspect doc path for bundle %s inside %s: %w", bundleID, docDir, err)
	}
	if !info.IsDir() {
		// doc exists but is not directory => WARN and skip
		h.logger.Warnf("bundle %s: expected %s to be a directory, skipping help tag generation", bundleID, docDir)
		return nil
	}

	h.logger.Debugf("generating help tags for bundle %s inside %s", bundleID, docDir)
	escapedCmd := fmt.Sprintf("execute 'helptags' fnameescape(%q)", filepath.ToSlash(docDir))
	cmd := exec.Command("vim", "-Nu", "NONE", "-n", "-e", "-s", "-c", escapedCmd, "-c", "qa!")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate help tags for bundle %s inside %s: %w (output: %s)", bundleID, docDir, err, string(out))
	}
	return nil
}
