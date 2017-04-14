package hariti

import (
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	cases := []struct {
		Input  string
		Output []string
	}{
		{"use vim-hariti", []string{"use", "vim-hariti"}},
		{"use kamichidu/vim-hariti", []string{"use", "kamichidu/vim-hariti"}},
		{"use https://github.com/scripts/CSApprox", []string{"use", "https://github.com/scripts/CSApprox"}},
		{"use https://bitbucket.org/kamichidu/vim-hariti", []string{"use", "https://bitbucket.org/kamichidu/vim-hariti"}},
		{"use local $HOME/sources/vim-plugins/", []string{"use", "local", "$HOME/sources/vim-plugins/"}},
		{`use kamichidu/vim-hariti
			as hariti
			depends (
				hoge
				hoge/fuga
				https://github.com/hoge/fuga
			)
			enable_if "!has('gui_running')"
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
					- *first universal
					- *second universal
			}`, []string{
			"use", "kamichidu/vim-hariti",
			"as", "hariti",
			"depends", "(", "hoge", "hoge/fuga", "https://github.com/hoge/fuga", ")",
			"enable_if", "\"!has('gui_running')\"",
			"build", "{",
			"on", "windows", "-", "w first", "\n", "-", "w second", "\n",
			"on", "mac", "-", "m first", "\n", "-", "m second", "\n",
			"on", "unix", "-", "u first", "\n", "-", "u second", "\n",
			"on", "*", "-", "*first universal", "\n", "-", "*second universal", "\n",
			"}"},
		},
	}
	for _, tc := range cases {
		i := 0
		l := NewLexer()
		l.Init(strings.NewReader(tc.Input))
		for {
			tok := l.Scan()
			if tok == EOF {
				break
			}

			t.Logf("token = %s\n", l.TokenText())
			if i >= len(tc.Output) {
				t.Fatal("Index overflow")
			}
			if l.TokenText() != tc.Output[i] {
				t.Errorf("Expected %#v, but got %#v\nBailing out now", tc.Output[i], l.TokenText())
				break
			}
			i++
		}
		if i != len(tc.Output) {
			t.Errorf("Didn't consume %#v", tc.Output[i+1:])
		}
		if len(l.state) != 1 {
			t.Errorf("Didn't manage state correctly, final state %#v", l.state)
		}
	}
}

func TestLex(t *testing.T) {
	cases := []struct {
		Input  string
		Output []Token
	}{
		{"use vim-hariti", []Token{
			Token{Use, "use"}, Token{Ident, "vim-hariti"},
		}},
		{"use kamichidu/vim-hariti", []Token{
			Token{Use, "use"}, Token{Ident, "kamichidu/vim-hariti"},
		}},
		{"use https://github.com/scripts/CSApprox", []Token{
			Token{Use, "use"}, Token{Ident, "https://github.com/scripts/CSApprox"},
		}},
		{"use https://bitbucket.org/kamichidu/vim-hariti", []Token{
			Token{Use, "use"}, Token{Ident, "https://bitbucket.org/kamichidu/vim-hariti"},
		}},
		{"use local $HOME/sources/vim-plugins/", []Token{
			Token{Use, "use"}, Token{Local, "local"}, Token{Ident, "$HOME/sources/vim-plugins/"},
		}},
		{`use kamichidu/vim-hariti
			as hariti
			depends (
				hoge
				hoge/fuga
				https://github.com/hoge/fuga
			)
			enable_if "!has('gui_running')"
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
					- *first universal
					- *second universal
			}`, []Token{
			Token{Use, "use"},
			Token{Ident, "kamichidu/vim-hariti"},
			Token{As, "as"},
			Token{Ident, "hariti"},
			Token{Depends, "depends"},
			Token{'(', "("},
			Token{Ident, "hoge"},
			Token{Ident, "hoge/fuga"},
			Token{Ident, "https://github.com/hoge/fuga"},
			Token{')', ")"},
			Token{EnableIf, "enable_if"},
			Token{String, "\"!has('gui_running')\""},
			Token{Build, "build"},
			Token{'{', "{"},
			Token{On, "on"},
			Token{OSType, "windows"},
			Token{'-', "-"},
			Token{Ident, "w first"},
			Token{'\n', "\n"},
			Token{'-', "-"},
			Token{Ident, "w second"},
			Token{'\n', "\n"},
			Token{On, "on"},
			Token{OSType, "mac"},
			Token{'-', "-"},
			Token{Ident, "m first"},
			Token{'\n', "\n"},
			Token{'-', "-"},
			Token{Ident, "m second"},
			Token{'\n', "\n"},
			Token{On, "on"},
			Token{OSType, "unix"},
			Token{'-', "-"},
			Token{Ident, "u first"},
			Token{'\n', "\n"},
			Token{'-', "-"},
			Token{Ident, "u second"},
			Token{'\n', "\n"},
			Token{On, "on"},
			Token{OSType, "*"},
			Token{'-', "-"},
			Token{Ident, "*first universal"},
			Token{'\n', "\n"},
			Token{'-', "-"},
			Token{Ident, "*second universal"},
			Token{'\n', "\n"},
			Token{'}', "}"},
		}},
	}
	for _, tc := range cases {
		i := 0
		l := NewLexer()
		l.Init(strings.NewReader(tc.Input))
		for {
			lval := &yySymType{}
			tok := l.Lex(lval)
			if tok == EOF {
				break
			}

			t.Logf("token = %#v\n", lval.tok)
			if i >= len(tc.Output) {
				t.Fatal("Index overflow")
			}
			if lval.tok != tc.Output[i] {
				t.Errorf("Expected %#v, but got %#v\nBailing out now", tc.Output[i], lval.tok)
				break
			}
			i++
		}
		if i != len(tc.Output) {
			t.Errorf("Didn't consume %#v", tc.Output[i+1:])
		}
	}
}
