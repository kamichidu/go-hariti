package graph

import (
	"net/url"
)

type SourceType string

const (
	SourceTypeRemote SourceType = "remote"
	SourceTypeLocal  SourceType = "local"
)

type Source struct {
	Type SourceType
	URL  *url.URL
	Path string
}

type BuildStep struct {
	OS  string
	Cmd string
}

type Bundle struct {
	ID           string
	Source       Source
	Dependencies []string
	EnableIf     string
	Build        []BuildStep
	Aliases      []string
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

type Replacement struct {
	From      string
	To        string
	LocalPath string
}

type Graph struct {
	Bundles  []Bundle
	Replaces []Replacement
}
