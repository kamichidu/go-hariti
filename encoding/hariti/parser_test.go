package hariti

import (
	"reflect"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	lexer := NewLexer()
	lexer.Init(strings.NewReader(`
		use kamichidu/vim-hariti
		  as hariti
		  depends ()
		use local $HOME/sources/vim-plugins/
		use kamichidu/vim-hariti
		  as hariti
		  depends (
		    hoge
			fuga/hoge
			https://github.com/kamichidu/go-hariti
		  )
		  build {
			on windows
			  - w first
			  - w second
			on mac
			  - m first
			  - m second
			on unix
			  - u first
			  - u second
			on *
			  - * first
			  - * second
		  }
	`))
	// yyDebug = 5
	res := yyParse(lexer)
	t.Logf("res = %#v\n", res)
	if res != 0 {
		t.Error("Parse failed")
	}
	t.Log("bundles:")
	if len(lexer.Bundles) != 3 {
		t.Errorf("Expected 3 bundles, but got %d bundles", len(lexer.Bundles))
	}

	expects := []Bundle{
		&RemoteBundle{
			Uri:          "kamichidu/vim-hariti",
			Aliases:      []string{"hariti"},
			Dependencies: []string{},
			EnableIfExpr: "",
			BuildScripts: make(map[string][]string, 0),
		},
		&LocalBundle{
			Uri: "$HOME/sources/vim-plugins/",
		},
		&RemoteBundle{
			Uri:          "kamichidu/vim-hariti",
			Aliases:      []string{"hariti"},
			Dependencies: []string{"hoge", "fuga/hoge", "https://github.com/kamichidu/go-hariti"},
			EnableIfExpr: "",
			BuildScripts: map[string][]string{
				"windows": []string{"w first", "w second"},
				"mac":     []string{"m first", "m second"},
				"unix":    []string{"u first", "u second"},
				"*":       []string{"* first", "* second"},
			},
		},
	}
	for i, bundle := range lexer.Bundles {
		t.Logf("\t%2d: %#v\n", i, bundle)
		if len(expects) > i && !reflect.DeepEqual(bundle, expects[i]) {
			t.Errorf("\nExpected\t%#v\nActual\t\t%#v", expects[i], bundle)
		}
	}
}
