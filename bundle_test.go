package hariti

import (
	"bytes"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/pmezard/go-difflib/difflib"
)

func TestMarshalBundles(t *testing.T) {
	bundles := Bundles{
		&RemoteBundle{
			Name:         "fuga",
			URL:          &url.URL{RawPath: "hoge/fuga"},
			LocalPath:    "lp",
			Aliases:      []string{"as"},
			Dependencies: []*RemoteBundle{},
		},
		&LocalBundle{
			LocalPath: "~/sources/vim-plugins/vim-hariti/",
		},
	}

	buffer := new(bytes.Buffer)
	err := MarshalBundles(buffer, bundles)
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

	bundles, err := UnmarshalBundles(r)
	if err != nil {
		t.Fatalf("UnmarshalBundles: %s", err)
	}
	expected := Bundles{
		&RemoteBundle{
			Name:    "Shougo/vimproc.vim",
			Aliases: []string{"vimproc"},
			BuildScript: &BuildScript{
				Windows: "mingw32-make -f make_mingw64.mak",
				Mac:     "make -f make_mac.mak",
				Linux:   "make -f make_unix.mak",
				All:     "echo all",
			},
		},
		&RemoteBundle{
			Name:    "Shougo/unite.vim",
			Aliases: []string{"unite"},
		},
		&RemoteBundle{
			Name:    "osyo-manga/vim-watchdogs",
			Aliases: []string{"watchdogs"},
			Dependencies: []*RemoteBundle{
				&RemoteBundle{Name: "thinca/vim-quickrun"},
				&RemoteBundle{Name: "Shougo/vimproc.vim"},
				&RemoteBundle{Name: "osyo-manga/shabadou.vim"},
				&RemoteBundle{Name: "jceb/vim-hier"},
				&RemoteBundle{Name: "dannyob/quickfixstatus"},
			},
		},
		&RemoteBundle{
			Name:         "godlygeek/csapprox",
			Aliases:      []string{"csapprox"},
			EnableIfExpr: "!has('gui_running')",
		},
		&LocalBundle{
			LocalPath: "~/sources/vim-hariti/",
		},
	}
	if !reflect.DeepEqual(bundles, expected) {
		t.Errorf("Bundles differ:\n%s", strings.Join(pretty.Diff(bundles, expected), "\n"))
	}
}
