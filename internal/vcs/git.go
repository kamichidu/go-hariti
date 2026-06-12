package vcs

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/internal/graph"
)

type Git struct{}

func (self *Git) Clone(ctx context.Context, bundle graph.Bundle, update bool, out, errOut io.Writer, logger Logger) error {
	var cmd *exec.Cmd
	localPath := bundle.Source.Path
	urlStr := ""
	if bundle.Source.URL != nil {
		urlStr = bundle.Source.URL.String()
	}

	if info, err := os.Stat(localPath); err != nil {
		if logger != nil {
			logger.Infof("Cloning %s to %s\n", urlStr, localPath)
		}
		cmd = exec.Command("git", "clone", "--recursive", urlStr, localPath)
		cmd.Stdout = out
		cmd.Stderr = errOut
	} else if info.IsDir() && update {
		if logger != nil {
			logger.Infof("Pulling in %s", localPath)
		}
		cmd = exec.Command("git", "pull", "--ff", "--ff-only")
		cmd.Dir = localPath
		cmd.Stdout = out
		cmd.Stderr = errOut
	} else {
		return nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (self *Git) IsModified(ctx context.Context, bundle graph.Bundle, out, errOut io.Writer) (bool, error) {
	localPath := bundle.Source.Path
	urlStr := ""
	if bundle.Source.URL != nil {
		urlStr = bundle.Source.URL.String()
	}

	var cmd *exec.Cmd
	if info, err := os.Stat(localPath); err != nil {
		return false, fmt.Errorf("Repository %s not cloned into %s", urlStr, localPath)
	} else if !info.IsDir() {
		return false, fmt.Errorf("%s doesn't seems like a repository %s", localPath, urlStr)
	} else {
		cmd = exec.Command("git", "diff", "--exit-code")
		cmd.Dir = localPath
		cmd.Stdout = out
		cmd.Stderr = errOut
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return false, ctx.Err()
	case err := <-errCh:
		if err != nil {
			return true, err
		} else {
			return false, nil
		}
	}
}

func (self *Git) CanHandle(ctx context.Context, urlStr string) bool {
	cmd := exec.Command("git", "ls-remote", urlStr)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return false
	case err := <-errCh:
		return err == nil
	}
}

func (self *Git) HeadRevision(ctx context.Context, bundle graph.Bundle, out, errOut io.Writer) (string, error) {
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
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", ctx.Err()
	case err := <-errCh:
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(stdout.String()), nil
	}
}

func (self *Git) Archive(ctx context.Context, bundle graph.Bundle, revision string, destDir string, errOut io.Writer) error {
	localPath := bundle.Source.Path

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
		if err == io.EOF {
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

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			outFile.Close()
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive execution failed: %w", err)
	}

	return nil
}

type Logger interface {
	Infof(format string, args ...interface{})
}
