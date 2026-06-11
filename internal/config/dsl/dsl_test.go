package dsl_test

import (
	"reflect"
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
				EnableIf: "!has('gui_running')",
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
				},
				Dependencies: []string{},
				EnableIf:     "",
				Build:        []graph.BuildStep{},
				Aliases:      []string{"unite"},
			},
		},
		Replaces: []graph.Replacement{},
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
