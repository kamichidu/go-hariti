package yaml_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti/internal/config/yaml"
	"github.com/kamichidu/go-hariti/internal/graph"
)

func TestToGraph(t *testing.T) {
	yamlStr := `
version: "0.0"
bundles:
  - name: Shougo/vimproc.vim
    aliases: [vimproc]
    build:
      windows: mingw32-make -f make_mingw64.mak
      mac:     make -f make_mac.mak
      linux:   make -f make_unix.mak
      all:     echo all
  - path: ~/sources/vim-hariti/
`

	bundles, err := yaml.UnmarshalBundles(strings.NewReader(yamlStr))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	bundlesFile := &yaml.BundlesFile{
		Version: "0.0",
		Bundles: bundles,
	}

	g := bundlesFile.ToGraph()

	expected := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "Shougo/vimproc.vim",
				Source: graph.Source{
					Type: graph.SourceTypeRemote,
					URL:  nil,
				},
				Dependencies: nil,
				EnableIf:     "",
				Build: []graph.BuildStep{
					{OS: "windows", Cmd: "mingw32-make -f make_mingw64.mak"},
					{OS: "mac", Cmd: "make -f make_mac.mak"},
					{OS: "linux", Cmd: "make -f make_unix.mak"},
					{OS: "all", Cmd: "echo all"},
				},
				Aliases: []string{"vimproc"},
			},
			{
				ID: "~/sources/vim-hariti/",
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: "~/sources/vim-hariti/",
				},
				Dependencies: nil,
				Build:        nil,
				Aliases:      nil,
			},
		},
		Replaces: []graph.Replacement{},
	}

	if !reflect.DeepEqual(g, expected) {
		t.Errorf("expected graph %+v, got %+v", expected, g)
	}
}
