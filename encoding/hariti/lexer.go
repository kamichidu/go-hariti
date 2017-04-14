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

func (self *defaultState) Tag() string {
	return "default"
}

func (self *defaultState) Whitespace() uint64 {
	return scanner.GoWhitespace
}

func (self *defaultState) IsIdentRune(ch rune, i int) bool {
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

func (self *bareStringState) Tag() string {
	return "bare-string"
}

func (self *bareStringState) Whitespace() uint64 {
	return scanner.GoWhitespace
}

func (self *bareStringState) IsIdentRune(ch rune, i int) bool {
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

func (self *barePathStringState) Tag() string {
	return "bare-path-string"
}
func (self *barePathStringState) Whitespace() uint64 {
	return scanner.GoWhitespace
}

func (self *barePathStringState) IsIdentRune(ch rune, i int) bool {
	switch ch {
	case '\\':
		self.escape = true
		return true
	case ' ', '\r', '\n':
		escaped := self.escape
		self.escape = false
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

func (self *rawScriptState) Tag() string {
	return "raw-script"
}

func (self *rawScriptState) Whitespace() uint64 {
	return 1<<'\r' | 1<<' '
}

func (self *rawScriptState) IsIdentRune(ch rune, i int) bool {
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

func (self *Lexer) Scan() rune {
	tok := self.Scanner.Scan()

	switch self.stateTag() {
	case "default":
		switch self.TokenText() {
		case "use", "(":
			self.pushState("bare-string")
		case "-":
			self.pushState("raw-script")
		}
	case "bare-string":
		switch self.TokenText() {
		case "local":
			self.pushState("bare-path-string")
		case "as", "enable_if", "depends", "build": // following candidates for repository
			self.popState()
		case ")":
			self.popState()
		}
	case "bare-path-string":
		self.popState()
	case "raw-script":
		switch self.TokenText() {
		case "\n":
			self.popState()
		}
	}
	// for `use kamichidu` like syntax, in this case bare-string state always on the stack
	if tok == scanner.EOF && len(self.state) > 1 {
		self.popState()
	}

	return tok
}

func (self *Lexer) stateTag() string {
	return self.state[len(self.state)-1].Tag()
}

func (self *Lexer) pushState(tag string) {
	var state lexerState
	switch tag {
	case "default":
		state = &defaultState{
			Lexer: self,
		}
	case "bare-string":
		state = &bareStringState{
			Lexer: self,
		}
	case "bare-path-string":
		state = &barePathStringState{
			Lexer: self,
		}
	case "raw-script":
		state = &rawScriptState{
			Lexer: self,
		}
	default:
		log.Panicf("illegal state tag: %s\n", tag)
	}
	self.state = append(self.state, state)
	self.Whitespace = state.Whitespace()
	self.IsIdentRune = state.IsIdentRune
}

func (self *Lexer) popState() {
	self.state = self.state[:len(self.state)-1]
	if len(self.state) == 0 {
		log.Panicln("illegal state")
	}
	state := self.state[len(self.state)-1]
	self.Whitespace = state.Whitespace()
	self.IsIdentRune = state.IsIdentRune
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

func (self *Lexer) Lex(lval *yySymType) int {
	if debug {
		log.Printf("use state %#v", self.stateTag())
	}
	tok := self.Scan()
	if debug {
		log.Printf("\ttok = %#v, token = %#v\n", tok, self.TokenText())
	}
	if tok == EOF {
		lval.tok = Token{
			Id:   EOF,
			Text: "",
		}
		return EOF
	}
	text := self.TokenText()
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

func (self *Lexer) Error(msg string) {
	log.Printf("%s: %s\n", self.String(), msg)
}

var _ yyLexer = (*Lexer)(nil)
