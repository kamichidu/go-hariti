package dsl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamichidu/go-hariti/internal/config/dsl/ast"
	"github.com/kamichidu/go-hariti/internal/graph"
)

// Loader handles recursive parsing of .hariti files with relative path resolution and circular dependency detection.
type Loader struct {
	visited map[string]bool
}

func NewLoader() *Loader {
	return &Loader{
		visited: make(map[string]bool),
	}
}

func (l *Loader) Load(path string) (*ast.File, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	if l.visited[absPath] {
		return nil, fmt.Errorf("circular include detected for file: %s", absPath)
	}

	l.visited[absPath] = true
	defer func() {
		l.visited[absPath] = false
	}()

	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", absPath, err)
	}

	file, err := Parse(absPath, src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", absPath, err)
	}

	merged := &ast.File{
		Bundles: make([]ast.BundleDecl, 0, len(file.Bundles)),
	}

	// Copy direct bundles
	merged.Bundles = append(merged.Bundles, file.Bundles...)

	// Recursively resolve includes
	dir := filepath.Dir(absPath)
	for _, inc := range file.Includes {
		var targetPath string
		if filepath.IsAbs(inc.Path) {
			targetPath = inc.Path
		} else {
			targetPath = filepath.Join(dir, inc.Path)
		}

		if strings.ContainsAny(inc.Path, "*?[]") {
			matches, err := filepath.Glob(targetPath)
			if err != nil {
				return nil, fmt.Errorf("failed to expand glob pattern %s in %s: %w", inc.Path, absPath, err)
			}
			for _, match := range matches {
				incFile, err := l.Load(match)
				if err != nil {
					return nil, err
				}
				merged.Bundles = append(merged.Bundles, incFile.Bundles...)
			}
		} else {
			incFile, err := l.Load(targetPath)
			if err != nil {
				return nil, err
			}
			merged.Bundles = append(merged.Bundles, incFile.Bundles...)
		}
	}

	return merged, nil
}

func LoadGraph(path string) (*graph.Graph, error) {
	loader := NewLoader()
	file, err := loader.Load(path)
	if err != nil {
		return nil, err
	}
	return ToGraph(file)
}
