package hariti_test

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti/internal/config/yaml"
	"github.com/kr/pretty"
	"github.com/pmezard/go-difflib/difflib"
)

func TestMarshalBundles(t *testing.T) {
	bundles := yaml.Bundles{
		yaml.Bundle{
			Remote: &yaml.RemoteBundle{
				Name:         "fuga",
				Aliases:      []string{"as"},
				Dependencies: []*yaml.RemoteBundle{},
			},
		},
		yaml.Bundle{
			Local: &yaml.LocalBundle{
				Path: "~/sources/vim-plugins/vim-hariti/",
			},
		},
	}

	buffer := new(bytes.Buffer)
	err := yaml.MarshalBundles(buffer, bundles)
	if err != nil {
		t.Fatal(err)
	}

	expects := strings.Join([]string{
		`---`,
		`version: "0.0"`,
		`bundles:`,
		`- name: fuga`,
		`  aliases:`,
		`  - as`,
		`- path: ~/sources/vim-plugins/vim-hariti/`,
		``,
	}, "\n")
	if buffer.String() != expects {
		diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: difflib.SplitLines(buffer.String()), B: difflib.SplitLines(expects)})
		t.Errorf("YAML differ:\n%s", diff)
	}
}

func TestUnmarshalBundles(t *testing.T) {
	r, err := os.Open("./testdata/bundles.yml")
	if err != nil {
		t.Fatalf("Open: %s", err)
	}
	defer r.Close()

	bundles, err := yaml.UnmarshalBundles(r)
	if err != nil {
		t.Fatalf("UnmarshalBundles: %s", err)
	}
	expected := yaml.Bundles{
		yaml.Bundle{
			Remote: &yaml.RemoteBundle{
				Name:    "Shougo/vimproc.vim",
				Aliases: []string{"vimproc"},
				BuildScript: &yaml.BuildScript{
					Windows: "mingw32-make -f make_mingw64.mak",
					Mac:     "make -f make_mac.mak",
					Linux:   "make -f make_unix.mak",
					All:     "echo all",
				},
			},
		},
		yaml.Bundle{
			Remote: &yaml.RemoteBundle{
				Name:    "Shougo/unite.vim",
				Aliases: []string{"unite"},
			},
		},
		yaml.Bundle{
			Remote: &yaml.RemoteBundle{
				Name:    "osyo-manga/vim-watchdogs",
				Aliases: []string{"watchdogs"},
				Dependencies: []*yaml.RemoteBundle{
					&yaml.RemoteBundle{Name: "thinca/vim-quickrun"},
					&yaml.RemoteBundle{Name: "Shougo/vimproc.vim"},
					&yaml.RemoteBundle{Name: "osyo-manga/shabadou.vim"},
					&yaml.RemoteBundle{Name: "jceb/vim-hier"},
					&yaml.RemoteBundle{Name: "dannyob/quickfixstatus"},
				},
			},
		},
		yaml.Bundle{
			Remote: &yaml.RemoteBundle{
				Name:         "godlygeek/csapprox",
				Aliases:      []string{"csapprox"},
				EnableIfExpr: "!has('gui_running')",
			},
		},
		yaml.Bundle{
			Local: &yaml.LocalBundle{
				Path: "~/sources/vim-hariti/",
			},
		},
	}
	if !reflect.DeepEqual(bundles, expected) {
		t.Errorf("Bundles differ:\n%s", strings.Join(pretty.Diff(bundles, expected), "\n"))
	}
}
