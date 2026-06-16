package hariti

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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
		//nolint:errcheck // safe: srcFile is read-only; closing error cannot affect file integrity or durability
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}

		_, err = io.Copy(destFile, srcFile)
		closeErr := destFile.Close()
		if err == nil {
			err = closeErr
		}
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
	return strings.EqualFold(stepOS, currentOS)
}

func (h *Hariti) Deploy(ctx context.Context, g *graph.Graph, opts DeployOptions) (string, error) {
	h.Logger.Infof("Starting generation deployment...")

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

			v := vcs.Detect(bundle.Source.URL)
			if v == nil {
				return "", fmt.Errorf("failed to detect VCS for remote bundle %s", bundle.ID)
			}

			vcsCtx := vcs.WithLogger(ctx, h.Logger)
			err := v.Archive(vcsCtx, bundle, revision, destDir)
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
				buildCmd.Stdout = io.Discard
				buildCmd.Stderr = io.Discard
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
				fmt.Fprintf(&packaddContent, "if %s\n  set runtimepath+=%s\n", bundle.EnableIf, localPath)
				afterPath := filepath.Join(localPath, "after")
				if _, err := os.Stat(afterPath); err == nil {
					fmt.Fprintf(&packaddContent, "  set runtimepath+=%s\n", afterPath)
				}
				packaddContent.WriteString("endif\n")
			} else {
				fmt.Fprintf(&packaddContent, "set runtimepath+=%s\n", localPath)
				afterPath := filepath.Join(localPath, "after")
				if _, err := os.Stat(afterPath); err == nil {
					fmt.Fprintf(&packaddContent, "set runtimepath+=%s\n", afterPath)
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

	return genID, nil
}
