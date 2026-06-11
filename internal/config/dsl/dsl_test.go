package dsl_test

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti/internal/config/dsl"
	"github.com/kamichidu/go-hariti/internal/config/dsl/ast"
	"github.com/kamichidu/go-hariti/internal/graph"
)

func TestParse_Use(t *testing.T) {
	src := `use Shougo/vimproc.vim`
	f, err := dsl.Parse("", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := &ast.File{
		Bundles: []ast.BundleDecl{
			{
				Use: "Shougo/vimproc.vim",
			},
		},
	}

	if !reflect.DeepEqual(f, expected) {
		t.Errorf("expected %+v, got %+v", expected, f)
	}
}

func TestParse_As(t *testing.T) {
	src := `use Shougo/vimproc.vim
  as vimproc`
	f, err := dsl.Parse("", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := &ast.File{
		Bundles: []ast.BundleDecl{
			{
				Use:     "Shougo/vimproc.vim",
				Aliases: []string{"vimproc"},
			},
		},
	}

	if !reflect.DeepEqual(f, expected) {
		t.Errorf("expected %+v, got %+v", expected, f)
	}
}

func TestParse_Depends(t *testing.T) {
	src := `use osyo-manga/vim-watchdogs
  depends (
    thinca/vim-quickrun
    Shougo/vimproc.vim
  )`
	f, err := dsl.Parse("", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := &ast.File{
		Bundles: []ast.BundleDecl{
			{
				Use:     "osyo-manga/vim-watchdogs",
				Depends: []string{"thinca/vim-quickrun", "Shougo/vimproc.vim"},
			},
		},
	}

	if !reflect.DeepEqual(f, expected) {
		t.Errorf("expected %+v, got %+v", expected, f)
	}
}

func TestParse_EnableIf(t *testing.T) {
	src := `use godlygeek/csapprox
  enable_if "!has('gui_running')"`
	f, err := dsl.Parse("", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := &ast.File{
		Bundles: []ast.BundleDecl{
			{
				Use:      "godlygeek/csapprox",
				EnableIf: func() *string { s := "!has('gui_running')"; return &s }(),
			},
		},
	}

	if !reflect.DeepEqual(f, expected) {
		t.Errorf("expected %+v, got %+v", expected, f)
	}
}

func TestParse_Build(t *testing.T) {
	src := `use Shougo/vimproc.vim
  build {
    on linux
      - make -f make_unix.mak
    on *
      - echo all
  }`
	f, err := dsl.Parse("", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := &ast.File{
		Bundles: []ast.BundleDecl{
			{
				Use: "Shougo/vimproc.vim",
				Build: []ast.BuildBlock{
					{
						OS:       "linux",
						Commands: []string{"make -f make_unix.mak"},
					},
					{
						OS:       "*",
						Commands: []string{"echo all"},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(f, expected) {
		t.Errorf("expected %+v, got %+v", expected, f)
	}
}

func TestParse_MultipleBundles(t *testing.T) {
	src := `use Shougo/vimproc.vim
  as vimproc

use Shougo/unite.vim
  as unite`
	f, err := dsl.Parse("", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := &ast.File{
		Bundles: []ast.BundleDecl{
			{
				Use:     "Shougo/vimproc.vim",
				Aliases: []string{"vimproc"},
			},
			{
				Use:     "Shougo/unite.vim",
				Aliases: []string{"unite"},
			},
		},
	}

	if !reflect.DeepEqual(f, expected) {
		t.Errorf("expected %+v, got %+v", expected, f)
	}
}

func TestParseGraph_Success(t *testing.T) {
	src := `use Shougo/vimproc.vim
  as vimproc
  build {
    on linux
      - make -f make_unix.mak
    on *
      - echo all
  }

use Shougo/unite.vim
  as unite`

	g, err := dsl.ParseGraph("", []byte(src))
	if err != nil {
		t.Fatalf("ParseGraph error: %v", err)
	}

	expected := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "Shougo/vimproc.vim",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  func() *url.URL { u, _ := dsl.ResolveSource("Shougo/vimproc.vim"); return u.URL }(),
				},
				Dependencies: []string{},
				EnableIf:     "",
				Build: []graph.BuildStep{
					{OS: "linux", Cmd: "make -f make_unix.mak"},
					{OS: "all", Cmd: "echo all"},
				},
				Aliases: []string{"vimproc"},
			},
			{
				ID: "Shougo/unite.vim",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  func() *url.URL { u, _ := dsl.ResolveSource("Shougo/unite.vim"); return u.URL }(),
				},
				Dependencies: []string{},
				EnableIf:     "",
				Build:        []graph.BuildStep{},
				Aliases:      []string{"unite"},
			},
		},
	}

	if !reflect.DeepEqual(g, expected) {
		t.Errorf("expected graph %+v, got %+v", expected, g)
	}
}

func TestParseGraph_ValidationFail(t *testing.T) {
	// 1. empty bundle ID
	src1 := `use ""`
	_, err := dsl.ParseGraph("", []byte(src1))
	if err == nil {
		t.Error("expected empty ID validation error, but got none")
	}

	// 2. duplicate bundle ID
	src2 := `use Shougo/unite.vim
use Shougo/unite.vim`
	_, err = dsl.ParseGraph("", []byte(src2))
	if err == nil {
		t.Error("expected duplicate ID validation error, but got none")
	}
}

func TestParseGraph_UseBlockForm(t *testing.T) {
	src := `use my/local-plugin {
  source ./plugins/local-plugin
}`

	g, err := dsl.ParseGraph("", []byte(src))
	if err != nil {
		t.Fatalf("ParseGraph error: %v", err)
	}

	expected := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "my/local-plugin",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: "./plugins/local-plugin",
				},
				Dependencies: []string{},
				EnableIf:     "",
				Build:        []graph.BuildStep{},
				Aliases:      []string{},
			},
		},
	}

	if !reflect.DeepEqual(g, expected) {
		t.Errorf("expected graph %+v, got %+v", expected, g)
	}
}

func TestParseGraph_Replace(t *testing.T) {
	src := `use Shougo/vimproc.vim
  as vimproc
  depends (
    old/dep
  )

replace Shougo/vimproc.vim {
  source ./forks/vimproc
  depends (
    foo/bar
  )
}`

	g, err := dsl.ParseGraph("", []byte(src))
	if err != nil {
		t.Fatalf("ParseGraph error: %v", err)
	}

	expected := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "Shougo/vimproc.vim",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: "./forks/vimproc",
				},
				Dependencies: []string{"foo/bar"},
				EnableIf:     "",
				Build:        []graph.BuildStep{},
				Aliases:      []string{},
			},
		},
	}

	if !reflect.DeepEqual(g, expected) {
		t.Errorf("expected graph %+v, got %+v", expected, g)
	}
}

func TestParseGraph_Merge(t *testing.T) {
	src := `use Shougo/vimproc.vim
  as vimproc
  depends (
    old/dep
  )

merge Shougo/vimproc.vim {
  source ~/src/vimproc
}`

	g, err := dsl.ParseGraph("", []byte(src))
	if err != nil {
		t.Fatalf("ParseGraph error: %v", err)
	}

	expected := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "Shougo/vimproc.vim",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: filepath.Join(func() string { h, _ := os.UserHomeDir(); return h }(), "src/vimproc"),
				},
				Dependencies: []string{"old/dep"},
				EnableIf:     "",
				Build:        []graph.BuildStep{},
				Aliases:      []string{"vimproc"},
			},
		},
	}

	if !reflect.DeepEqual(g, expected) {
		t.Errorf("expected graph %+v, got %+v", expected, g)
	}
}

func TestParseGraph_MissingTarget(t *testing.T) {
	srcReplace := `replace missing/plugin { source ./x }`
	_, err := dsl.ParseGraph("", []byte(srcReplace))
	if err == nil {
		t.Error("expected replace missing target error, but got nil")
	}

	srcMerge := `merge missing/plugin { source ./x }`
	_, err = dsl.ParseGraph("", []byte(srcMerge))
	if err == nil {
		t.Error("expected merge missing target error, but got nil")
	}
}

func TestParseGraph_UseOptionalBlockSyntax(t *testing.T) {
	srcBraced := `use Shougo/vimproc.vim {
  source ./forks/vimproc
  as vimproc
}`

	srcUnbraced := `use Shougo/vimproc.vim
  source ./forks/vimproc
  as vimproc`

	gBraced, err := dsl.ParseGraph("", []byte(srcBraced))
	if err != nil {
		t.Fatalf("ParseGraph braced error: %v", err)
	}

	gUnbraced, err := dsl.ParseGraph("", []byte(srcUnbraced))
	if err != nil {
		t.Fatalf("ParseGraph unbraced error: %v", err)
	}

	if !reflect.DeepEqual(gBraced, gUnbraced) {
		t.Errorf("expected braced and unbraced forms to be identical, braced: %+v, unbraced: %+v", gBraced, gUnbraced)
	}
}

func TestParseGraph_TopLevelDirectiveBoundary(t *testing.T) {
	src := `use Shougo/vimproc.vim
  as vimproc

replace Shougo/vimproc.vim {
  source ./forks/vimproc
}`

	g, err := dsl.ParseGraph("", []byte(src))
	if err != nil {
		t.Fatalf("ParseGraph error: %v", err)
	}

	if len(g.Bundles) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(g.Bundles))
	}

	b := g.Bundles[0]
	if b.Source.Type != graph.SourceTypeLocal || b.Source.Path != "./forks/vimproc" {
		t.Errorf("replace directive was not applied correctly, got source: %+v", b.Source)
	}
}

func TestParseGraph_DumpGraphResolvedJSON(t *testing.T) {
	g := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "foo",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: "/path",
				},
				Dependencies: []string{},
				Build:        []graph.BuildStep{},
				Aliases:      []string{},
			},
		},
	}

	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("failed to marshal graph: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "replaces") {
		t.Errorf("expected marshaled JSON to exclude replaces field, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "merges") {
		t.Errorf("expected marshaled JSON to exclude merges, got: %s", jsonStr)
	}
}
