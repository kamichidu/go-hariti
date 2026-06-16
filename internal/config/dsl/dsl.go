package dsl

import (
	"io"

	"github.com/kamichidu/go-hariti/graph"
	"github.com/kamichidu/go-hariti/internal/config/dsl/ast"
	"github.com/kamichidu/go-hariti/internal/config/dsl/parser"
)

func Parse(filename string, src []byte) (*ast.File, error) {
	parsed, err := parser.Parse(filename, src)
	if err != nil {
		return nil, err
	}
	return parsed.(*ast.File), nil
}

func ParseReader(r io.Reader) (*ast.File, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Parse("", src)
}

func ParseGraph(filename string, src []byte) (*graph.Graph, error) {
	file, err := Parse(filename, src)
	if err != nil {
		return nil, err
	}
	return ToGraph(file)
}
