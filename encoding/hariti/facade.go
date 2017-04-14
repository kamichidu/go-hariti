package hariti

import (
	"errors"
	"io"
)

//go:generate goyacc -o parser.go -p yy parser.go.y
func Parse(src io.Reader) ([]Bundle, error) {
	lexer := NewLexer()
	lexer.Init(src)
	res := yyParse(lexer)
	if res != 0 {
		return nil, errors.New("Failed to parse")
	}
	return lexer.Bundles, nil
}
