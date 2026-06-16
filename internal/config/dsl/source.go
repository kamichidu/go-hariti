package dsl

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/graph"
)

func ResolveSource(expr string) (graph.Source, error) {
	// 1. Local path checks
	if strings.HasPrefix(expr, "/") ||
		strings.HasPrefix(expr, "./") ||
		strings.HasPrefix(expr, "../") ||
		strings.HasPrefix(expr, "~") ||
		strings.HasPrefix(expr, "$") {

		path := expr

		// Expand Home dir
		if strings.HasPrefix(path, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return graph.Source{}, fmt.Errorf("failed to get user home directory: %w", err)
			}
			path = filepath.Join(home, path[1:])
		}

		// Expand Env vars
		if strings.Contains(path, "$") {
			// Env vars must start with $ or ${
			if !strings.HasPrefix(path, "$") {
				return graph.Source{}, fmt.Errorf("unsupported ambiguous environment variable placement: %s", expr)
			}
			path = os.ExpandEnv(path)
		}

		return graph.Source{
			Type: graph.SourceTypeLocal,
			Path: path,
		}, nil
	}

	// 2. Reject ambiguous forms containing '$' that are not local paths
	if strings.Contains(expr, "$") {
		return graph.Source{}, fmt.Errorf("unsupported ambiguous environment variable placement: %s", expr)
	}

	// 3. Explicit Remote URL
	if strings.HasPrefix(expr, "http://") ||
		strings.HasPrefix(expr, "https://") ||
		strings.HasPrefix(expr, "ssh://") {

		parsed, err := url.ParseRequestURI(expr)
		if err != nil {
			return graph.Source{}, fmt.Errorf("invalid remote URL %s: %w", expr, err)
		}
		return graph.Source{
			Type: graph.SourceTypeRemote,
			URL:  parsed,
		}, nil
	}

	// 4. Git SSH shorthand
	if strings.HasPrefix(expr, "git@") {
		// Scp-like source git@github.com:owner/repo.git
		cleanExpr := strings.Replace(expr, ":", "/", 1)
		if !strings.HasPrefix(cleanExpr, "ssh://") {
			cleanExpr = "ssh://" + cleanExpr
		}
		parsed, err := url.Parse(cleanExpr)
		if err != nil {
			return graph.Source{}, fmt.Errorf("invalid git SSH shorthand %s: %w", expr, err)
		}
		return graph.Source{
			Type: graph.SourceTypeRemote,
			URL:  parsed,
		}, nil
	}

	// 5. GitHub shorthand
	slashCount := strings.Count(expr, "/")
	if slashCount == 1 {
		resolved := "https://github.com/" + expr
		parsed, err := url.ParseRequestURI(resolved)
		if err != nil {
			return graph.Source{}, fmt.Errorf("invalid GitHub shorthand %s: %w", expr, err)
		}
		return graph.Source{
			Type: graph.SourceTypeRemote,
			URL:  parsed,
		}, nil
	} else if slashCount > 1 {
		return graph.Source{}, fmt.Errorf("invalid GitHub shorthand %s: contains multiple slashes", expr)
	}

	// 6. Vim.org shorthand
	resolved := "https://github.com/vim-scripts/" + expr
	parsed, err := url.ParseRequestURI(resolved)
	if err != nil {
		return graph.Source{}, fmt.Errorf("invalid Vim.org shorthand %s: %w", expr, err)
	}
	return graph.Source{
		Type: graph.SourceTypeRemote,
		URL:  parsed,
	}, nil
}
