package ast

type File struct {
	Bundles []BundleDecl
}

type BundleDecl struct {
	Use      string
	Aliases  []string
	Depends  []string
	EnableIf string
	Build    []BuildBlock
}

type BuildBlock struct {
	OS       string
	Commands []string
}
