package git

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/graph"
	"github.com/kamichidu/go-hariti/vcs"
)

type Git struct{}

func runCmd(ctx context.Context, cmd *exec.Cmd) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			//nolint:errcheck // safe: process may already be dead during cancellation/timeout
			cmd.Process.Kill()
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (g *Git) Sync(c context.Context, bundle graph.Bundle) error {
	log := vcs.LoggerFromContext(c)
	out := vcs.WriterFromContext(c)
	errOut := vcs.ErrWriterFromContext(c)

	localPath := bundle.Source.Path
	urlStr := ""
	if bundle.Source.URL != nil {
		urlStr = bundle.Source.URL.String()
	}

	if log != nil {
		log.Debugf("Git sync started for bundle %s with URL %s in local path %q", bundle.ID, urlStr, localPath)
	}

	if info, err := os.Stat(localPath); err != nil {
		if log != nil {
			log.Debugf("Cloning %s to %s\n", urlStr, localPath)
		}
		cmd := exec.Command("git", "clone", "--recursive", urlStr, localPath)
		cmd.Stdout = out
		cmd.Stderr = errOut
		if err := runCmd(c, cmd); err != nil {
			return err
		}
	} else if info.IsDir() {
		if log != nil {
			log.Debugf("Fetching and hard resetting in %s", localPath)
		}

		// 1. Fetch all
		if log != nil {
			log.Debugf("Executing git fetch --all --prune in %s", localPath)
		}
		fetchCmd := exec.Command("git", "fetch", "--all", "--prune")
		fetchCmd.Dir = localPath
		fetchCmd.Stdout = out
		fetchCmd.Stderr = errOut
		if err := runCmd(c, fetchCmd); err != nil {
			return fmt.Errorf("git fetch failed: %w", err)
		}

		// 2. Resolve tracked upstream ref
		if log != nil {
			log.Debugf("Resolving tracked upstream ref in %s", localPath)
		}
		revParseCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
		revParseCmd.Dir = localPath
		var upstreamStdout bytes.Buffer
		revParseCmd.Stdout = &upstreamStdout
		revParseCmd.Stderr = errOut
		if err := runCmd(c, revParseCmd); err != nil {
			return fmt.Errorf("failed to resolve tracked upstream ref: %w", err)
		}
		upstream := strings.TrimSpace(upstreamStdout.String())
		if upstream == "" {
			return fmt.Errorf("no tracked upstream ref resolved for branch in %s", localPath)
		}

		// 3. Hard reset to resolved upstream ref
		if log != nil {
			log.Debugf("Executing git reset --hard %s in %s", upstream, localPath)
		}
		resetCmd := exec.Command("git", "reset", "--hard", upstream)
		resetCmd.Dir = localPath
		resetCmd.Stdout = out
		resetCmd.Stderr = errOut
		if err := runCmd(c, resetCmd); err != nil {
			return fmt.Errorf("git reset to %s failed: %w", upstream, err)
		}

		// 4. Update submodules conditionally if .gitmodules exists
		gitmodules := filepath.Join(localPath, ".gitmodules")
		if _, err := os.Stat(gitmodules); err == nil {
			if log != nil {
				log.Debugf("Updating submodules in %s", localPath)
			}
			submoduleCmd := exec.Command("git", "submodule", "update", "--init", "--recursive")
			submoduleCmd.Dir = localPath
			submoduleCmd.Stdout = out
			submoduleCmd.Stderr = errOut
			if err := runCmd(c, submoduleCmd); err != nil {
				return fmt.Errorf("git submodule update failed: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	} else {
		return nil
	}

	return nil
}

func (g *Git) CanHandle(c context.Context, u *url.URL) bool {
	if u == nil {
		return false
	}
	urlStr := u.String()
	cmd := exec.Command("git", "ls-remote", urlStr)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-c.Done():
		if cmd.Process != nil {
			//nolint:errcheck // safe: process may already be dead during cancellation/timeout
			cmd.Process.Kill()
		}
		return false
	case err := <-errCh:
		return err == nil
	}
}

func (g *Git) HeadRevision(c context.Context, bundle graph.Bundle) (string, error) {
	errOut := vcs.ErrWriterFromContext(c)
	localPath := bundle.Source.Path
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = localPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = errOut

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-c.Done():
		if cmd.Process != nil {
			//nolint:errcheck // safe: process may already be dead during cancellation/timeout
			cmd.Process.Kill()
		}
		return "", c.Err()
	case err := <-errCh:
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(stdout.String()), nil
	}
}

func (g *Git) Archive(c context.Context, bundle graph.Bundle, revision string, destDir string) error {
	log := vcs.LoggerFromContext(c)
	errOut := vcs.ErrWriterFromContext(c)
	localPath := bundle.Source.Path

	if log != nil {
		log.Debugf("Git archive started for bundle %s with revision %s in local path %q, destDir %q", bundle.ID, revision, localPath, destDir)
	}

	cmd := exec.Command("git", "archive", "--format=tar", revision)
	cmd.Dir = localPath
	cmd.Stderr = errOut

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start git archive: %w", err)
	}

	tarReader := tar.NewReader(stdoutPipe)
	for {
		header, err := tarReader.Next()
		if io.EOF == err {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		target := filepath.Join(destDir, filepath.Clean(header.Name))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			_, err = io.Copy(outFile, tarReader)
			closeErr := outFile.Close()
			if err != nil {
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			if closeErr != nil {
				return fmt.Errorf("failed to close file %s: %w", target, closeErr)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive execution failed: %w", err)
	}

	return nil
}

var _ vcs.VCS = (*Git)(nil)

func init() {
	vcs.Register(new(Git))
}
