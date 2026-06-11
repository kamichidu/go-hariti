package yaml

import (
	"github.com/kamichidu/go-hariti/internal/graph"
)

func (b *BundlesFile) ToGraph() (*graph.Graph, error) {
	g := &graph.Graph{
		Bundles:  make([]graph.Bundle, 0, len(b.Bundles)),
		Replaces: make([]graph.Replacement, 0),
	}
	for _, item := range b.Bundles {
		if item.Local != nil {
			g.Bundles = append(g.Bundles, graph.Bundle{
				ID: item.Local.Path,
				Source: graph.Source{
					Type: graph.SourceTypeLocal,
					Path: item.Local.Path,
				},
			})
		} else if item.Remote != nil {
			g.Bundles = append(g.Bundles, convertRemote(item.Remote))
		}
	}

	g.Normalize()

	if err := graph.Validate(*g); err != nil {
		return nil, err
	}

	return g, nil
}

func convertRemote(rb *RemoteBundle) graph.Bundle {
	var buildSteps []graph.BuildStep
	if rb.BuildScript != nil {
		if rb.BuildScript.Windows != "" {
			buildSteps = append(buildSteps, graph.BuildStep{OS: "windows", Cmd: rb.BuildScript.Windows})
		}
		if rb.BuildScript.Mac != "" {
			buildSteps = append(buildSteps, graph.BuildStep{OS: "mac", Cmd: rb.BuildScript.Mac})
		}
		if rb.BuildScript.Linux != "" {
			buildSteps = append(buildSteps, graph.BuildStep{OS: "linux", Cmd: rb.BuildScript.Linux})
		}
		if rb.BuildScript.All != "" {
			buildSteps = append(buildSteps, graph.BuildStep{OS: "all", Cmd: rb.BuildScript.All})
		}
	}

	var deps []string
	for _, dep := range rb.Dependencies {
		deps = append(deps, dep.Name)
	}

	return graph.Bundle{
		ID: rb.Name,
		Source: graph.Source{
			Type: graph.SourceTypeRemote,
			URL:  nil,
		},
		Dependencies: deps,
		EnableIf:     rb.EnableIfExpr,
		Build:        buildSteps,
		Aliases:      rb.Aliases,
	}
}
