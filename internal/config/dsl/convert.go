package dsl

import (
	"github.com/kamichidu/go-hariti/internal/config/dsl/ast"
	"github.com/kamichidu/go-hariti/internal/graph"
)

func ToGraph(file *ast.File) (*graph.Graph, error) {
	g := &graph.Graph{
		Bundles:  make([]graph.Bundle, 0, len(file.Bundles)),
		Replaces: make([]graph.Replacement, 0),
	}

	for _, decl := range file.Bundles {
		var buildSteps []graph.BuildStep
		for _, bb := range decl.Build {
			osName := bb.OS
			if osName == "*" {
				osName = "all"
			}
			for _, cmd := range bb.Commands {
				buildSteps = append(buildSteps, graph.BuildStep{
					OS:  osName,
					Cmd: cmd,
				})
			}
		}

		b := graph.Bundle{
			ID: decl.Use,
			Source: graph.Source{
				Type: graph.SourceTypeRemote,
			},
			Dependencies: decl.Depends,
			EnableIf:     decl.EnableIf,
			Build:        buildSteps,
			Aliases:      decl.Aliases,
		}

		g.Bundles = append(g.Bundles, b)
	}

	g.Normalize()

	if err := graph.Validate(*g); err != nil {
		return nil, err
	}

	return g, nil
}
