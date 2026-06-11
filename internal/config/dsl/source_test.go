package dsl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kamichidu/go-hariti/internal/config/dsl"
	"github.com/kamichidu/go-hariti/internal/graph"
)

func TestResolveSource_Local(t *testing.T) {
	_ = os.Setenv("TEST_HARITI_VAR", "my-env-var")
	defer func() {
		_ = os.Unsetenv("TEST_HARITI_VAR")
	}()

	tests := []struct {
		name     string
		expr     string
		expected graph.Source
	}{
		{
			name: "absolute path",
			expr: "/path/to/plugin",
			expected: graph.Source{
				Type: graph.SourceTypeLocal,
				Path: "/path/to/plugin",
			},
		},
		{
			name: "relative path",
			expr: "./plugins/foo",
			expected: graph.Source{
				Type: graph.SourceTypeLocal,
				Path: "./plugins/foo",
			},
		},
		{
			name: "parent relative path",
			expr: "../plugins/foo",
			expected: graph.Source{
				Type: graph.SourceTypeLocal,
				Path: "../plugins/foo",
			},
		},
		{
			name: "env var starting with $",
			expr: "$TEST_HARITI_VAR/foo",
			expected: graph.Source{
				Type: graph.SourceTypeLocal,
				Path: "my-env-var/foo",
			},
		},
		{
			name: "env var starting with ${}",
			expr: "${TEST_HARITI_VAR}/foo",
			expected: graph.Source{
				Type: graph.SourceTypeLocal,
				Path: "my-env-var/foo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := dsl.ResolveSource(tt.expr)
			if err != nil {
				t.Fatalf("ResolveSource error: %v", err)
			}
			if res.Type != tt.expected.Type {
				t.Errorf("expected Type %v, got %v", tt.expected.Type, res.Type)
			}
			if res.Path != tt.expected.Path {
				t.Errorf("expected Path %v, got %v", tt.expected.Path, res.Path)
			}
		})
	}
}

func TestResolveSource_HomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home: %v", err)
	}

	res, err := dsl.ResolveSource("~/src/foo")
	if err != nil {
		t.Fatalf("ResolveSource error: %v", err)
	}

	expectedPath := filepath.Join(home, "src/foo")
	if res.Type != graph.SourceTypeLocal {
		t.Errorf("expected Type local, got %s", res.Type)
	}
	if res.Path != expectedPath {
		t.Errorf("expected Path %s, got %s", expectedPath, res.Path)
	}
}

func TestResolveSource_Remote(t *testing.T) {
	tests := []struct {
		name        string
		expr        string
		expectedURL string
	}{
		{
			name:        "https URL",
			expr:        "https://github.com/foo/bar",
			expectedURL: "https://github.com/foo/bar",
		},
		{
			name:        "http URL",
			expr:        "http://example.com/foo/bar",
			expectedURL: "http://example.com/foo/bar",
		},
		{
			name:        "ssh URL",
			expr:        "ssh://git@example.com/foo/bar",
			expectedURL: "ssh://git@example.com/foo/bar",
		},
		{
			name:        "Git SSH shorthand",
			expr:        "git@github.com:foo/bar.git",
			expectedURL: "ssh://git@github.com/foo/bar.git",
		},
		{
			name:        "GitHub shorthand",
			expr:        "foo/bar",
			expectedURL: "https://github.com/foo/bar",
		},
		{
			name:        "Vim.org shorthand",
			expr:        "vim-hariti",
			expectedURL: "https://github.com/vim-scripts/vim-hariti",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := dsl.ResolveSource(tt.expr)
			if err != nil {
				t.Fatalf("ResolveSource error: %v", err)
			}
			if res.Type != graph.SourceTypeRemote {
				t.Errorf("expected Type remote, got %s", res.Type)
			}
			if res.URL == nil {
				t.Fatal("expected non-nil URL")
			}
			if res.URL.String() != tt.expectedURL {
				t.Errorf("expected URL %s, got %s", tt.expectedURL, res.URL.String())
			}
		})
	}
}

func TestResolveSource_Errors(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "GitHub shorthand with multiple slashes",
			expr: "foo/bar/baz",
		},
		{
			name: "ambiguous env var in remote path",
			expr: "owner/$REPO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dsl.ResolveSource(tt.expr)
			if err == nil {
				t.Errorf("expected error for %s, but got nil", tt.expr)
			}
		})
	}
}
