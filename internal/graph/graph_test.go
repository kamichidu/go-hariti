package graph_test

import (
	"testing"

	"github.com/kamichidu/go-hariti/internal/graph"
)

func TestNormalize(t *testing.T) {
	g := &graph.Graph{}
	g.Normalize()

	if g.Bundles == nil || len(g.Bundles) != 0 {
		t.Error("expected non-nil empty Bundles slice")
	}

	g2 := &graph.Graph{
		Bundles: []graph.Bundle{
			{
				ID: "foo",
				Source: graph.Source{
					Path: "/path/to/foo",
				},
				Dependencies: nil,
				Build:        nil,
				Aliases:      nil,
			},
		},
	}
	g2.Normalize()

	b := g2.Bundles[0]
	if b.Dependencies == nil || len(b.Dependencies) != 0 {
		t.Error("expected non-nil empty Dependencies slice")
	}
	if b.Build == nil || len(b.Build) != 0 {
		t.Error("expected non-nil empty Build slice")
	}
	if b.Aliases == nil || len(b.Aliases) != 0 {
		t.Error("expected non-nil empty Aliases slice")
	}
	if b.Source.Type != graph.SourceTypeLocal {
		t.Errorf("expected SourceTypeLocal, got %s", b.Source.Type)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		graph   graph.Graph
		wantErr bool
	}{
		{
			name: "valid graph",
			graph: graph.Graph{
				Bundles: []graph.Bundle{
					{
						ID: "foo",
						Source: graph.Source{
							Type: graph.SourceTypeLocal,
							Path: "/some/path",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty bundle ID",
			graph: graph.Graph{
				Bundles: []graph.Bundle{
					{
						ID: "",
						Source: graph.Source{
							Type: graph.SourceTypeLocal,
							Path: "/some/path",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate bundle ID",
			graph: graph.Graph{
				Bundles: []graph.Bundle{
					{
						ID: "foo",
						Source: graph.Source{
							Type: graph.SourceTypeLocal,
							Path: "/some/path",
						},
					},
					{
						ID: "foo",
						Source: graph.Source{
							Type: graph.SourceTypeRemote,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid Source.Type",
			graph: graph.Graph{
				Bundles: []graph.Bundle{
					{
						ID: "foo",
						Source: graph.Source{
							Type: "invalid-type",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "local source without path",
			graph: graph.Graph{
				Bundles: []graph.Bundle{
					{
						ID: "foo",
						Source: graph.Source{
							Type: graph.SourceTypeLocal,
							Path: "",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty dependency name",
			graph: graph.Graph{
				Bundles: []graph.Bundle{
					{
						ID: "foo",
						Source: graph.Source{
							Type: graph.SourceTypeRemote,
						},
						Dependencies: []string{""},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := graph.Validate(tt.graph)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
