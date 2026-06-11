package ast

type File struct {
	Includes []IncludeDecl
	Bundles  []BundleDecl
	Replaces []ReplaceDecl
	Merges   []MergeDecl
}

type BundleDecl struct {
	Use      string
	Source   *string
	Aliases  []string
	Depends  []string
	EnableIf *string
	Build    []BuildBlock
}

type BuildBlock struct {
	OS       string
	Commands []string
}

type IncludeDecl struct {
	Path string
}

type ReplaceDecl struct {
	Target string
	Bundle BundlePatch
}

type MergeDecl struct {
	Target string
	Patch  BundlePatch
}

type BundlePatch struct {
	Source   *string
	Aliases  []string
	Depends  *[]string
	EnableIf *string
	Build    *[]BuildBlock
}
