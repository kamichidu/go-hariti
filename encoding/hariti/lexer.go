package hariti

import (
	"log"
	"text/scanner"
)

const debug = true

const (
	EOF = scanner.EOF
)

type Lexer struct {
	scanner.Scanner
	state   []lexerState
	Bundles []Bundle
}

type lexerState interface {
	Tag() string
	Whitespace() uint64
	IsIdentRune(rune, int) bool
}

type defaultState struct {
	Lexer *Lexer
}

func (s *defaultState) Tag() string {
	return "default"
}

func (s *defaultState) Whitespace() uint64 {
	return scanner.GoWhitespace
}

func (s *defaultState) IsIdentRune(ch rune, i int) bool {
	switch {
	case ch >= 'a' && ch <= 'z':
		return true
	case ch >= 'A' && ch <= 'Z':
		return true
	case i > 0 && ch >= '0' && ch <= '9':
		return true
	case ch == '_' || ch == '-':
		return true
	default:
		return false
	}
}

var _ lexerState = (*defaultState)(nil)

type bareStringState struct {
	Lexer *Lexer
}

func (s *bareStringState) Tag() string {
	return "bare-string"
}

func (s *bareStringState) Whitespace() uint64 {
	return scanner.GoWhitespace
}

func (s *bareStringState) IsIdentRune(ch rune, i int) bool {
	switch ch {
	case ' ', '\t', '\r', '\n', scanner.EOF:
		return false
	default:
		return true
	}
}

var _ lexerState = (*bareStringState)(nil)

type barePathStringState struct {
	Lexer  *Lexer
	escape bool
}

func (s *barePathStringState) Tag() string {
	return "bare-path-string"
}
func (s *barePathStringState) Whitespace() uint64 {
	return scanner.GoWhitespace
}

func (s *barePathStringState) IsIdentRune(ch rune, i int) bool {
	switch ch {
	case '\\':
		s.escape = true
		return true
	case ' ', '\r', '\n':
		escaped := s.escape
		s.escape = false
		return escaped
	case scanner.EOF:
		return false
	default:
		return true
	}
}

var _ lexerState = (*barePathStringState)(nil)

type rawScriptState struct {
	Lexer *Lexer
}

func (s *rawScriptState) Tag() string {
	return "raw-script"
}

func (s *rawScriptState) Whitespace() uint64 {
	return 1<<'\r' | 1<<' '
}

func (s *rawScriptState) IsIdentRune(ch rune, i int) bool {
	switch ch {
	case '\n', scanner.EOF:
		return false
	default:
		return true
	}
}

var _ lexerState = (*rawScriptState)(nil)

func NewLexer() *Lexer {
	l := &Lexer{
		state: []lexerState{},
	}
	l.Mode = scanner.ScanIdents
	l.pushState("default")
	return l
}

func (s *Lexer) Scan() rune {
	tok := s.Scanner.Scan()

	switch s.stateTag() {
	case "default":
		switch s.TokenText() {
		case "use", "(":
			s.pushState("bare-string")
		case "-":
			s.pushState("raw-script")
		}
	case "bare-string":
		switch s.TokenText() {
		case "local":
			s.pushState("bare-path-string")
		case "as", "enable_if", "depends", "build": // following candidates for repository
			s.popState()
		case ")":
			s.popState()
		}
	case "bare-path-string":
		s.popState()
	case "raw-script":
		switch s.TokenText() {
		case "\n":
			s.popState()
		}
	}
	// for `use kamichidu` like syntax, in this case bare-string state always on the stack
	if tok == scanner.EOF && len(s.state) > 1 {
		s.popState()
	}

	return tok
}

func (s *Lexer) stateTag() string {
	return s.state[len(s.state)-1].Tag()
}

func (s *Lexer) pushState(tag string) {
	var state lexerState
	switch tag {
	case "default":
		state = &defaultState{
			Lexer: s,
		}
	case "bare-string":
		state = &bareStringState{
			Lexer: s,
		}
	case "bare-path-string":
		state = &barePathStringState{
			Lexer: s,
		}
	case "raw-script":
		state = &rawScriptState{
			Lexer: s,
		}
	default:
		log.Panicf("illegal state tag: %s\n", tag)
	}
	s.state = append(s.state, state)
	s.Whitespace = state.Whitespace()
	s.IsIdentRune = state.IsIdentRune
}

func (s *Lexer) popState() {
	s.state = s.state[:len(s.state)-1]
	if len(s.state) == 0 {
		log.Panicln("illegal state")
	}
	state := s.state[len(s.state)-1]
	s.Whitespace = state.Whitespace()
	s.IsIdentRune = state.IsIdentRune
}

type Token struct {
	Id   int
	Text string
}

var tokens = map[string]Token{
	"(":         Token{'(', "("},
	")":         Token{')', ")"},
	"{":         Token{'{', "{"},
	"}":         Token{'}', "}"},
	"-":         Token{'-', "-"},
	"\n":        Token{'\n', "\n"},
	"use":       Token{Use, "use"},
	"local":     Token{Local, "local"},
	"as":        Token{As, "as"},
	"depends":   Token{Depends, "depends"},
	"enable_if": Token{EnableIf, "enable_if"},
	"build":     Token{Build, "build"},
	"on":        Token{On, "on"},
	"windows":   Token{OSType, "windows"},
	"mac":       Token{OSType, "mac"},
	"unix":      Token{OSType, "unix"},
	"*":         Token{OSType, "*"},
}

func (s *Lexer) Lex(lval *yySymType) int {
	if debug {
		log.Printf("use state %#v", s.stateTag())
	}
	tok := s.Scan()
	if debug {
		log.Printf("\ttok = %#v, token = %#v\n", tok, s.TokenText())
	}
	if tok == EOF {
		lval.tok = Token{
			Id:   EOF,
			Text: "",
		}
		return EOF
	}
	text := s.TokenText()
	if token, found := tokens[text]; found {
		lval.tok = token
		return lval.tok.Id
	}
	if len(text) > 0 && text[0] == '\'' || text[0] == '"' {
		lval.tok = Token{
			Id:   String,
			Text: text,
		}
	} else {
		lval.tok = Token{
			Id:   Ident,
			Text: text,
		}
	}
	return lval.tok.Id
}

func (s *Lexer) Error(msg string) {
	log.Printf("%s: %s\n", s.String(), msg)
}

var _ yyLexer = (*Lexer)(nil)
