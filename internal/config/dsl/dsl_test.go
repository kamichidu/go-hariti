package dsl_test

import (
	"reflect"
	"testing"

	"github.com/kamichidu/go-hariti/internal/config/dsl"
	"github.com/kamichidu/go-hariti/internal/config/dsl/ast"
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
