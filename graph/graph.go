package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type SourceType string

const (
	SourceTypeRemote SourceType = "remote"
	SourceTypeLocal  SourceType = "local"
)

type Source struct {
	Type SourceType `json:"type"`
	URL  *url.URL   `json:"-"`
	Path string     `json:"path,omitempty"`
}

func (s Source) MarshalJSON() ([]byte, error) {
	var urlStr string
	if s.URL != nil {
		urlStr = s.URL.String()
	}
	return json.Marshal(&struct {
		Type SourceType `json:"type"`
		URL  string     `json:"url,omitempty"`
		Path string     `json:"path,omitempty"`
	}{
		Type: s.Type,
		URL:  urlStr,
		Path: s.Path,
	})
}

type BuildStep struct {
	OS  string `json:"os"`
	Cmd string `json:"cmd"`
}

type Bundle struct {
	ID           string      `json:"id"`
	Source       Source      `json:"source"`
	Dependencies []string    `json:"dependencies"`
	EnableIf     string      `json:"enable_if,omitempty"`
	Build        []BuildStep `json:"build"`
	Aliases      []string    `json:"aliases"`
}

func (b Bundle) GetName() string {
	return b.ID
}

func (b Bundle) GetLocalPath() string {
	return b.Source.Path
}

func (b Bundle) GetAliases() []string {
	return b.Aliases
}

type Graph struct {
	Bundles []Bundle `json:"bundles"`
}

func (g *Graph) Normalize() {
	if g.Bundles == nil {
		g.Bundles = make([]Bundle, 0)
	}

	for i := range g.Bundles {
		if g.Bundles[i].Dependencies == nil {
			g.Bundles[i].Dependencies = make([]string, 0)
		}
		if g.Bundles[i].Build == nil {
			g.Bundles[i].Build = make([]BuildStep, 0)
		}
		if g.Bundles[i].Aliases == nil {
			g.Bundles[i].Aliases = make([]string, 0)
		}

		for j := range g.Bundles[i].Build {
			if g.Bundles[i].Build[j].OS == "" {
				g.Bundles[i].Build[j].OS = "all"
			}
		}

		if g.Bundles[i].Source.Type == "" {
			if g.Bundles[i].Source.Path != "" {
				g.Bundles[i].Source.Type = SourceTypeLocal
			} else {
				g.Bundles[i].Source.Type = SourceTypeRemote
			}
		}
	}
}

func Validate(g Graph) error {
	ids := make(map[string]struct{})
	for _, b := range g.Bundles {
		if b.ID == "" {
			return errors.New("bundle ID cannot be empty")
		}
		if _, exists := ids[b.ID]; exists {
			return fmt.Errorf("duplicate bundle ID: %s", b.ID)
		}
		ids[b.ID] = struct{}{}

		if b.Source.Type != SourceTypeRemote && b.Source.Type != SourceTypeLocal {
			return fmt.Errorf("invalid source type for bundle %s: %s", b.ID, b.Source.Type)
		}

		if b.Source.Type == SourceTypeLocal && b.Source.Path == "" {
			return fmt.Errorf("local source path cannot be empty for bundle %s", b.ID)
		}

		for _, dep := range b.Dependencies {
			if dep == "" {
				return fmt.Errorf("bundle %s contains an empty dependency string", b.ID)
			}
		}
	}

	return nil
}
