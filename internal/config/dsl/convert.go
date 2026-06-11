package dsl

import (
	"fmt"

	"github.com/kamichidu/go-hariti/internal/config/dsl/ast"
	"github.com/kamichidu/go-hariti/internal/graph"
)

func ToGraph(file *ast.File) (*graph.Graph, error) {
	bundlesMap := make(map[string]graph.Bundle)
	bundlesOrder := make([]string, 0, len(file.Bundles))

	// 1. Initial list of bundles from ast.Bundles
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

		// Resolve source path
		sourceExpr := decl.Use
		if decl.Source != nil {
			sourceExpr = *decl.Source
		}
		src, err := ResolveSource(sourceExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source for bundle %s: %w", decl.Use, err)
		}

		enableIfVal := ""
		if decl.EnableIf != nil {
			enableIfVal = *decl.EnableIf
		}

		b := graph.Bundle{
			ID:           decl.Use,
			Source:       src,
			Dependencies: decl.Depends,
			EnableIf:     enableIfVal,
			Build:        buildSteps,
			Aliases:      decl.Aliases,
		}

		bundlesMap[b.ID] = b
		bundlesOrder = append(bundlesOrder, b.ID)
	}

	// 2. Apply Replaces
	for _, rep := range file.Replaces {
		targetID := rep.Target
		_, exists := bundlesMap[targetID]
		if !exists {
			return nil, fmt.Errorf("replace target %s does not exist in the compile graph", targetID)
		}

		var buildSteps []graph.BuildStep
		if rep.Bundle.Build != nil {
			for _, bb := range *rep.Bundle.Build {
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
		}

		sourceExpr := targetID
		if rep.Bundle.Source != nil {
			sourceExpr = *rep.Bundle.Source
		}
		src, err := ResolveSource(sourceExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source in replace %s: %w", targetID, err)
		}

		deps := []string{}
		if rep.Bundle.Depends != nil {
			deps = *rep.Bundle.Depends
		}

		enableIfVal := ""
		if rep.Bundle.EnableIf != nil {
			enableIfVal = *rep.Bundle.EnableIf
		}

		replaced := graph.Bundle{
			ID:           targetID, // preserve identity
			Source:       src,
			Dependencies: deps,
			EnableIf:     enableIfVal,
			Build:        buildSteps,
			Aliases:      rep.Bundle.Aliases,
		}

		bundlesMap[targetID] = replaced
	}

	// 3. Apply Merges
	for _, m := range file.Merges {
		targetID := m.Target
		orig, exists := bundlesMap[targetID]
		if !exists {
			return nil, fmt.Errorf("merge target %s does not exist in the compile graph", targetID)
		}

		merged := orig

		if m.Patch.Source != nil {
			src, err := ResolveSource(*m.Patch.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve source in merge %s: %w", targetID, err)
			}
			merged.Source = src
		}

		if len(m.Patch.Aliases) > 0 {
			merged.Aliases = append(merged.Aliases, m.Patch.Aliases...)
		}

		if m.Patch.Depends != nil {
			merged.Dependencies = *m.Patch.Depends
		}

		if m.Patch.EnableIf != nil {
			merged.EnableIf = *m.Patch.EnableIf
		}

		if m.Patch.Build != nil {
			var buildSteps []graph.BuildStep
			for _, bb := range *m.Patch.Build {
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
			merged.Build = buildSteps
		}

		bundlesMap[targetID] = merged
	}

	// 4. Construct final graph.Graph containing resolved bundles in order
	g := &graph.Graph{
		Bundles: make([]graph.Bundle, 0, len(bundlesOrder)),
	}
	for _, id := range bundlesOrder {
		g.Bundles = append(g.Bundles, bundlesMap[id])
	}

	g.Normalize()

	if err := graph.Validate(*g); err != nil {
		return nil, err
	}

	return g, nil
}
